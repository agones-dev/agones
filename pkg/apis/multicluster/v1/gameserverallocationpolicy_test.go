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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectionInfoIterator(t *testing.T) {
	testCases := []struct {
		name      string
		in        []*GameServerAllocationPolicy
		want      []ClusterConnectionInfo
		unordered bool
	}{
		{
			name: "Simple test",
			in: []*GameServerAllocationPolicy{
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   100,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName:         "cluster1",
							SecretName:          "secret-name",
							AllocationEndpoints: []string{"allocation-endpoint"},
							Namespace:           "ns1",
							ServerCA:            []byte("c2VydmVyQ0E="),
						},
					},
				},
			},
			want: []ClusterConnectionInfo{
				{
					ClusterName:         "cluster1",
					SecretName:          "secret-name",
					AllocationEndpoints: []string{"allocation-endpoint"},
					Namespace:           "ns1",
					ServerCA:            []byte("c2VydmVyQ0E="),
				},
			},
		},
		{
			name: "Different priorities and weight same cluster",
			in: []*GameServerAllocationPolicy{
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   100,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster-name",
						},
					},
				},
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 2,
						Weight:   300,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster-name",
						},
					},
				},
			},
			want: []ClusterConnectionInfo{
				{
					ClusterName: "cluster-name",
				},
			},
		},
		{
			name: "Different clusters same priority",
			in: []*GameServerAllocationPolicy{
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   300,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster1",
						},
					},
				},
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   100,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster2",
						},
					},
				},
			},
			want: []ClusterConnectionInfo{
				{
					ClusterName: "cluster1",
				},
				{
					ClusterName: "cluster2",
				},
			},
			unordered: true,
		},
		{
			name: "Different clusters different priorities",
			in: []*GameServerAllocationPolicy{
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   300,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster1",
						},
					},
				},
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 2,
						Weight:   100,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster2",
						},
					},
				},
			},
			want: []ClusterConnectionInfo{
				{
					ClusterName: "cluster1",
				},
				{
					ClusterName: "cluster2",
				},
			},
		},
		{
			name: "Different clusters repeated with different priorities",
			in: []*GameServerAllocationPolicy{
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   300,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster1",
						},
					},
				},
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   100,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster2",
						},
					},
				},
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 2,
						Weight:   300,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster1",
						},
					},
				},
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 2,
						Weight:   100,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster2",
						},
					},
				},
			},
			want: []ClusterConnectionInfo{
				{
					ClusterName: "cluster1",
				},
				{
					ClusterName: "cluster2",
				},
			},
			unordered: true,
		},
		{
			name: "Zero weight never chosen",
			in: []*GameServerAllocationPolicy{
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   0,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster1",
						},
					},
				},
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   100,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster2",
						},
					},
				},
			},
			want: []ClusterConnectionInfo{
				{
					ClusterName: "cluster2",
				},
			},
		},
		{
			name: "Multiple allocation endpoints test",
			in: []*GameServerAllocationPolicy{
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   100,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName:         "cluster1",
							SecretName:          "secret-name",
							AllocationEndpoints: []string{"alloc1", "alloc2"},
							Namespace:           "ns1",
							ServerCA:            []byte("c2VydmVyQ0E="),
						},
					},
				},
			},
			want: []ClusterConnectionInfo{
				{
					ClusterName:         "cluster1",
					SecretName:          "secret-name",
					AllocationEndpoints: []string{"alloc1", "alloc2"},
					Namespace:           "ns1",
					ServerCA:            []byte("c2VydmVyQ0E="),
				},
			},
		},
		{
			name: "Empty policy list",
			in:   []*GameServerAllocationPolicy{},
			want: nil,
		},
		{
			name: "Nil policy list",
			in:   nil,
			want: nil,
		},
		{
			name: "Same clusters and same priorities",
			in: []*GameServerAllocationPolicy{
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   100,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster-name",
						},
					},
				},
				{
					Spec: GameServerAllocationPolicySpec{
						Priority: 1,
						Weight:   300,
						ConnectionInfo: ClusterConnectionInfo{
							ClusterName: "cluster-name",
						},
					},
				},
			},
			want: []ClusterConnectionInfo{
				{
					ClusterName: "cluster-name",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var results []ClusterConnectionInfo
			iterator := NewConnectionInfoIterator(tc.in)
			for {
				connectionInfo := iterator.Next()
				if connectionInfo == nil {
					break
				}
				results = append(results, *connectionInfo)
			}

			if tc.unordered {
				assert.ElementsMatch(t, tc.want, results, "Failed test \"%s\"", tc.name)
			} else {
				assert.Equal(t, tc.want, results, "Failed test \"%s\"", tc.name)
			}
		})
	}
}

func TestConnectionInfoIterator_SameClustersAndPriorities(t *testing.T) {
	in := []*GameServerAllocationPolicy{
		{
			Spec: GameServerAllocationPolicySpec{
				Priority: 444,
				Weight:   100,
				ConnectionInfo: ClusterConnectionInfo{
					ClusterName: "cluster-name",
				},
			},
		},
		{
			Spec: GameServerAllocationPolicySpec{
				Priority: 444,
				Weight:   300,
				ConnectionInfo: ClusterConnectionInfo{
					ClusterName: "cluster-name",
				},
			},
		},
	}

	iterator := NewConnectionInfoIterator(in)
	res := iterator.priorityToCluster[444]["cluster-name"]

	// check an internal slice of policies
	if assert.Equal(t, 2, len(res)) {
		assert.Equal(t, "cluster-name", res[0].Spec.ConnectionInfo.ClusterName)
		assert.Equal(t, int32(444), res[0].Spec.Priority)
		assert.Equal(t, "cluster-name", res[1].Spec.ConnectionInfo.ClusterName)
		assert.Equal(t, int32(444), res[1].Spec.Priority)
	}
}
