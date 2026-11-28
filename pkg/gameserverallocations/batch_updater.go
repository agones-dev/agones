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
	"sync/atomic"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
)

// batchUpdate represents a worker that processes updates for GameServers based on pending allocation requests
type batchUpdater struct {
	allocator *Allocator
	// updateQueue is the workqueue that provides GameServer keys to be updated by this worker
	// This is to ensure they process one GameServer key at a time, avoiding concurrent updates on the same GameServer
	updateQueue workqueue.TypedInterface[string]
	// pending holds the pending allocation requests for each GameServer key
	pending *pendingAllocations
	// staleCandidates indicates whether the allocation candidates are stale and need to be refreshed
	staleCandidates *atomic.Bool
}

// runWorker will process GameServer keys from the updateQueue, handling their pending allocation requests
func (bu *batchUpdater) runWorker(ctx context.Context) {
	for {
		key, shutdown := bu.updateQueue.Get()
		if shutdown {
			return
		}

		// Get the cached GameServer and start processing its pending allocation requests
		cached, _ := bu.allocator.allocationCache.getGameServer(key)
		// Start processing the pending allocation requests for this GameServer key
		reqs := bu.pending.startUpdate(key, cached)
		// Update the GameServer with the pending allocation requests and get the responses
		responses := bu.updateGameServer(ctx, cached, reqs)
		// Finish processing the pending allocation requests for this GameServer key
		bu.pending.finishUpdate(key)
		// Mark the GameServer key as done in the updateQueue
		bu.updateQueue.Done(key)

		// Process the responses for each allocation request, marking candidates as stale if necessary
		for _, res := range responses {
			// If the allocation failed, mark the candidates as stale
			if res.err != nil {
				bu.staleCandidates.Store(true)
			}

			// Send the response back to the requester
			res.request.response <- res
		}
	}
}

// updateGameServer processes the pending allocation requests for a specific GameServer
func (bu *batchUpdater) updateGameServer(ctx context.Context, cached *agonesv1.GameServer, reqs []request) []response {
	if len(reqs) == 0 {
		return nil
	}

	var refused []response
	if cached == nil || cached.IsBeingDeleted() {
		for _, req := range reqs {
			refused = append(refused, response{request: req, err: ErrConflictInGameServerSelection})
		}
		return refused
	}

	toUpdate := cached.DeepCopy()
	var allocated []response
	var counterErrors, listErrors error
	for _, req := range reqs {
		if req.ctx.Err() != nil {
			bu.allocator.tryRespondWithError(req, ErrTotalTimeoutExceeded)
			continue
		}
		if !matchesAllocation(toUpdate, req.gsa) {
			refused = append(refused, response{request: req, err: ErrConflictInGameServerSelection})
			continue
		}

		counterErr, listErr := applyAllocationInMemory(toUpdate, req.gsa, bu.allocator.listMaxCapacity)
		counterErrors = goErrors.Join(counterErrors, counterErr)
		listErrors = goErrors.Join(listErrors, listErr)
		allocated = append(allocated, response{request: req, gs: toUpdate.DeepCopy()})
	}
	if len(allocated) == 0 {
		return refused
	}

	updated, err := bu.allocator.gameServerGetter.GameServers(toUpdate.ObjectMeta.Namespace).Update(ctx, toUpdate, metav1.UpdateOptions{})
	switch {
	case k8serrors.IsNotFound(err):
		_ = bu.allocator.allocationCache.RemoveGameServer(toUpdate)
		err = ErrConflictInGameServerSelection
	case k8serrors.IsConflict(err):
		bu.refreshCachedGameServer(ctx, toUpdate)
		err = goErrors.Join(ErrGameServerUpdateConflict, err)
	}
	if err != nil {
		for _, res := range allocated {
			refused = append(refused, response{request: res.request, err: err})
		}
		return refused
	}

	bu.allocator.allocationCache.AddGameServer(updated)
	if counterErrors != nil {
		bu.allocator.recorder.Event(updated, corev1.EventTypeWarning, "CounterActionError", counterErrors.Error())
	}
	if listErrors != nil {
		bu.allocator.recorder.Event(updated, corev1.EventTypeWarning, "ListActionError", listErrors.Error())
	}
	bu.allocator.recorder.Event(updated, corev1.EventTypeNormal, string(updated.Status.State), "Allocated")

	return append(allocated, refused...)
}

// refreshCachedGameServer refresh the cached gameserver with its latest version, or removes it if it is gone
func (bu *batchUpdater) refreshCachedGameServer(ctx context.Context, gs *agonesv1.GameServer) {
	live, err := bu.allocator.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Get(ctx, gs.ObjectMeta.Name, metav1.GetOptions{})
	switch {
	case err == nil:
		bu.allocator.allocationCache.refreshGameServer(live)
	case k8serrors.IsNotFound(err):
		_ = bu.allocator.allocationCache.RemoveGameServer(gs)
	}
}
