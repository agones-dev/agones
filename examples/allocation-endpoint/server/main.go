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

// Binary that implements Allocation Endpoint Proxy.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	numOfConnections      = 12 // This indicates the number of connections AP will open and cache for each Agones Cluster
	minAllocationWeight   = 0
	maxAllocationWeight   = 100
	failureTimes          = 300
	failureDuration       = 10 * time.Second
	exhaustedDuration     = 10 * time.Second
	refreshDuration       = 60 * time.Second
	allocationTimeout     = 20 * time.Second
	connectionTimeout     = 5 * time.Second
	connectionPoolTimeout = 60 * time.Minute
)

var (
	port                 = "443"
	clustersInfo         []*ClusterInfo
	clustersConnections  = make(map[string]*ClusterConns)
	mClustersConnections = sync.RWMutex{}
	failedClusters       = sync.Map{}
	ctx                  = context.Background()
	cred                 = credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	tokenSource          oauth2.TokenSource
)

type allocationEndpointService struct {
	pb.AllocationServiceServer
}

type failedConnectCluster struct {
	failedTime time.Time
	count      int
}

func main() {
	log.Info("Starting allocation proxy server...")

	readEnvVariables()

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}
	opts := []grpc.ServerOption{
		grpc.StatsHandler(&ocgrpc.ServerHandler{}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Timeout:           10 * time.Minute,
		}),
	}

	grpcServer := grpc.NewServer(opts...)

	pb.RegisterAllocationServiceServer(grpcServer, &allocationEndpointService{})
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed ro start Allocation Endpoint Proxy Server:%v", err)
	}
}

func readEnvVariables() {
	// Read all environment variables using
	// Environ which returns a copy of strings
	// representing the environment,
	// in the form "key=value".
	envVars := map[string]string{}
	for _, env := range os.Environ() {
		envPair := strings.SplitN(env, "=", 2)
		key := envPair[0]
		value := envPair[1]
		envVars[key] = value
	}

	var err error
	if portVal, ok := envVars["PORT"]; ok {
		port = portVal
	}

	clustersInfoStr := envVars["CLUSTERS_INFO"]
	if clustersInfo, err = extractClusterInfo(clustersInfoStr); err != nil {
		log.Fatalf("Error parsing CLUSTERS_INFO=\"%v\": %v", clustersInfoStr, err)
	}

	audience := envVars["AUDIENCE"]
	if audience == "" {
		log.Fatal("The AUDIENCE for connecting to the backend is missing from the environment variables.")
	}

	saKey := envVars["SA_KEY"]
	if saKey == "" {
		log.Fatal("The SA_KEY for for connecting to the backend is missing.")
	}

	// With a global TokenSource tokens would be reused and auto-refreshed at need.
	if tokenSource, err = google.JWTAccessTokenSourceFromJSON([]byte(saKey), audience); err != nil {
		log.Fatalf("Error building JWT access token source: %v", err)
	}
}

func extractClusterInfo(clustersInfoStr string) ([]*ClusterInfo, error) {
	clustersInfo := []*ClusterInfo{}

	if err := json.Unmarshal([]byte(clustersInfoStr), &clustersInfo); err != nil {
		return nil, err
	}
	allocationRate(clustersInfo)
	return clustersInfo, nil
}

// Calculate the allocation rate on clusters
func allocationRate(clustersInfo []*ClusterInfo) {
	numEnabledClusters := 0
	numRateControlledClusters := 0
	totalWeight := 0
	for _, clusterInfo := range clustersInfo {
		if clusterInfo.AllocationWeight != minAllocationWeight {
			numEnabledClusters++
			totalWeight += clusterInfo.AllocationWeight
			if clusterInfo.AllocationWeight != maxAllocationWeight {
				numRateControlledClusters++
			}
		}
	}
	// All clusters are rate controlled
	if numEnabledClusters == numRateControlledClusters {
		for _, clusterInfo := range clustersInfo {
			clusterInfo.AllocationRate = float64(clusterInfo.AllocationWeight) / float64(totalWeight)
		}
		return
	}
	// Some clusters are not rate controlled thus need to take more traffic from the rate controlled ones
	averageRate := float64(1) / float64(numEnabledClusters)
	existingRate := float64(0)
	for _, clusterInfo := range clustersInfo {
		if clusterInfo.AllocationWeight != 100 {
			clusterInfo.AllocationRate = averageRate * float64(clusterInfo.AllocationWeight) / float64(100)
			existingRate += clusterInfo.AllocationRate
		}
	}
	for _, clusterInfo := range clustersInfo {
		if clusterInfo.AllocationWeight == 100 {
			clusterInfo.AllocationRate = (float64(1) - existingRate) / float64(numEnabledClusters-numRateControlledClusters)
		}
	}
}

func getConnection(ctx context.Context, clusterInfo *ClusterInfo) (*grpc.ClientConn, int, error) {
	mClustersConnections.RLock()
	clusterConnection, ok := clustersConnections[clusterInfo.Name]
	mClustersConnections.RUnlock()
	if !ok {
		clusterConnection = NewClusterConns(numOfConnections)
		mClustersConnections.Lock()
		clustersConnections[clusterInfo.Name] = clusterConnection
		mClustersConnections.Unlock()
	}
	perm := rand.Perm(numOfConnections)
	for _, randIndex := range perm {
		currentConn, createdTime, _ := clusterConnection.Get(randIndex)
		if currentConn == nil || time.Since(createdTime) > connectionPoolTimeout {
			conn, err := connectToAgonesCluster(ctx, clusterInfo)
			if err != nil {
				log.Error(err)
				continue
			}
			clusterConnection.Set(randIndex, conn, time.Now())
			return conn, randIndex, nil
		}
		return currentConn, randIndex, nil
	}

	return nil, -1, errors.Errorf("could not connect to %s with endpoint %s", clusterInfo.Name, clusterInfo.Endpoint)
}

func (s *allocationEndpointService) Allocate(ctx context.Context, req *pb.AllocationRequest) (*pb.AllocationResponse, error) {
	allocatedClusters := make(map[string]bool)
	var err error
	var resp *pb.AllocationResponse

	ctx, cancel := context.WithTimeout(ctx, allocationTimeout)
	defer cancel()
	for i := 0; i < len(clustersInfo); i++ {
		select {
		case <-ctx.Done():
			return nil, status.Error(codes.DeadlineExceeded, err.Error())
		default:
		}

		selectedClstr := selectCluster(clustersInfo, allocatedClusters)
		if selectedClstr == nil {
			return nil, status.Errorf(codes.NotFound, "No active cluster. Check your clusters connections with last error: %v", err)
		}
		allocatedClusters[selectedClstr.Name] = true

		// Skip the cluster if connection failed more than failure times within failure duration of the first failure.
		if failedCluster, ok := failedClusters.Load(selectedClstr.Name); ok &&
			time.Since(failedCluster.(failedConnectCluster).failedTime) < failureDuration &&
			failedCluster.(failedConnectCluster).count >= failureTimes {
			continue
		}

		log.Infof("sending allocation request to %s", selectedClstr.Name)

		resp, err = allocateFromAgones(ctx, selectedClstr, req)
		if err != nil {
			log.Warningf("allocation request to Agones failed: %v", err)
			go updateFailed(selectedClstr.Name, err)
			continue
		}
		return resp, nil
	}
	return nil, err
}

func updateFailed(clusterName string, err error) {
	// Mark all game servers in the cluster are allocated for now.
	failedClusterInterface, ok := failedClusters.Load(clusterName)
	if ok && time.Since(failedClusterInterface.(failedConnectCluster).failedTime) < failureDuration {
		failedCluster := failedClusterInterface.(failedConnectCluster)
		failedCluster.count++
		failedClusters.Store(clusterName, failedCluster)
	} else {
		failedCluster := failedConnectCluster{failedTime: time.Now(), count: 1}
		failedClusters.Store(clusterName, failedCluster)
	}
}

func connectToAgonesCluster(ctx context.Context, clusterInfo *ClusterInfo) (*grpc.ClientConn, error) {
	// nolint: staticcheck	
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:443", clusterInfo.Endpoint), grpc.WithTransportCredentials(cred))
	if err != nil {
		return nil, errors.Wrapf(err, "could not connect to %s with endpoint %s", clusterInfo.Name, clusterInfo.Endpoint)
	}
	return conn, nil
}

func allocateFromAgones(ctx context.Context, clusterInfo *ClusterInfo, req *pb.AllocationRequest) (*pb.AllocationResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	conn, index, err := getConnection(ctx, clusterInfo)
	if err != nil {
		return nil, err
	}

	req.Namespace = clusterInfo.Namespace

	// append access tokens
	tk, err := tokenSource.Token()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("could not retrieve token to connect the backend: %v", err))
	}
	ctx = metadata.AppendToOutgoingContext(ctx, "Authorization", fmt.Sprintf("Bearer %s", tk.AccessToken))

	client := pb.NewAllocationServiceClient(conn)

	resp, err := client.Allocate(ctx, req)
	if err != nil {
		log.Infof("Error while executing Sending Allocation request to Agones: %v", err)
		if status.Code(err) != codes.ResourceExhausted {
			mClustersConnections.RLock()
			clustersConnection := clustersConnections[clusterInfo.Name]
			mClustersConnections.RUnlock()
			clustersConnection.Set(index, nil, time.Time{})
		}
		return nil, err
	}

	return resp, err
}
