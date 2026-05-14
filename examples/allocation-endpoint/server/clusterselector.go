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

// Package main contains the cluster selection logic.
package main

import (
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// A ClusterInfo contains the game server cluster info.
type ClusterInfo struct {
	Name             string `json:"name"`
	Endpoint         string `json:"endpoint"`
	Namespace        string `json:"namespace"`
	AllocationWeight int    `json:"allocation_weight"`
	AllocationRate   float64
}

// ClusterConns is a connection pool of a cluster.
type ClusterConns struct {
	conns      []*grpc.ClientConn
	createdTSs []time.Time
	m          sync.RWMutex
}

// NewClusterConns returns a new ClusterConns with a predefined list of the number of connections.
func NewClusterConns(numOfConnections int) *ClusterConns {
	return &ClusterConns{
		conns:      make([]*grpc.ClientConn, numOfConnections),
		createdTSs: make([]time.Time, numOfConnections),
		m:          sync.RWMutex{},
	}
}

// Get returns the grpc connection, the connection created time of the index position in the connection pool list and true if exists, otherwise, return false
func (cc *ClusterConns) Get(index int) (*grpc.ClientConn, time.Time, bool) {
	cc.m.RLock()
	defer cc.m.RUnlock()
	if len(cc.conns) <= index || len(cc.createdTSs) <= index {
		return nil, time.Time{}, false
	}
	return cc.conns[index], cc.createdTSs[index], true
}

// Set sets the grpc connection and created time in the connection pool list by the index given if the connection list is longer than the index given, otherwise, return errors.
func (cc *ClusterConns) Set(index int, conn *grpc.ClientConn, time time.Time) error {
	cc.m.Lock()
	defer cc.m.Unlock()
	if len(cc.conns) <= index || len(cc.createdTSs) <= index {
		return errors.Errorf("current connections are less than %d", index)
	}
	cc.conns[index] = conn
	cc.createdTSs[index] = time
	return nil
}

// selectCluster selects cluster based on cluster allocation weight
func selectCluster(clustersInfo []*ClusterInfo, allocatedClusters map[string]bool) *ClusterInfo {
	var availableClusters []*ClusterInfo
	var aggregatedRates []float64
	cur := float64(0)

	for _, cluster := range clustersInfo {
		if _, ok := allocatedClusters[cluster.Name]; !ok && cluster.AllocationWeight != 0 {
			cur += cluster.AllocationRate
			aggregatedRates = append(aggregatedRates, cur)
			availableClusters = append(availableClusters, cluster)
		}
	}

	if len(aggregatedRates) == 0 {
		return nil
	}
	r := rand.Float64() * cur
	prev := float64(0)

	for i, aggregatedRate := range aggregatedRates {
		if r > prev && r <= aggregatedRate {
			return availableClusters[i]
		}
		prev = aggregatedRate
	}
	return nil
}
