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

package allocator

import (
	"context"
	"testing"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	helper "agones.dev/agones/test/e2e/allochelper"
	e2e "agones.dev/agones/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	retryInterval = 5 * time.Second
	retryTimeout  = 45 * time.Second
)

func TestAllocatorAfterDeleteReplica(t *testing.T) {
	ctx := context.Background()
	logger := e2e.TestLogger(t)

	// initialize gRPC client, which tests the connection
	grpcClient, err := helper.GetAllocatorClient(ctx, t, framework)
	require.NoError(t, err, "Could not initialize rpc client")

	// poll and wait until all allocator pods are available
	err = wait.PollUntilContextTimeout(context.Background(), retryInterval, retryTimeout, true, func(ctx context.Context) (done bool, err error) {
		deployment, err := framework.KubeClient.AppsV1().Deployments("agones-system").Get(ctx, "agones-allocator", metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		if deployment.Status.Replicas != deployment.Status.AvailableReplicas {
			logger.Infof("Waiting for agones-allocator to stabilize: %d/%d replicas available", deployment.Status.AvailableReplicas, deployment.Status.ReadyReplicas)
			return false, nil
		}
		return true, nil
	})
	require.NoError(t, err, "Failed to stabilize agones-allocator")

	// create fleet
	flt, err := helper.CreateFleet(ctx, framework.Namespace, framework)
	require.NoError(t, err)
	framework.AssertFleetCondition(t, flt, e2e.FleetReadyCount(flt.Spec.Replicas))

	logger.Infof("=== agones-allocator available, gRPC client initialized ===")

	// One probe into the test, delete all of the allocators except 1
	go func() {
		time.Sleep(retryInterval)

		list, err := framework.KubeClient.CoreV1().Pods("agones-system").List(
			ctx, metav1.ListOptions{LabelSelector: labels.Set{"multicluster.agones.dev/role": "allocator"}.String()})
		if assert.NoError(t, err, "Could not list allocator pods") {
			for _, pod := range list.Items[1:] {
				logger.Infof("Deleting Pod %s", pod.ObjectMeta.Name)
				err = helper.DeleteAgonesPod(ctx, pod.ObjectMeta.Name, "agones-system", framework)
				assert.NoError(t, err, "Could not delete allocator pod")
			}
		}
	}()

	request := &pb.AllocationRequest{
		Namespace:                    framework.Namespace,
		PreferredGameServerSelectors: []*pb.GameServerSelector{{MatchLabels: map[string]string{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}}},
		Scheduling:                   pb.AllocationRequest_Packed,
		Metadata:                     &pb.MetaPatch{Labels: map[string]string{"gslabel": "allocatedbytest"}},
	}

	// Wait and keep making calls till we know the draining time has passed
	_ = wait.PollUntilContextTimeout(context.Background(), retryInterval, retryTimeout, true, func(_ context.Context) (bool, error) {
		response, err := grpcClient.Allocate(context.Background(), request)
		logger.Infof("err = %v (code = %v), response = %v", err, status.Code(err), response)
		helper.ValidateAllocatorResponse(t, response)
		require.NoError(t, err, "Failed grpc allocation request")
		return false, nil
	})
}
