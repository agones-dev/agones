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

// Package main runs an Allocation Endpoint client with configurations for benchmarking
package main

import (
	"context"
	"crypto/tls"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const port = "443"

var (
	clientCount    = flag.Int("clientCount", 1, "Number of clients to perform allocations")
	perClientCalls = flag.Int("perClientCalls", 1, "Per Client allocations running in parallel")
	runs           = flag.Int("runs", 1, "The number of times the experiment is run and averaged across")
	serverAddr     = flag.String("url", "", "The Cloud Run URL for allocation endpoint.")
	logger         = log.New(os.Stdout, "", 0)
	muErr          sync.Mutex // guards error variable update
	muSucc         sync.Mutex // guards success variable update
)

//go:embed sa_key.json
var saKey string

func main() {
	flag.Parse()

	logger.Printf("Running %d times averaged across %d allocation calls per %d clients\n", *runs, *perClientCalls, *clientCount)

	var errorCalls, successCalls int64
	var successLatency float64
	errorGroup := make(map[string]int64)
	startRun := time.Now()
	waitBetweenRunsInSec := time.Second * 3

	audience := fmt.Sprintf("https://%s", *serverAddr)
	ts, err := idtoken.NewTokenSource(context.Background(), audience, idtoken.WithCredentialsJSON([]byte(saKey)))
	if err != nil {
		logger.Fatal(err)
	}

	var wg sync.WaitGroup
	for k := 0; k < *clientCount; k++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			cred := credentials.NewTLS(&tls.Config{})
			opts := []grpc.DialOption{grpc.WithTransportCredentials(cred)}

			conn, err := grpc.Dial(fmt.Sprintf("%s:%s", *serverAddr, port), opts...)
			if err != nil {
				logger.Printf("Failed to dial for client %v: %v", k, err)
				return
			}

			client := pb.NewAllocationServiceClient(conn)

			// Repeat the same experiment for # of runs
			for r := 0; r < *runs; r++ {
				var wgInner sync.WaitGroup

				for i := 0; i < *perClientCalls; i++ {
					wgInner.Add(1)
					go func(i int) {
						defer wgInner.Done()

						start := time.Now()
						if err := allocate(client, ts); err != nil {

							muErr.Lock()
							errorCalls++
							errorGroup[status.Code(err).String()] = errorGroup[status.Code(err).String()] + 1
							muErr.Unlock()

							logger.Printf("Error received: %v", err)
						} else {

							muSucc.Lock()
							successCalls++
							successLatency += time.Now().Sub(start).Seconds()
							muSucc.Unlock()
						}
					}(i)
				}
				wgInner.Wait()
				time.Sleep(waitBetweenRunsInSec)
			}
			conn.Close()
		}(k)
	}
	wg.Wait()

	runDuration := time.Now().Sub(startRun).Seconds() - float64(*runs)*waitBetweenRunsInSec.Seconds() // except all the sleeps
	report(errorCalls, successCalls, runDuration, successLatency, errorGroup)
}

// Print out latency, RPC and error rates based on its category
func report(errorCalls, successCalls int64, runDuration, successLatency float64, errorGroup map[string]int64) {
	totalCalls := errorCalls + successCalls
	rpc := float64(successCalls) / runDuration
	var latency float64
	if successCalls > 0 {
		latency = successLatency / float64(successCalls)
	}

	logger.Printf("call count ----------------------- %d\n", totalCalls)
	logger.Printf("run duration in s ---------------- %f\n", runDuration)
	logger.Printf("successful call count ------------ %d\n", successCalls)
	logger.Printf("successful call latency in s ----- %f\n", latency)
	logger.Printf("successful request per second ---- %f\n", rpc)
	logger.Printf("error call count ----------------- %d\n", errorCalls)
	logger.Printf("error Cartegories:")
	for k, v := range errorGroup {
		logger.Printf("  %s - %d\n", k, v)
	}
}

func allocate(client pb.AllocationServiceClient, ts oauth2.TokenSource) error {
	t, err := ts.Token()
	if err != nil {
		logger.Fatal(err)
	}
	token := fmt.Sprintf("Bearer %s", t.AccessToken)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	ctx = metadata.AppendToOutgoingContext(ctx, "Authorization", token)

	defer cancel()
	resp, err := client.Allocate(ctx, &pb.AllocationRequest{})
	if err != nil {
		return err
	}
	logger.Printf("allocated game server: %v\n", *resp)
	return nil
}
