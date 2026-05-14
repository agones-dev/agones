// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	multiclusterv1 "agones.dev/agones/pkg/apis/multicluster/v1"
	"agones.dev/agones/pkg/util/runtime"
	helper "agones.dev/agones/test/e2e/allochelper"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	agonesSystemNamespace          = "agones-system"
	allocatorServiceName           = "agones-allocator"
	allocatorTLSName               = "allocator-tls"
	tlsCrtTag                      = "tls.crt"
	tlsKeyTag                      = "tls.key"
	allocatorReqURLFmt             = "%s:%d"
	allocatorClientSecretName      = "allocator-client.default"
	allocatorClientSecretNamespace = "default"
)

func TestAllocatorWithDeprecatedRequired(t *testing.T) {
	ctx := context.Background()

	ip, port := helper.GetAllocatorEndpoint(ctx, t, framework)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := helper.RefreshAllocatorTLSCerts(ctx, t, ip, framework)

	var flt *agonesv1.Fleet
	var err error
	if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
		flt, err = helper.CreateFleetWithOpts(ctx, framework.Namespace, framework, func(f *agonesv1.Fleet) {
			f.Spec.Template.Spec.Players = &agonesv1.PlayersSpec{
				InitialCapacity: 10,
			}
		})
	} else {
		flt, err = helper.CreateFleet(ctx, framework.Namespace, framework)
	}
	require.NoError(t, err)
	defer framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Delete(ctx, flt.Name, metav1.DeleteOptions{}) // nolint: errcheck

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	request := &pb.AllocationRequest{
		Namespace:                    framework.Namespace,
		RequiredGameServerSelector:   &pb.GameServerSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}},
		PreferredGameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:                   pb.AllocationRequest_Packed,
		Metadata:                     &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest"}},
	}

	var response *pb.AllocationResponse
	// wait for the allocation system to come online
	err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		// create the grpc client each time, as we may end up looking at an old cert
		dialOpts, err := helper.CreateRemoteClusterDialOptions(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA, framework)
		if err != nil {
			return false, err
		}

		conn, err := grpc.NewClient(requestURL, dialOpts...)
		if err != nil {
			logrus.WithError(err).Info("failing grpc.NewClient")
			return false, nil
		}
		defer conn.Close() // nolint: errcheck

		grpcClient := pb.NewAllocationServiceClient(conn)
		response, err = grpcClient.Allocate(context.Background(), request)
		if err != nil {
			logrus.WithError(err).Info("failing Allocate request")
			return false, nil
		}
		helper.ValidateAllocatorResponse(t, response)

		// let's do a re-allocation
		if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
			// nolint:staticcheck
			request.PreferredGameServerSelectors[0].GameServerState = pb.GameServerSelector_ALLOCATED
			allocatedResponse, err := grpcClient.Allocate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, response.GameServerName, allocatedResponse.GameServerName)
			helper.ValidateAllocatorResponse(t, allocatedResponse)

			// do a capacity based allocation
			logrus.Info("testing capacity allocation filter")
			// nolint:staticcheck
			request.PreferredGameServerSelectors[0].Players = &pb.PlayerSelector{
				MinAvailable: 5,
				MaxAvailable: 10,
			}
			allocatedResponse, err = grpcClient.Allocate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, response.GameServerName, allocatedResponse.GameServerName)
			helper.ValidateAllocatorResponse(t, allocatedResponse)

			// do a capacity based allocation that should fail
			// nolint:staticcheck
			request.PreferredGameServerSelectors = nil
			// nolint:staticcheck
			request.RequiredGameServerSelector.GameServerState = pb.GameServerSelector_ALLOCATED
			// nolint:staticcheck
			request.RequiredGameServerSelector.Players = &pb.PlayerSelector{MinAvailable: 99, MaxAvailable: 200}

			allocatedResponse, err = grpcClient.Allocate(context.Background(), request)
			assert.Nil(t, allocatedResponse)
			status, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.ResourceExhausted, status.Code())
		}

		return true, nil
	})

	require.NoError(t, err)
}

func TestAllocatorWithSelectors(t *testing.T) {
	ctx := context.Background()

	ip, port := helper.GetAllocatorEndpoint(ctx, t, framework)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := helper.RefreshAllocatorTLSCerts(ctx, t, ip, framework)

	var flt *agonesv1.Fleet
	var err error
	if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
		flt, err = helper.CreateFleetWithOpts(ctx, framework.Namespace, framework, func(f *agonesv1.Fleet) {
			f.Spec.Template.Spec.Players = &agonesv1.PlayersSpec{
				InitialCapacity: 10,
			}
		})
	} else {
		flt, err = helper.CreateFleet(ctx, framework.Namespace, framework)
	}
	assert.NoError(t, err)
	defer framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Delete(ctx, flt.Name, metav1.DeleteOptions{}) // nolint: errcheck

	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	request := &pb.AllocationRequest{
		Namespace:           framework.Namespace,
		GameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:          pb.AllocationRequest_Packed,
		Metadata:            &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest", "blue-frog.fred_thing": "test.dog_fred-blue"}},
	}

	var response *pb.AllocationResponse
	// wait for the allocation system to come online
	err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		// create the grpc client each time, as we may end up looking at an old cert
		dialOpts, err := helper.CreateRemoteClusterDialOptions(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA, framework)
		if err != nil {
			return false, err
		}

		conn, err := grpc.NewClient(requestURL, dialOpts...)
		if err != nil {
			logrus.WithError(err).Info("failing grpc.NewClient")
			return false, nil
		}
		defer conn.Close() // nolint: errcheck

		grpcClient := pb.NewAllocationServiceClient(conn)
		response, err = grpcClient.Allocate(context.Background(), request)
		if err != nil {
			logrus.WithError(err).Info("failing Allocate request")
			return false, nil
		}
		helper.ValidateAllocatorResponse(t, response)

		// let's do a re-allocation
		if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) {
			request.GameServerSelectors[0].GameServerState = pb.GameServerSelector_ALLOCATED
			allocatedResponse, err := grpcClient.Allocate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, response.GameServerName, allocatedResponse.GameServerName)
			helper.ValidateAllocatorResponse(t, allocatedResponse)
			assert.Equal(t, flt.ObjectMeta.Name, allocatedResponse.Metadata.Labels[agonesv1.FleetNameLabel])

			// do a capacity based allocation
			logrus.Info("testing capacity allocation filter")
			request.GameServerSelectors[0].Players = &pb.PlayerSelector{
				MinAvailable: 5,
				MaxAvailable: 10,
			}
			allocatedResponse, err = grpcClient.Allocate(context.Background(), request)
			require.NoError(t, err)
			require.Equal(t, response.GameServerName, allocatedResponse.GameServerName)
			helper.ValidateAllocatorResponse(t, allocatedResponse)

			// do a capacity based allocation that should fail
			request.GameServerSelectors[0].GameServerState = pb.GameServerSelector_ALLOCATED
			request.GameServerSelectors[0].Players = &pb.PlayerSelector{MinAvailable: 99, MaxAvailable: 200}

			allocatedResponse, err = grpcClient.Allocate(context.Background(), request)
			assert.Nil(t, allocatedResponse)
			status, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.ResourceExhausted, status.Code())
		}

		return true, nil
	})

	assert.NoError(t, err)
}

func TestRestAllocatorWithDeprecatedRequired(t *testing.T) {
	ctx := context.Background()

	ip, port := helper.GetAllocatorEndpoint(ctx, t, framework)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := helper.RefreshAllocatorTLSCerts(ctx, t, ip, framework)

	flt, err := helper.CreateFleet(ctx, framework.Namespace, framework)
	if !assert.Nil(t, err) {
		return
	}
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	defer framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Delete(ctx, flt.Name, metav1.DeleteOptions{}) // nolint: errcheck

	request := &pb.AllocationRequest{
		Namespace:                    framework.Namespace,
		RequiredGameServerSelector:   &pb.GameServerSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}},
		PreferredGameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:                   pb.AllocationRequest_Packed,
		Metadata:                     &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest"}},
	}
	tlsCfg, err := helper.GetTLSConfig(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA, framework)
	if !assert.Nil(t, err) {
		return
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}
	jsonRes, err := json.Marshal(request)
	if !assert.Nil(t, err) {
		return
	}
	req, err := http.NewRequest("POST", "https://"+requestURL+"/gameserverallocation", bytes.NewBuffer(jsonRes))
	if !assert.Nil(t, err) {
		logrus.WithError(err).Info("failed to create rest request")
		return
	}

	// wait for the allocation system to come online
	err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 5*time.Minute, true, func(_ context.Context) (bool, error) {
		resp, err := client.Do(req)
		if err != nil {
			logrus.WithError(err).Info("failed Allocate rest request")
			return false, nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logrus.WithError(err).Info("failed to read Allocate response body")
			return false, nil
		}
		defer resp.Body.Close() // nolint: errcheck
		var response pb.AllocationResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			logrus.WithError(err).Info("failed to unmarshal Allocate response")
			return false, nil
		}
		helper.ValidateAllocatorResponse(t, &response)
		return true, nil
	})

	assert.NoError(t, err)
}

func TestAllocatorWithCountersAndLists(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.Skip("FeatureCountsAndLists is not enabled")
		return
	}
	ctx := context.Background()

	ip, port := helper.GetAllocatorEndpoint(ctx, t, framework)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := helper.RefreshAllocatorTLSCerts(ctx, t, ip, framework)

	var flt *agonesv1.Fleet
	var err error
	flt, err = helper.CreateFleetWithOpts(ctx, framework.Namespace, framework, func(f *agonesv1.Fleet) {
		f.Spec.Template.Spec.Counters = map[string]agonesv1.CounterStatus{
			"players": {
				Capacity: 10,
			},
		}
		f.Spec.Template.Spec.Lists = map[string]agonesv1.ListStatus{
			"rooms": {
				Capacity: 10,
			},
		}
	})
	assert.NoError(t, err)
	defer framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Delete(ctx, flt.Name, metav1.DeleteOptions{}) // nolint: errcheck
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	request := &pb.AllocationRequest{
		Namespace: framework.Namespace,
		GameServerSelectors: []*pb.GameServerSelector{{
			MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name},
			Counters: map[string]*pb.CounterSelector{
				"players": {
					MinAvailable: 1,
				},
			},
			Lists: map[string]*pb.ListSelector{
				"rooms": {
					MinAvailable: 1,
				},
			},
		}},
		Counters: map[string]*pb.CounterAction{
			"players": {
				Action: wrapperspb.String(agonesv1.GameServerPriorityIncrement),
				Amount: wrapperspb.Int64(1),
			},
		},
		Lists: map[string]*pb.ListAction{
			"rooms": {
				AddValues:    []string{"1"},
				DeleteValues: []string{"2"}, // This action is ignored. (Value does not exist.)
			},
		},
	}
	err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		dialOpts, err := helper.CreateRemoteClusterDialOptions(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA, framework)
		if err != nil {
			return false, err
		}
		conn, err := grpc.NewClient(requestURL, dialOpts...)
		if err != nil {
			logrus.WithError(err).Info("failing grpc.NewClient")
			return false, nil
		}
		defer conn.Close() // nolint: errcheck

		grpcClient := pb.NewAllocationServiceClient(conn)
		response, err := grpcClient.Allocate(context.Background(), request)
		if err != nil {
			return false, nil
		}
		assert.Contains(t, response.GetCounters(), "players")
		assert.Equal(t, int64(10), response.GetCounters()["players"].Capacity.GetValue())
		assert.Equal(t, int64(1), response.GetCounters()["players"].Count.GetValue())
		assert.Contains(t, response.GetLists(), "rooms")
		assert.Equal(t, int64(10), response.GetLists()["rooms"].Capacity.GetValue())
		assert.EqualValues(t, request.Lists["rooms"].AddValues, response.GetLists()["rooms"].Values)
		assert.NotEqualValues(t, request.Lists["rooms"].DeleteValues, response.GetLists()["rooms"].Values)
		return true, nil
	})
	require.NoError(t, err)
}

func TestRestAllocatorWithCountersAndLists(t *testing.T) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		t.Skip("FeatureCountsAndLists is not enabled")
		return
	}
	ctx := context.Background()

	ip, port := helper.GetAllocatorEndpoint(ctx, t, framework)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := helper.RefreshAllocatorTLSCerts(ctx, t, ip, framework)

	var flt *agonesv1.Fleet
	var err error
	flt, err = helper.CreateFleetWithOpts(ctx, framework.Namespace, framework, func(f *agonesv1.Fleet) {
		f.Spec.Template.Spec.Counters = map[string]agonesv1.CounterStatus{
			"players": {
				Capacity: 10,
			},
		}
		f.Spec.Template.Spec.Lists = map[string]agonesv1.ListStatus{
			"rooms": {
				Values:   []string{"one", "two", "three"},
				Capacity: 10,
			},
		}
	})
	assert.NoError(t, err)
	defer framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Delete(ctx, flt.Name, metav1.DeleteOptions{}) // nolint: errcheck
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	request := &pb.AllocationRequest{
		Namespace: framework.Namespace,
		GameServerSelectors: []*pb.GameServerSelector{{
			MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name},
			Counters: map[string]*pb.CounterSelector{
				"players": {
					MinAvailable: 1,
				},
			},
			Lists: map[string]*pb.ListSelector{
				"rooms": {
					MinAvailable: 1,
				},
			},
		}},
		Counters: map[string]*pb.CounterAction{
			"players": {
				Action: wrapperspb.String(agonesv1.GameServerPriorityIncrement),
				Amount: wrapperspb.Int64(1),
			},
		},
		Lists: map[string]*pb.ListAction{
			"rooms": {
				AddValues:    []string{"1"},
				DeleteValues: []string{"three", "one"},
			},
		},
	}
	err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		tlsCfg, err := helper.GetTLSConfig(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA, framework)
		if !assert.Nil(t, err) {
			return false, err
		}
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
		}
		jsonRes, err := protojson.Marshal(request)
		if !assert.Nil(t, err) {
			return false, nil
		}
		req, err := http.NewRequest("POST", "https://"+requestURL+"/gameserverallocation", bytes.NewBuffer(jsonRes))
		if !assert.Nil(t, err) {
			return false, nil
		}
		resp, err := client.Do(req)
		if err != nil {
			return false, nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close() // nolint: errcheck
		if resp.StatusCode != http.StatusOK {
			return false, nil
		}
		var response pb.AllocationResponse
		err = protojson.Unmarshal(body, &response)
		if err != nil {
			return false, nil
		}
		assert.Contains(t, response.GetCounters(), "players")
		assert.Equal(t, int64(10), response.GetCounters()["players"].Capacity.GetValue())
		assert.Equal(t, int64(1), response.GetCounters()["players"].Count.GetValue())
		assert.Contains(t, response.GetLists(), "rooms")
		assert.Equal(t, int64(10), response.GetLists()["rooms"].Capacity.GetValue())
		assert.Contains(t, response.GetLists()["rooms"].Values, request.Lists["rooms"].AddValues[0])
		assert.NotContains(t, response.GetLists()["rooms"].Values, request.Lists["rooms"].DeleteValues[0])
		assert.NotContains(t, response.GetLists()["rooms"].Values, request.Lists["rooms"].DeleteValues[1])
		return true, nil
	})
	require.NoError(t, err)
}

func TestRestAllocatorWithSelectors(t *testing.T) {
	ctx := context.Background()

	ip, port := helper.GetAllocatorEndpoint(ctx, t, framework)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := helper.RefreshAllocatorTLSCerts(ctx, t, ip, framework)

	flt, err := helper.CreateFleet(ctx, framework.Namespace, framework)
	require.NoError(t, err)
	defer framework.AgonesClient.AgonesV1().Fleets(framework.Namespace).Delete(ctx, flt.Name, metav1.DeleteOptions{}) // nolint: errcheck
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	request := &pb.AllocationRequest{
		Namespace:           framework.Namespace,
		GameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:          pb.AllocationRequest_Packed,
		Metadata:            &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest", "blue-frog.fred_thing": "test.dog_fred-blue"}},
	}
	tlsCfg, err := helper.GetTLSConfig(ctx, allocatorClientSecretNamespace, allocatorClientSecretName, tlsCA, framework)
	if !assert.Nil(t, err) {
		return
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}
	jsonRes, err := json.Marshal(request)
	if !assert.Nil(t, err) {
		return
	}
	req, err := http.NewRequest("POST", "https://"+requestURL+"/gameserverallocation", bytes.NewBuffer(jsonRes))
	if !assert.Nil(t, err) {
		logrus.WithError(err).Info("failed to create rest request")
		return
	}

	// wait for the allocation system to come online
	var response pb.AllocationResponse
	err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 5*time.Minute, true, func(_ context.Context) (bool, error) {
		resp, err := client.Do(req)
		if err != nil {
			logrus.WithError(err).Info("failed Allocate rest request")
			return false, nil
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logrus.WithError(err).Info("failed to read Allocate response body")
			return false, nil
		}
		defer resp.Body.Close() // nolint: errcheck
		err = json.Unmarshal(body, &response)
		if err != nil {
			logrus.WithError(err).Info("failed to unmarshal Allocate response")
			return false, nil
		}
		helper.ValidateAllocatorResponse(t, &response)
		return true, nil
	})
	require.NoError(t, err)

	gs, err := framework.AgonesClient.AgonesV1().GameServers(framework.Namespace).Get(ctx, response.GameServerName, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, agonesv1.GameServerStateAllocated, gs.Status.State)
	assert.Equal(t, "allocatedbytest", gs.ObjectMeta.Labels["gslabel"])
	assert.Equal(t, "test.dog_fred-blue", gs.ObjectMeta.Labels["blue-frog.fred_thing"])
}

// Tests multi-cluster allocation by reusing the same cluster but across namespace.
// Multi-cluster is represented as two namespaces A and B in the same cluster.
// Namespace A received the allocation request, but because namespace B has the highest priority, A will forward the request to B.
func TestAllocatorCrossNamespace(t *testing.T) {
	ctx := context.Background()

	ip, port := helper.GetAllocatorEndpoint(ctx, t, framework)
	requestURL := fmt.Sprintf(allocatorReqURLFmt, ip, port)
	tlsCA := helper.RefreshAllocatorTLSCerts(ctx, t, ip, framework)

	// Create namespaces A and B
	namespaceA := framework.Namespace // let's reuse an existing one
	helper.CopyDefaultAllocatorClientSecret(ctx, t, namespaceA, framework)

	namespaceB := fmt.Sprintf("allocator-b-%s", uuid.NewUUID())
	err := framework.CreateNamespace(namespaceB)
	if !assert.Nil(t, err) {
		return
	}
	defer func() {
		if derr := framework.DeleteNamespace(namespaceB); derr != nil {
			t.Error(derr)
		}
	}()

	policyName := fmt.Sprintf("a-to-b-%s", uuid.NewUUID())
	p := &multiclusterv1.GameServerAllocationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyName,
			Namespace: namespaceA,
		},
		Spec: multiclusterv1.GameServerAllocationPolicySpec{
			Priority: 1,
			Weight:   1,
			ConnectionInfo: multiclusterv1.ClusterConnectionInfo{
				SecretName:          allocatorClientSecretName,
				Namespace:           namespaceB,
				AllocationEndpoints: []string{ip},
				ServerCA:            tlsCA,
			},
		},
	}
	helper.CreateAllocationPolicy(ctx, t, framework, p)

	// Create a fleet in namespace B. Allocation should not happen in A according to policy
	flt, err := helper.CreateFleet(ctx, namespaceB, framework)
	if !assert.Nil(t, err) {
		return
	}
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))
	defer framework.AgonesClient.AgonesV1().Fleets(namespaceB).Delete(ctx, flt.Name, metav1.DeleteOptions{}) // nolint: errcheck

	request := &pb.AllocationRequest{
		Namespace: namespaceA,
		// Enable multi-cluster setting
		MultiClusterSetting: &pb.MultiClusterSetting{Enabled: true},
		GameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
	}

	// wait for the allocation system to come online
	err = wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		// create the grpc client each time, as we may end up looking at an old cert
		dialOpts, err := helper.CreateRemoteClusterDialOptions(ctx, namespaceA, allocatorClientSecretName, tlsCA, framework)
		if err != nil {
			return false, err
		}

		conn, err := grpc.NewClient(requestURL, dialOpts...)
		if err != nil {
			logrus.WithError(err).Info("failing grpc.NewClient")
			return false, nil
		}
		defer conn.Close() // nolint: errcheck

		grpcClient := pb.NewAllocationServiceClient(conn)
		response, err := grpcClient.Allocate(context.Background(), request)
		if err != nil {
			logrus.WithError(err).Info("failing Allocate request")
			return false, nil
		}
		helper.ValidateAllocatorResponse(t, response)
		return true, nil
	})

	assert.NoError(t, err)
}
