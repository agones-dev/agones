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

package converters

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pb "agones.dev/agones/pkg/allocation/go"
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/util/runtime"
)

func TestConvertAllocationRequestToGameServerAllocation(t *testing.T) {
	allocated := agonesv1.GameServerStateAllocated
	ready := agonesv1.GameServerStateReady
	increment := agonesv1.GameServerPriorityIncrement
	decrement := agonesv1.GameServerPriorityDecrement
	one := int64(1)
	ten := int64(10)

	tests := []struct {
		name     string
		features string
		in       *pb.AllocationRequest
		want     *allocationv1.GameServerAllocation
	}{
		{
			name:     "all fields are set (PlayerAllocationFilter, CountsAndListsFilter)",
			features: fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists),
			in: &pb.AllocationRequest{
				Namespace: "ns",
				MultiClusterSetting: &pb.MultiClusterSetting{
					Enabled: true,
					PolicySelector: &pb.LabelSelector{
						MatchLabels: map[string]string{
							"a": "b",
						},
					},
				},
				RequiredGameServerSelector: &pb.GameServerSelector{
					MatchLabels: map[string]string{
						"c": "d",
					},
					GameServerState: pb.GameServerSelector_READY,
					Players: &pb.PlayerSelector{
						MinAvailable: 10,
						MaxAvailable: 20,
					},
				},
				PreferredGameServerSelectors: []*pb.GameServerSelector{
					{
						MatchLabels: map[string]string{
							"e": "f",
						},
						GameServerState: pb.GameServerSelector_ALLOCATED,
						Players: &pb.PlayerSelector{
							MinAvailable: 5,
							MaxAvailable: 10,
						},
					},
					{
						MatchLabels: map[string]string{
							"g": "h",
						},
					},
				},
				GameServerSelectors: []*pb.GameServerSelector{
					{
						MatchLabels: map[string]string{
							"m": "n",
						},
						GameServerState: pb.GameServerSelector_READY,
						Counters: map[string]*pb.CounterSelector{
							"o": {
								MinCount:     0,
								MaxCount:     10,
								MinAvailable: 1,
								MaxAvailable: 10,
							},
						},
						Lists: map[string]*pb.ListSelector{
							"p": {
								ContainsValue: "abc",
								MinAvailable:  1,
								MaxAvailable:  10,
							},
						},
					},
				},
				Priorities: []*pb.Priority{
					{
						Type:  pb.Priority_Counter,
						Key:   "o",
						Order: pb.Priority_Descending,
					},
					{
						Type:  pb.Priority_List,
						Key:   "p",
						Order: pb.Priority_Ascending,
					},
				},
				Counters: map[string]*pb.CounterAction{
					"o": {
						Action: wrapperspb.String("Increment"),
						Amount: wrapperspb.Int64(1),
					},
					"q": {
						Action:   wrapperspb.String("Decrement"),
						Amount:   wrapperspb.Int64(1),
						Capacity: wrapperspb.Int64(10),
					},
				},
				Lists: map[string]*pb.ListAction{
					"p": {
						AddValues:    []string{"foo", "bar", "baz"},
						Capacity:     wrapperspb.Int64(10),
						DeleteValues: []string{"alice", "bob", "cat"},
					},
				},
				Scheduling: pb.AllocationRequest_Packed,
				Metadata: &pb.MetaPatch{
					Labels: map[string]string{
						"i": "j",
					},
				},
				MetaPatch: &pb.MetaPatch{
					Labels: map[string]string{
						"i": "j",
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: true,
						PolicySelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"a": "b",
							},
						},
					},
					Required: allocationv1.GameServerSelector{
						LabelSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"c": "d",
							},
						},
						GameServerState: &ready,
						Players:         &allocationv1.PlayerSelector{MinAvailable: 10, MaxAvailable: 20},
					},
					Preferred: []allocationv1.GameServerSelector{
						{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									"e": "f",
								},
							},
							GameServerState: &allocated,
							Players:         &allocationv1.PlayerSelector{MinAvailable: 5, MaxAvailable: 10},
						},
						{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									"g": "h",
								},
							},
							GameServerState: &ready,
						},
					},
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "o",
							Order: "Descending",
						},
						{
							Type:  "List",
							Key:   "p",
							Order: "Ascending",
						},
					},
					Counters: map[string]allocationv1.CounterAction{
						"o": {
							Action: &increment,
							Amount: &one,
						},
						"q": {
							Action:   &decrement,
							Amount:   &one,
							Capacity: &ten,
						},
					},
					Lists: map[string]allocationv1.ListAction{
						"p": {
							AddValues:    []string{"foo", "bar", "baz"},
							Capacity:     &ten,
							DeleteValues: []string{"alice", "bob", "cat"},
						},
					},
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									"m": "n",
								},
							},
							GameServerState: &ready,
							Counters: map[string]allocationv1.CounterSelector{
								"o": {
									MinCount:     0,
									MaxCount:     10,
									MinAvailable: 1,
									MaxAvailable: 10,
								},
							},
							Lists: map[string]allocationv1.ListSelector{
								"p": {
									ContainsValue: "abc",
									MinAvailable:  1,
									MaxAvailable:  10,
								},
							},
						},
					},
					Scheduling: apis.Packed,
					MetaPatch: allocationv1.MetaPatch{
						Labels: map[string]string{
							"i": "j",
						},
					},
				},
			},
		},
		{
			name:     "all fields are set",
			features: fmt.Sprintf("%s=false&%s=false", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists),
			in: &pb.AllocationRequest{
				Namespace: "ns",
				MultiClusterSetting: &pb.MultiClusterSetting{
					Enabled: true,
					PolicySelector: &pb.LabelSelector{
						MatchLabels: map[string]string{
							"a": "b",
						},
					},
				},
				RequiredGameServerSelector: &pb.GameServerSelector{
					MatchLabels: map[string]string{
						"c": "d",
					},
				},
				PreferredGameServerSelectors: []*pb.GameServerSelector{
					{
						MatchLabels: map[string]string{
							"e": "f",
						},
					},
					{
						MatchLabels: map[string]string{
							"g": "h",
						},
					},
				},
				GameServerSelectors: []*pb.GameServerSelector{
					{
						MatchLabels: map[string]string{
							"m": "n",
						},
					},
				},
				Scheduling: pb.AllocationRequest_Packed,
				Metadata: &pb.MetaPatch{
					Labels: map[string]string{
						"i": "j",
					},
				},
				MetaPatch: &pb.MetaPatch{
					Labels: map[string]string{
						"i": "j",
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "ns",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: true,
						PolicySelector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"a": "b",
							},
						},
					},
					Required: allocationv1.GameServerSelector{LabelSelector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"c": "d",
						},
					},
						GameServerState: &ready,
					},
					Preferred: []allocationv1.GameServerSelector{
						{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									"e": "f",
								},
							},
							GameServerState: &ready,
						},
						{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									"g": "h",
								},
							},
							GameServerState: &ready,
						},
					},
					Selectors: []allocationv1.GameServerSelector{
						{
							LabelSelector: metav1.LabelSelector{
								MatchLabels: map[string]string{
									"m": "n",
								},
							},
							GameServerState: &ready,
						},
					},
					Scheduling: apis.Packed,
					MetaPatch: allocationv1.MetaPatch{
						Labels: map[string]string{
							"i": "j",
						},
					},
				},
			},
		},
		{
			name:     "empty fields to GSA",
			features: fmt.Sprintf("%s=false&%s=false", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists),
			in: &pb.AllocationRequest{
				Namespace:                    "",
				MultiClusterSetting:          &pb.MultiClusterSetting{},
				RequiredGameServerSelector:   &pb.GameServerSelector{},
				PreferredGameServerSelectors: []*pb.GameServerSelector{},
				Scheduling:                   pb.AllocationRequest_Distributed,
				Metadata:                     &pb.MetaPatch{},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Scheduling: apis.Distributed,
					Required: allocationv1.GameServerSelector{
						GameServerState: &ready,
					},
				},
			},
		},
		{
			name:     "empty fields to GSA (PlayerAllocationFilter, CountsAndListsFilter)",
			features: fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists),
			in: &pb.AllocationRequest{
				Namespace:                    "",
				MultiClusterSetting:          &pb.MultiClusterSetting{},
				RequiredGameServerSelector:   &pb.GameServerSelector{},
				PreferredGameServerSelectors: []*pb.GameServerSelector{},
				Scheduling:                   pb.AllocationRequest_Distributed,
				Metadata:                     &pb.MetaPatch{},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Required: allocationv1.GameServerSelector{
						GameServerState: &ready,
					},
					Scheduling: apis.Distributed,
				},
			},
		},
		{
			name:     "empty fields to GSA with selectors fields",
			features: fmt.Sprintf("%s=false&%s=false", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists),
			in: &pb.AllocationRequest{
				Namespace:           "",
				MultiClusterSetting: &pb.MultiClusterSetting{},
				GameServerSelectors: []*pb.GameServerSelector{{}},
				Scheduling:          pb.AllocationRequest_Distributed,
				Metadata:            &pb.MetaPatch{},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Selectors: []allocationv1.GameServerSelector{{
						GameServerState: &ready,
					}},
					Scheduling: apis.Distributed,
				},
			},
		},
		{
			name:     "empty fields to GSA (PlayerAllocationFilter, CountsAndListsFilter) with selectors fields",
			features: fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists),
			in: &pb.AllocationRequest{
				Namespace:           "",
				MultiClusterSetting: &pb.MultiClusterSetting{},
				GameServerSelectors: []*pb.GameServerSelector{{}},
				Scheduling:          pb.AllocationRequest_Distributed,
				Metadata:            &pb.MetaPatch{},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Selectors: []allocationv1.GameServerSelector{
						{GameServerState: &ready},
					},
					Scheduling: apis.Distributed,
				},
			},
		},
		{
			name: "empty object to GSA",
			in:   &pb.AllocationRequest{},
			want: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: apis.Packed,
				},
			},
		},
		{
			name: "nil object",
			in:   nil,
			want: nil,
		},
		{
			name: "accepts deprecated metapatch field",
			in: &pb.AllocationRequest{
				Namespace:                    "",
				MultiClusterSetting:          &pb.MultiClusterSetting{},
				RequiredGameServerSelector:   &pb.GameServerSelector{},
				PreferredGameServerSelectors: []*pb.GameServerSelector{},
				Scheduling:                   pb.AllocationRequest_Distributed,
				MetaPatch: &pb.MetaPatch{
					Labels: map[string]string{
						"a": "b",
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Required: allocationv1.GameServerSelector{
						GameServerState: &ready,
					},
					Scheduling: apis.Distributed,
					MetaPatch: allocationv1.MetaPatch{
						Labels: map[string]string{
							"a": "b",
						},
					},
				},
			},
		},
		{
			name: "Prefers metadata over metapatch field",
			in: &pb.AllocationRequest{
				Namespace:                    "",
				MultiClusterSetting:          &pb.MultiClusterSetting{},
				RequiredGameServerSelector:   &pb.GameServerSelector{},
				PreferredGameServerSelectors: []*pb.GameServerSelector{},
				Scheduling:                   pb.AllocationRequest_Distributed,
				Metadata: &pb.MetaPatch{
					Labels: map[string]string{
						"a": "b",
					},
				},
				MetaPatch: &pb.MetaPatch{
					Labels: map[string]string{
						"c": "d",
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Scheduling: apis.Distributed,
					Required: allocationv1.GameServerSelector{
						GameServerState: &ready,
					},
					MetaPatch: allocationv1.MetaPatch{
						Labels: map[string]string{
							"a": "b",
						},
					},
				},
			},
		},
		{
			name:     "partially empty Counters and Lists fields to GSA (PlayerAllocationFilter, CountsAndListsFilter)",
			features: fmt.Sprintf("%s=true&%s=true", runtime.FeaturePlayerAllocationFilter, runtime.FeatureCountsAndLists),
			in: &pb.AllocationRequest{
				Namespace:                    "",
				MultiClusterSetting:          &pb.MultiClusterSetting{},
				RequiredGameServerSelector:   &pb.GameServerSelector{},
				PreferredGameServerSelectors: []*pb.GameServerSelector{},
				GameServerSelectors: []*pb.GameServerSelector{
					{
						GameServerState: pb.GameServerSelector_READY,
						Counters: map[string]*pb.CounterSelector{
							"c": {
								MinAvailable: 10,
							},
						},
						Lists: map[string]*pb.ListSelector{
							"d": {
								ContainsValue: "abc",
								MinAvailable:  1,
							},
						},
					},
				},
				Lists: map[string]*pb.ListAction{
					"d": {
						Capacity: wrapperspb.Int64(one),
					},
				},
				Scheduling: pb.AllocationRequest_Distributed,
				Metadata:   &pb.MetaPatch{},
			},
			want: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Required: allocationv1.GameServerSelector{
						GameServerState: &ready,
					},
					Selectors: []allocationv1.GameServerSelector{
						{
							GameServerState: &ready,
							Counters: map[string]allocationv1.CounterSelector{
								"c": {
									MinCount:     0,
									MaxCount:     0,
									MinAvailable: 10,
									MaxAvailable: 0,
								},
							},
							Lists: map[string]allocationv1.ListSelector{
								"d": {
									ContainsValue: "abc",
									MinAvailable:  1,
									MaxAvailable:  0,
								},
							},
						},
					},
					Lists: map[string]allocationv1.ListAction{
						"d": {
							Capacity: &one,
						},
					},
					Scheduling: apis.Distributed,
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			require.NoError(t, runtime.ParseFeatures(tc.features))

			out := ConvertAllocationRequestToGSA(tc.in)
			assert.Equal(t, tc.want, out, "mismatch with want after conversion: \"%s\"", tc.name)
		})
	}
}

func TestConvertGSAToAllocationRequest(t *testing.T) {
	increment := agonesv1.GameServerPriorityIncrement
	decrement := agonesv1.GameServerPriorityDecrement
	two := int64(2)
	twenty := int64(20)

	tests := []struct {
		name     string
		features string
		in       *allocationv1.GameServerAllocation
		want     *pb.AllocationRequest
	}{
		{
			name: "empty fields",
			in: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Scheduling: apis.Distributed,
				},
			},
			want: &pb.AllocationRequest{
				Namespace:           "",
				MultiClusterSetting: &pb.MultiClusterSetting{},
				Scheduling:          pb.AllocationRequest_Distributed,
				Metadata:            &pb.MetaPatch{},
				MetaPatch:           &pb.MetaPatch{},
			},
		}, {
			name: "empty object",
			in: &allocationv1.GameServerAllocation{
				Spec: allocationv1.GameServerAllocationSpec{
					Scheduling: apis.Packed,
				},
			},
			want: &pb.AllocationRequest{
				MultiClusterSetting: &pb.MultiClusterSetting{},
				Metadata:            &pb.MetaPatch{},
				MetaPatch:           &pb.MetaPatch{},
			},
		}, {
			name:     "partial GSA with CountsAndLists",
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			in: &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "",
				},
				Spec: allocationv1.GameServerAllocationSpec{
					MultiClusterSetting: allocationv1.MultiClusterSetting{
						Enabled: false,
					},
					Selectors: []allocationv1.GameServerSelector{
						{
							Counters: map[string]allocationv1.CounterSelector{
								"a": {
									MinCount:     0,
									MaxCount:     0,
									MinAvailable: 10,
									MaxAvailable: 0,
								},
							},
							Lists: map[string]allocationv1.ListSelector{
								"b": {
									ContainsValue: "abc",
									MinAvailable:  1,
									MaxAvailable:  0,
								},
							},
						},
					},
					Priorities: []agonesv1.Priority{
						{
							Type:  "Counter",
							Key:   "a",
							Order: "Ascending",
						},
						{
							Type:  "List",
							Key:   "b",
							Order: "Descending",
						},
					},
					Counters: map[string]allocationv1.CounterAction{
						"a": {
							Action:   &decrement,
							Amount:   &two,
							Capacity: &twenty,
						},
						"c": {
							Action: &increment,
							Amount: &two,
						},
					},
					Lists: map[string]allocationv1.ListAction{
						"b": {
							AddValues: []string{"hello", "world"},
						},
						"d": {
							Capacity: &two,
						},
						"c": {
							DeleteValues: []string{"good", "bye"},
						},
					},
					Scheduling: apis.Distributed,
				},
			},
			want: &pb.AllocationRequest{
				Namespace:                    "",
				MultiClusterSetting:          &pb.MultiClusterSetting{},
				PreferredGameServerSelectors: []*pb.GameServerSelector{},
				RequiredGameServerSelector: &pb.GameServerSelector{
					GameServerState: pb.GameServerSelector_READY,
					Counters: map[string]*pb.CounterSelector{
						"a": {
							MinCount:     0,
							MaxCount:     0,
							MinAvailable: 10,
							MaxAvailable: 0,
						},
					},
					Lists: map[string]*pb.ListSelector{
						"b": {
							ContainsValue: "abc",
							MinAvailable:  1,
							MaxAvailable:  0,
						},
					},
				},
				GameServerSelectors: []*pb.GameServerSelector{
					{
						GameServerState: pb.GameServerSelector_READY,
						Counters: map[string]*pb.CounterSelector{
							"a": {
								MinCount:     0,
								MaxCount:     0,
								MinAvailable: 10,
								MaxAvailable: 0,
							},
						},
						Lists: map[string]*pb.ListSelector{
							"b": {
								ContainsValue: "abc",
								MinAvailable:  1,
								MaxAvailable:  0,
							},
						},
					},
				},
				Scheduling: pb.AllocationRequest_Distributed,
				Metadata:   &pb.MetaPatch{},
				MetaPatch:  &pb.MetaPatch{},
				Priorities: []*pb.Priority{
					{
						Type:  pb.Priority_Counter,
						Key:   "a",
						Order: pb.Priority_Ascending,
					},
					{
						Type:  pb.Priority_List,
						Key:   "b",
						Order: pb.Priority_Descending,
					},
				},
				Counters: map[string]*pb.CounterAction{
					"a": {
						Action:   wrapperspb.String(decrement),
						Amount:   wrapperspb.Int64(two),
						Capacity: wrapperspb.Int64(twenty),
					},
					"c": {
						Action: wrapperspb.String(increment),
						Amount: wrapperspb.Int64(two),
					},
				},
				Lists: map[string]*pb.ListAction{
					"b": {
						AddValues: []string{"hello", "world"},
					},
					"d": {
						Capacity: wrapperspb.Int64(two),
					},
					"c": {
						DeleteValues: []string{"good", "bye"},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			require.NoError(t, runtime.ParseFeatures(tc.features))

			ar := ConvertGSAToAllocationRequest(tc.in)
			if !assert.Equal(t, tc.want, ar) {
				t.Errorf("mismatch with want after conversion \"%s\"", tc.name)
			}
		})
	}
}

func TestConvertGSAToAllocationResponse(t *testing.T) {
	tests := []struct {
		name                      string
		features                  string
		in                        *allocationv1.GameServerAllocation
		grpcUnallocatedStatusCode codes.Code
		want                      *pb.AllocationResponse
		wantErrCode               codes.Code
		skipConvertToGSA          bool
	}{
		{
			name: "status state is set to allocated",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address:  "address",
					NodeName: "node-name",
					Source:   "local",
				},
			},
			want: &pb.AllocationResponse{
				GameServerName: "GSN",
				Address:        "address",
				NodeName:       "node-name",
				Ports: []*pb.AllocationResponse_GameServerStatusPort{
					{
						Port: 123,
					},
					{
						Name: "port-name",
					},
				},
				Source: "local",
			},
		},
		{
			name: "status field is set to unallocated",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationUnAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address:  "address",
					NodeName: "node-name",
				},
			},
			wantErrCode:      codes.ResourceExhausted,
			skipConvertToGSA: true,
		},
		{
			name: "status state is set to contention",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State: allocationv1.GameServerAllocationContention,
				},
			},
			wantErrCode:      codes.Aborted,
			skipConvertToGSA: true,
		},
		{
			name: "Empty fields",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					Ports: []agonesv1.GameServerStatusPort{},
				},
			},
			wantErrCode:      codes.Unknown,
			skipConvertToGSA: true,
		},
		{
			name: "Empty objects",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State: allocationv1.GameServerAllocationAllocated,
				},
			},
			want: &pb.AllocationResponse{},
		},
		{
			name: "nil objects",
			in:   nil,
			want: nil,
		},
		{
			name: "status metadata contains labels and annotations",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address:  "address",
					NodeName: "node-name",
					Source:   "local",
					Metadata: &allocationv1.GameServerMetadata{
						Labels: map[string]string{
							"label-key": "label-value",
							"other-key": "other-value",
						},
						Annotations: map[string]string{
							"annotation-key": "annotation-value",
							"other-key":      "other-value",
						},
					},
				},
			},
			want: &pb.AllocationResponse{
				GameServerName: "GSN",
				Address:        "address",
				NodeName:       "node-name",
				Ports: []*pb.AllocationResponse_GameServerStatusPort{
					{
						Port: 123,
					},
					{
						Name: "port-name",
					},
				},
				Source: "local",
				Metadata: &pb.AllocationResponse_GameServerMetadata{
					Labels: map[string]string{
						"label-key": "label-value",
						"other-key": "other-value",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
						"other-key":      "other-value",
					},
				},
			},
		},
		{
			name: "addresses convert",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address: "address",
					Addresses: []corev1.NodeAddress{
						{Type: "SomeAddressType", Address: "123.123.123.123"},
						{Type: "AnotherAddressType", Address: "321.321.321.321"},
					},
					NodeName: "node-name",
					Source:   "local",
					Metadata: &allocationv1.GameServerMetadata{
						Labels: map[string]string{
							"label-key": "label-value",
							"other-key": "other-value",
						},
						Annotations: map[string]string{
							"annotation-key": "annotation-value",
							"other-key":      "other-value",
						},
					},
				},
			},
			want: &pb.AllocationResponse{
				GameServerName: "GSN",
				Address:        "address",
				Addresses: []*pb.AllocationResponse_GameServerStatusAddress{
					{Type: "SomeAddressType", Address: "123.123.123.123"},
					{Type: "AnotherAddressType", Address: "321.321.321.321"},
				},
				NodeName: "node-name",
				Ports: []*pb.AllocationResponse_GameServerStatusPort{
					{
						Port: 123,
					},
					{
						Name: "port-name",
					},
				},
				Source: "local",
				Metadata: &pb.AllocationResponse_GameServerMetadata{
					Labels: map[string]string{
						"label-key": "label-value",
						"other-key": "other-value",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
						"other-key":      "other-value",
					},
				},
			},
		},
		{
			name:     "all fields are set (CountsAndLists)",
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address: "address",
					Addresses: []corev1.NodeAddress{
						{Type: "SomeAddressType", Address: "123.123.123.123"},
						{Type: "AnotherAddressType", Address: "321.321.321.321"},
					},
					NodeName: "node-name",
					Source:   "local",
					Metadata: &allocationv1.GameServerMetadata{
						Labels: map[string]string{
							"label-key": "label-value",
							"other-key": "other-value",
						},
						Annotations: map[string]string{
							"annotation-key": "annotation-value",
							"other-key":      "other-value",
						},
					},
					Counters: map[string]agonesv1.CounterStatus{
						"p": {
							Count:    0,
							Capacity: 1,
						},
					},
					Lists: map[string]agonesv1.ListStatus{
						"p": {
							Values:   []string{"foo", "bar", "baz"},
							Capacity: 10,
						},
					},
				},
			},
			want: &pb.AllocationResponse{
				GameServerName: "GSN",
				Address:        "address",
				Addresses: []*pb.AllocationResponse_GameServerStatusAddress{
					{Type: "SomeAddressType", Address: "123.123.123.123"},
					{Type: "AnotherAddressType", Address: "321.321.321.321"},
				},
				NodeName: "node-name",
				Ports: []*pb.AllocationResponse_GameServerStatusPort{
					{
						Port: 123,
					},
					{
						Name: "port-name",
					},
				},
				Source: "local",
				Metadata: &pb.AllocationResponse_GameServerMetadata{
					Labels: map[string]string{
						"label-key": "label-value",
						"other-key": "other-value",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
						"other-key":      "other-value",
					},
				},
				Counters: map[string]*pb.AllocationResponse_CounterStatus{
					"p": {
						Count:    wrapperspb.Int64(0),
						Capacity: wrapperspb.Int64(1),
					},
				},
				Lists: map[string]*pb.AllocationResponse_ListStatus{
					"p": {
						Values:   []string{"foo", "bar", "baz"},
						Capacity: wrapperspb.Int64(10),
					},
				},
			},
		},
		{
			name:     "Counters fields are set (CountsAndLists)",
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address: "address",
					Addresses: []corev1.NodeAddress{
						{Type: "SomeAddressType", Address: "123.123.123.123"},
						{Type: "AnotherAddressType", Address: "321.321.321.321"},
					},
					NodeName: "node-name",
					Source:   "local",
					Metadata: &allocationv1.GameServerMetadata{
						Labels: map[string]string{
							"label-key": "label-value",
							"other-key": "other-value",
						},
						Annotations: map[string]string{
							"annotation-key": "annotation-value",
							"other-key":      "other-value",
						},
					},
					Counters: map[string]agonesv1.CounterStatus{
						"p": {
							Count:    0,
							Capacity: 1,
						},
					},
				},
			},
			want: &pb.AllocationResponse{
				GameServerName: "GSN",
				Address:        "address",
				Addresses: []*pb.AllocationResponse_GameServerStatusAddress{
					{Type: "SomeAddressType", Address: "123.123.123.123"},
					{Type: "AnotherAddressType", Address: "321.321.321.321"},
				},
				NodeName: "node-name",
				Ports: []*pb.AllocationResponse_GameServerStatusPort{
					{
						Port: 123,
					},
					{
						Name: "port-name",
					},
				},
				Source: "local",
				Metadata: &pb.AllocationResponse_GameServerMetadata{
					Labels: map[string]string{
						"label-key": "label-value",
						"other-key": "other-value",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
						"other-key":      "other-value",
					},
				},
				Counters: map[string]*pb.AllocationResponse_CounterStatus{
					"p": {
						Count:    wrapperspb.Int64(0),
						Capacity: wrapperspb.Int64(1),
					},
				},
			},
		},
		{
			name:     "Lists fields are set (CountsAndLists)",
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address: "address",
					Addresses: []corev1.NodeAddress{
						{Type: "SomeAddressType", Address: "123.123.123.123"},
						{Type: "AnotherAddressType", Address: "321.321.321.321"},
					},
					NodeName: "node-name",
					Source:   "local",
					Metadata: &allocationv1.GameServerMetadata{
						Labels: map[string]string{
							"label-key": "label-value",
							"other-key": "other-value",
						},
						Annotations: map[string]string{
							"annotation-key": "annotation-value",
							"other-key":      "other-value",
						},
					},
					Lists: map[string]agonesv1.ListStatus{
						"p": {
							Values:   []string{"foo", "bar", "baz"},
							Capacity: 10,
						},
					},
				},
			},
			want: &pb.AllocationResponse{
				GameServerName: "GSN",
				Address:        "address",
				Addresses: []*pb.AllocationResponse_GameServerStatusAddress{
					{Type: "SomeAddressType", Address: "123.123.123.123"},
					{Type: "AnotherAddressType", Address: "321.321.321.321"},
				},
				NodeName: "node-name",
				Ports: []*pb.AllocationResponse_GameServerStatusPort{
					{
						Port: 123,
					},
					{
						Name: "port-name",
					},
				},
				Source: "local",
				Metadata: &pb.AllocationResponse_GameServerMetadata{
					Labels: map[string]string{
						"label-key": "label-value",
						"other-key": "other-value",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
						"other-key":      "other-value",
					},
				},
				Lists: map[string]*pb.AllocationResponse_ListStatus{
					"p": {
						Values:   []string{"foo", "bar", "baz"},
						Capacity: wrapperspb.Int64(10),
					},
				},
			},
		},
		{
			name: "status field is set to unallocated, non-default unallocated",
			in: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:          allocationv1.GameServerAllocationUnAllocated,
					GameServerName: "GSN",
					Ports: []agonesv1.GameServerStatusPort{
						{
							Port: 123,
						},
						{
							Name: "port-name",
						},
					},
					Address:  "address",
					NodeName: "node-name",
				},
			},
			grpcUnallocatedStatusCode: codes.Unimplemented,
			wantErrCode:               codes.Unimplemented,
			skipConvertToGSA:          true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			require.NoError(t, runtime.ParseFeatures(tc.features))

			grpcUnallocatedStatusCode := tc.grpcUnallocatedStatusCode
			if grpcUnallocatedStatusCode == codes.OK {
				grpcUnallocatedStatusCode = codes.ResourceExhausted
			}

			out, err := ConvertGSAToAllocationResponse(tc.in, grpcUnallocatedStatusCode)
			if tc.wantErrCode != 0 {
				st, ok := status.FromError(err)
				if !assert.True(t, ok) {
					return
				}
				assert.Equal(t, tc.wantErrCode, st.Code())
			}
			if !assert.Equal(t, tc.want, out) {
				t.Errorf("mismatch with want after conversion: \"%s\"", tc.name)
			}

			if !tc.skipConvertToGSA {
				source := ""
				if tc.in != nil {
					source = tc.in.Status.Source
				}
				gsa := ConvertAllocationResponseToGSA(tc.want, source)
				if !assert.Equal(t, tc.in, gsa) {
					t.Errorf("mismatch with input after double conversion \"%s\"", tc.name)
				}
			}
		})
	}
}

func TestConvertAllocationResponseToGSA(t *testing.T) {
	tests := []struct {
		name     string
		features string
		in       *pb.AllocationResponse
		want     *allocationv1.GameServerAllocation
	}{
		{
			name: "Empty fields",
			in: &pb.AllocationResponse{
				Ports:  []*pb.AllocationResponse_GameServerStatusPort{},
				Source: "33.188.237.156:443",
			},
			want: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:  allocationv1.GameServerAllocationAllocated,
					Source: "33.188.237.156:443",
				},
			},
		},
		{
			name: "Addresses convert",
			in: &pb.AllocationResponse{
				Ports:  []*pb.AllocationResponse_GameServerStatusPort{},
				Source: "33.188.237.156:443",
				Addresses: []*pb.AllocationResponse_GameServerStatusAddress{
					{Type: "SomeAddressType", Address: "123.123.123.123"},
					{Type: "AnotherAddressType", Address: "321.321.321.321"},
				},
			},
			want: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:  allocationv1.GameServerAllocationAllocated,
					Source: "33.188.237.156:443",
					Addresses: []corev1.NodeAddress{
						{Type: "SomeAddressType", Address: "123.123.123.123"},
						{Type: "AnotherAddressType", Address: "321.321.321.321"},
					},
				},
			},
		},
		{
			name:     "Counters and Lists convert",
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			in: &pb.AllocationResponse{
				Ports:  []*pb.AllocationResponse_GameServerStatusPort{},
				Source: "33.188.237.156:443",
				Counters: map[string]*pb.AllocationResponse_CounterStatus{
					"p": {
						Count:    wrapperspb.Int64(1),
						Capacity: wrapperspb.Int64(3),
					},
				},
				Lists: map[string]*pb.AllocationResponse_ListStatus{
					"p": {
						Values:   []string{"foo", "bar", "baz"},
						Capacity: wrapperspb.Int64(3),
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:  allocationv1.GameServerAllocationAllocated,
					Source: "33.188.237.156:443",
					Counters: map[string]agonesv1.CounterStatus{
						"p": {
							Count:    1,
							Capacity: 3,
						},
					},
					Lists: map[string]agonesv1.ListStatus{
						"p": {
							Values:   []string{"foo", "bar", "baz"},
							Capacity: 3,
						},
					},
				},
			},
		},
		{
			name:     "Counters convert",
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			in: &pb.AllocationResponse{
				Ports:  []*pb.AllocationResponse_GameServerStatusPort{},
				Source: "33.188.237.156:443",
				Counters: map[string]*pb.AllocationResponse_CounterStatus{
					"p": {
						Count:    wrapperspb.Int64(1),
						Capacity: wrapperspb.Int64(3),
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:  allocationv1.GameServerAllocationAllocated,
					Source: "33.188.237.156:443",
					Counters: map[string]agonesv1.CounterStatus{
						"p": {
							Count:    1,
							Capacity: 3,
						},
					},
				},
			},
		},
		{
			name:     "List convert",
			features: fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists),
			in: &pb.AllocationResponse{
				Ports:  []*pb.AllocationResponse_GameServerStatusPort{},
				Source: "33.188.237.156:443",
				Lists: map[string]*pb.AllocationResponse_ListStatus{
					"p": {
						Values:   []string{"foo", "bar", "baz"},
						Capacity: wrapperspb.Int64(3),
					},
				},
			},
			want: &allocationv1.GameServerAllocation{
				TypeMeta: metav1.TypeMeta{
					Kind:       "GameServerAllocation",
					APIVersion: "allocation.agones.dev/v1",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State:  allocationv1.GameServerAllocationAllocated,
					Source: "33.188.237.156:443",
					Lists: map[string]agonesv1.ListStatus{
						"p": {
							Values:   []string{"foo", "bar", "baz"},
							Capacity: 3,
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runtime.FeatureTestMutex.Lock()
			defer runtime.FeatureTestMutex.Unlock()
			require.NoError(t, runtime.ParseFeatures(tc.features))
			out := ConvertAllocationResponseToGSA(tc.in, tc.in.Source)
			if !assert.Equal(t, tc.want, out) {
				t.Errorf("mismatch with want after conversion: \"%s\"", tc.name)
			}
		})
	}
}
