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

package v1

import (
	"math/rand"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GameServerAllocationPolicySpec defines the desired state of GameServerAllocationPolicy
type GameServerAllocationPolicySpec struct {
	// +kubebuilder:validation:Minimum=0
	Priority int32 `json:"priority"`
	// +kubebuilder:validation:Minimum=0
	Weight         int                   `json:"weight"`
	ConnectionInfo ClusterConnectionInfo `json:"connectionInfo,omitempty"`
}

// ClusterConnectionInfo defines the connection information for a cluster
type ClusterConnectionInfo struct {
	// Optional: the name of the targeted cluster
	ClusterName string `json:"clusterName"`
	// The endpoints for the allocator service in the targeted cluster.
	// If the AllocationEndpoints is not set, the allocation happens on local cluster.
	// If there are multiple endpoints any of the endpoints that can handle allocation request should suffice
	AllocationEndpoints []string `json:"allocationEndpoints,omitempty"`
	// The name of the secret that contains TLS client certificates to connect the allocator server in the targeted cluster
	SecretName string `json:"secretName"`
	// The cluster namespace from which to allocate gameservers
	Namespace string `json:"namespace"`
	// The PEM encoded server CA, used by the allocator client to authenticate the remote server.
	ServerCA []byte `json:"serverCa,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerAllocationPolicy is the Schema for the gameserverallocationpolicies API
// +k8s:openapi-gen=true
type GameServerAllocationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec GameServerAllocationPolicySpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerAllocationPolicyList contains a list of GameServerAllocationPolicy
type GameServerAllocationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServerAllocationPolicy `json:"items"`
}

// ConnectionInfoIterator an iterator on ClusterConnectionInfo
type ConnectionInfoIterator struct {
	// currPriority Current priority index from the orderedPriorities
	currPriority int
	// orderedPriorities list of ordered priorities
	orderedPriorities []int32
	// priorityToCluster Map of priority to cluster-policies map
	priorityToCluster map[int32]map[string][]*GameServerAllocationPolicy
	// clusterBlackList the cluster blacklist for the clusters that has already returned
	clusterBlackList map[string]bool
}

// Next returns the next ClusterConnectionInfo value if available or nil if iterator reaches the end.
func (it *ConnectionInfoIterator) Next() *ClusterConnectionInfo {
	for it.currPriority < len(it.orderedPriorities) {
		// Get clusters with the highest priority
		currPriority := it.orderedPriorities[it.currPriority]
		clusterPolicy := it.priorityToCluster[currPriority]

		if result := it.getClusterConnectionInfo(clusterPolicy); result == nil {
			// If there is no cluster with the current priority, choose cluster with next highest priority
			it.currPriority++
		} else {
			// To avoid the same cluster again add that to a black list
			it.clusterBlackList[result.ClusterName] = true
			return result
		}
	}

	return nil
}

// NewConnectionInfoIterator creates an iterator for connection info
func NewConnectionInfoIterator(policies []*GameServerAllocationPolicy) *ConnectionInfoIterator {
	priorityToCluster := make(map[int32]map[string][]*GameServerAllocationPolicy)
	for _, policy := range policies {
		priority := policy.Spec.Priority
		clusterName := policy.Spec.ConnectionInfo.ClusterName

		// 1. Add priorities to the map of priority to cluster-priorities map
		clusterPolicy, ok := priorityToCluster[priority]
		if !ok {
			clusterPolicy = make(map[string][]*GameServerAllocationPolicy)
			priorityToCluster[priority] = clusterPolicy
		}

		// 2. Add cluster to the cluster-priorities map
		if _, ok := clusterPolicy[clusterName]; !ok {
			clusterPolicy[clusterName] = []*GameServerAllocationPolicy{policy}
		} else {
			clusterPolicy[clusterName] = append(clusterPolicy[clusterName], policy)
		}
	}

	// 3. Sort priorities
	priorities := make([]int32, 0, len(priorityToCluster))
	for k := range priorityToCluster {
		priorities = append(priorities, k)
	}
	sort.Slice(priorities, func(i, j int) bool { return priorities[i] < priorities[j] })

	// 4. Store initial values for the iterator
	return &ConnectionInfoIterator{priorityToCluster: priorityToCluster, currPriority: 0, orderedPriorities: priorities, clusterBlackList: make(map[string]bool)}
}

// getClusterConnectionInfo returns a ClusterConnectionInfo selected base on weighted randomization.
func (it *ConnectionInfoIterator) getClusterConnectionInfo(clusterPolicy map[string][]*GameServerAllocationPolicy) *ClusterConnectionInfo {
	connections := []*ClusterConnectionInfo{}
	weights := []int{}
	for cluster, policies := range clusterPolicy {
		if _, ok := it.clusterBlackList[cluster]; ok {
			continue
		}
		weights = append(weights, avgWeight(policies))
		connections = append(connections, &policies[0].Spec.ConnectionInfo)
	}

	if len(connections) == 0 {
		return nil
	}

	return selectRandomWeighted(connections, weights)
}

// avgWeight calculates average over allocation policy Weight field.
func avgWeight(policies []*GameServerAllocationPolicy) int {
	if len(policies) == 0 {
		return 0
	}
	var sum int
	for _, policy := range policies {
		sum += policy.Spec.Weight
	}
	return sum / len(policies)
}

// selectRandomWeighted selects a ClusterConnectionInfo info from a weighted list of ClusterConnectionInfo
func selectRandomWeighted(connections []*ClusterConnectionInfo, weights []int) *ClusterConnectionInfo {
	sum := 0
	for _, weight := range weights {
		sum += weight
	}

	if sum <= 0 {
		return nil
	}

	randValue := rand.Intn(sum)
	sum = 0
	for i, weight := range weights {
		sum += weight
		if randValue < sum {
			return connections[i]
		}
	}
	return nil
}
