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

package gameserverallocations

import (
	"context"
	goErrors "errors"
	"time"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// batchResponses is an async list of responses for matching requests
type batchResponses struct {
	counterErrors error
	listErrors    error
	responses     []response
}

// batchAllocationUpdateWorkers tries to update each newly allocated gs with the last state.
// If the update fails because of a version conflict, all allocations that were applied onto
// a gs will receive an error, thus being available for retries.
func (c *Allocator) batchAllocationUpdateWorkers(ctx context.Context, workerCount int) chan<- batchResponses {
	batchUpdateQueue := make(chan batchResponses)

	for i := 0; i < workerCount; i++ {
		go func() {
			for {
				select {
				case batchRes := <-batchUpdateQueue:
					if len(batchRes.responses) > 0 {
						lastGsState := batchRes.responses[len(batchRes.responses)-1].gs

						var propagatedErr error
						updatedGs, updateErr := c.gameServerGetter.GameServers(lastGsState.ObjectMeta.Namespace).Update(ctx, lastGsState, metav1.UpdateOptions{})
						if updateErr != nil {
							if !k8serrors.IsConflict(errors.Cause(updateErr)) {
								c.allocationCache.AddGameServer(lastGsState)
							}
							propagatedErr = goErrors.Join(ErrGameServerUpdateConflict, updateErr)
						} else {
							c.allocationCache.AddGameServer(updatedGs)

							if batchRes.counterErrors != nil {
								c.recorder.Event(updatedGs, corev1.EventTypeWarning, "CounterActionError", batchRes.counterErrors.Error())
							}
							if batchRes.listErrors != nil {
								c.recorder.Event(updatedGs, corev1.EventTypeWarning, "ListActionError", batchRes.listErrors.Error())
							}
							c.recorder.Event(updatedGs, corev1.EventTypeNormal, string(updatedGs.Status.State), "Allocated")
						}

						for _, res := range batchRes.responses {
							res.err = propagatedErr
							res.request.response <- res
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return batchUpdateQueue
}

// ListenAndBatchAllocate is a blocking function that runs in a loop processing allocation
// requests in batches. Unlike ListenAndAllocate, it applies allocations locally to a
// GameServer before batching updates — multiple allocations to the same GameServer within
// a flush window result in a single Kubernetes update, reducing API pressure and improving
// session packing.
func (c *Allocator) ListenAndBatchAllocate(ctx context.Context, updateWorkerCount int) {
	batchUpdateQueue := c.batchAllocationUpdateWorkers(ctx, updateWorkerCount)

	var list []*agonesv1.GameServer
	var sortKey uint64
	requestCount := 0
	gsToReorderIndex := -1
	var gsToReorder *agonesv1.GameServer

	batchResponsesPerGs := make(map[string]batchResponses)

	flush := func() {
		for _, batchRes := range batchResponsesPerGs {
			batchUpdateQueue <- batchRes
		}
		batchResponsesPerGs = make(map[string]batchResponses)

		list = nil
		requestCount = 0
		gsToReorderIndex = -1
		gsToReorder = nil
	}

	checkSortKey := func(gsa *allocationv1.GameServerAllocation) {
		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
			newSortKey, err := gsa.SortKey()
			if err != nil {
				c.baseLogger.WithError(err).Warn("error getting sortKey for GameServerAllocationSpec")
			}
			if sortKey == 0 {
				sortKey = newSortKey
			}

			if newSortKey != sortKey {
				sortKey = newSortKey
				flush()
			}
		}
	}

	checkRefreshList := func(gsa *allocationv1.GameServerAllocation) {
		if requestCount >= maxBatchBeforeRefresh {
			flush()
		}
		requestCount++

		checkSortKey(gsa)

		if list == nil {
			if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) || gsa.Spec.Scheduling == apis.Packed {
				list = c.allocationCache.ListSortedGameServers(gsa)
			} else {
				list = c.allocationCache.ListSortedGameServersPriorities(gsa)
			}
		} else if gsToReorderIndex >= 0 {
			c.allocationCache.ReorderGameServerAfterAllocation(list, gsToReorderIndex, gsToReorder, gsa.Spec.Priorities, gsa.Spec.Scheduling)
		}
	}

	for {
		select {
		case req := <-c.pendingRequests:
			if req.ctx.Err() != nil {
				c.tryRespondWithError(req, ErrTotalTimeoutExceeded)
				continue
			}

			checkRefreshList(req.gsa)

			foundGs, foundGsIndex, err := findGameServerForAllocation(req.gsa, list)
			if err != nil {
				req.response <- response{request: req, gs: nil, err: err}
				continue
			}

			existingBatch, alreadyAllocated := batchResponsesPerGs[string(foundGs.UID)]
			if !alreadyAllocated {
				if removeErr := c.allocationCache.RemoveGameServer(foundGs); removeErr != nil {
					removeErr = errors.Wrap(removeErr, "error removing gameserver from cache")
					req.response <- response{request: req, gs: nil, err: removeErr}
					list = append(list[:foundGsIndex], list[foundGsIndex+1:]...)
					continue
				}
			}

			gsToReorder = foundGs.DeepCopy()
			gsToReorderIndex = foundGsIndex
			applyErr, counterErrors, listErrors := c.applyAllocationToLocalGameServer(req.gsa.Spec.MetaPatch, gsToReorder, req.gsa)
			if applyErr == nil {
				if alreadyAllocated {
					existingBatch.responses = append(existingBatch.responses, response{request: req, gs: gsToReorder.DeepCopy(), err: nil})
					existingBatch.counterErrors = goErrors.Join(existingBatch.counterErrors, counterErrors)
					existingBatch.listErrors = goErrors.Join(existingBatch.listErrors, listErrors)
					batchResponsesPerGs[string(gsToReorder.UID)] = existingBatch
				} else {
					batchResponsesPerGs[string(gsToReorder.UID)] = batchResponses{
						responses:     []response{{request: req, gs: gsToReorder.DeepCopy(), err: nil}},
						counterErrors: counterErrors,
						listErrors:    listErrors,
					}
				}
			} else {
				req.response <- response{request: req, gs: nil, err: applyErr}
			}

		case <-ctx.Done():
			flush()
			return

		default:
			flush()
			time.Sleep(c.batchWaitTime)
		}
	}
}

// applyAllocationToLocalGameServer patches the GameServer with allocation metadata and sets
// it to Allocated state without persisting to Kubernetes. Counter/List actions are applied
// if FeatureCountsAndLists is enabled.
func (c *Allocator) applyAllocationToLocalGameServer(mp allocationv1.MetaPatch, gs *agonesv1.GameServer, gsa *allocationv1.GameServerAllocation) (applyErr, counterErrors, listErrors error) {
	ts, err := time.Now().MarshalText()
	if err != nil {
		return err, nil, nil
	}
	if gs.ObjectMeta.Annotations == nil {
		gs.ObjectMeta.Annotations = make(map[string]string, 1+len(mp.Annotations))
	}
	gs.ObjectMeta.Annotations[LastAllocatedAnnotationKey] = string(ts)
	gs.Status.State = agonesv1.GameServerStateAllocated

	if mp.Labels != nil {
		if gs.ObjectMeta.Labels == nil {
			gs.ObjectMeta.Labels = make(map[string]string, len(mp.Labels))
		}
		for key, value := range mp.Labels {
			gs.ObjectMeta.Labels[key] = value
		}
	}

	for key, value := range mp.Annotations {
		gs.ObjectMeta.Annotations[key] = value
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if gsa.Spec.Counters != nil {
			for counter, ca := range gsa.Spec.Counters {
				counterErrors = goErrors.Join(counterErrors, ca.CounterActions(counter, gs))
			}
		}
		if gsa.Spec.Lists != nil {
			for list, la := range gsa.Spec.Lists {
				listErrors = goErrors.Join(listErrors, la.ListActions(list, gs))
			}
		}
	}

	return
}
