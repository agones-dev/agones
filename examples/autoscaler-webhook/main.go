// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Autoscaler webhook server which handles FleetAutoscaleReview json payload
package main

import (
	"encoding/json"
	"flag"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"

	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/util/runtime" // for the logger
)

// Parameters which define thresholds to trigger scalling up and scale factor
var (
	replicaUpperThreshold        = 0.7
	replicaLowerThreshold        = 0.3
	scaleFactor                  = 2.
	minReplicasCount             = int32(2)
	fixedReplicasOverrideEnabled bool
)

// Variables for the logger
var (
	logger = runtime.NewLoggerWithSource("main")
)

// Get all parameters from ENV variables
// Extra check is performed not to fall into the infinite loop:
// replicaDownTrigger < replicaUpperThreshold/scaleFactor
func getEnvVariables() {
	if ep := os.Getenv("SCALE_FACTOR"); ep != "" {
		factor, err := strconv.ParseFloat(ep, 64)
		if err != nil {
			logger.WithError(err).Fatal("Could not parse environment SCALE_FACTOR variable")
		} else if factor > 1 {
			scaleFactor = factor
		}
	}

	if ep := os.Getenv("REPLICA_UPSCALE_TRIGGER"); ep != "" {
		replicaUpTrigger, err := strconv.ParseFloat(ep, 64)
		if err != nil {
			logger.WithError(err).Fatal("Could not parse environment REPLICA_UPSCALE_TRIGGER variable")
		} else if replicaUpTrigger > 0.1 {
			replicaUpperThreshold = replicaUpTrigger
		}
	}

	if ep := os.Getenv("REPLICA_DOWNSCALE_TRIGGER"); ep != "" {
		replicaDownTrigger, err := strconv.ParseFloat(ep, 64)
		if err != nil {
			logger.WithError(err).Fatal("Could not parse environment REPLICA_DOWNSCALE_TRIGGER variable")
		} else if replicaDownTrigger < replicaUpperThreshold/scaleFactor {
			replicaLowerThreshold = replicaDownTrigger
		}
	}

	if ep := os.Getenv("MIN_REPLICAS_COUNT"); ep != "" {
		minReplicas, err := strconv.ParseInt(ep, 10, 32)
		if err != nil {
			logger.WithError(err).Fatal("Could not parse environment MIN_REPLICAS_COUNT variable")
		} else if minReplicas >= 0 {
			minReplicasCount = int32(minReplicas)
		}
	}

	if ep := os.Getenv("FIXED_REPLICAS"); ep != "" {
		if ep == "true" {
			fixedReplicasOverrideEnabled = true
			logger.Info("FIXED_REPLICAS override is enabled")
		} else {
			fixedReplicasOverrideEnabled = false
			logger.Info("FIXED_REPLICAS override is disabled")
		}
	}

	// Extra check: In order not to fall into infinite loop
	// we change down scale trigger, so that after we scale up
	// fleet does not immediately scales down and vice versa
	if replicaLowerThreshold >= replicaUpperThreshold/scaleFactor {
		replicaLowerThreshold = replicaUpperThreshold / (scaleFactor + 1)
	}
}

// Main will set up an http server and three endpoints
func main() {
	port := flag.String("port", "8000", "The port to listen to TCP requests")
	flag.Parse()
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	getEnvVariables()
	// Run the HTTP server using the bound certificate and key for TLS
	// Serve 200 status on /health for k8s health checks
	http.HandleFunc("/health", handleHealth)

	// Return the target replica count which is used by Webhook fleet autoscaling policy
	http.HandleFunc("/scale", handleAutoscale)

	_, err := os.Stat("/home/service/certs/tls.crt")
	if err == nil {
		logger.Info("Starting HTTPS server on port ", *port)
		if err := http.ListenAndServeTLS(":"+*port, "/home/service/certs/tls.crt", "/home/service/certs/tls.key", nil); err != nil {
			logger.WithError(err).Fatal("HTTPS server failed to run")
		}
	} else {
		logger.Info("Starting HTTP server on port ", *port)
		if err := http.ListenAndServe(":"+*port, nil); err != nil {
			logger.WithError(err).Fatal("HTTP server failed to run")
		}
	}
}

// Let /health return Healthy and status code 200
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := io.WriteString(w, "Healthy")
	if err != nil {
		logger.WithError(err).Fatal("Error writing string Healthy from /health")
	}
}

// handleAutoscale is a handler function which return the replica count
// based on received status of the fleet
func handleAutoscale(w http.ResponseWriter, r *http.Request) {
	if r == nil {
		http.Error(w, "Empty request", http.StatusInternalServerError)
		return
	}

	var faReq autoscalingv1.FleetAutoscaleReview
	res, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	err = json.Unmarshal(res, &faReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	faResp := autoscalingv1.FleetAutoscaleResponse{
		Scale:    false,
		Replicas: faReq.Request.Status.Replicas,
		UID:      faReq.Request.UID,
	}

	if fixedReplicasOverrideEnabled {
		if faReq.Request.Annotations != nil {
			if value, ok := faReq.Request.Annotations["fixedReplicas"]; ok {
				replicas, err := strconv.Atoi(value)
				if err != nil {
					logger.WithError(err).Error("Invalid fixedReplicas value in annotations")
					http.Error(w, "Invalid fixedReplicas value. Must be an integer.", http.StatusBadRequest)
					return
				}

				if replicas < 0 {
					logger.Error("fixedReplicas value cannot be negative")
					http.Error(w, "Invalid fixedReplicas value. Must be >= 0.", http.StatusBadRequest)
					return
				}

				faResp.Scale = true
				faResp.Replicas = int32(replicas)
			}
		}
	} else if faReq.Request.Status.Replicas != 0 {
		allocatedPercent := float64(faReq.Request.Status.AllocatedReplicas) / float64(faReq.Request.Status.Replicas)
		if allocatedPercent > replicaUpperThreshold {
			// After scaling we would have percentage of 0.7/2 = 0.35 > replicaLowerThreshold
			// So we won't scale down immediately after scale up
			currentReplicas := float64(faReq.Request.Status.Replicas)
			faResp.Scale = true
			faResp.Replicas = int32(math.Ceil(currentReplicas * scaleFactor))
		} else if allocatedPercent < replicaLowerThreshold && faReq.Request.Status.Replicas > minReplicasCount {
			faResp.Scale = true
			faResp.Replicas = int32(math.Ceil(float64(faReq.Request.Status.Replicas) / scaleFactor))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	review := &autoscalingv1.FleetAutoscaleReview{
		Request:  faReq.Request,
		Response: &faResp,
	}
	logger.WithField("review", review).Info("FleetAutoscaleReview")
	result, _ := json.Marshal(&review)

	_, err = io.WriteString(w, string(result))
	if err != nil {
		logger.WithError(err).Fatal("Error writing json from /scale")
	}
}
