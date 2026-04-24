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

package gameserversets

import (
	"sort"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var deletionTime = metav1.Now()

func TestGameServerStateCache(t *testing.T) {
	var cache gameServerStateCache
	gsSet1 := &agonesv1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "set-1",
			Namespace: "ns1",
		},
	}
	gsSet1b := &agonesv1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "set-1",
			Namespace: "ns2",
		},
	}
	gsSet2 := &agonesv1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "set-2",
			Namespace: "ns1",
		},
	}

	entry1 := cache.forGameServerSet(gsSet1)
	if got, want := entry1, cache.forGameServerSet(gsSet1); got != want {
		t.Errorf("unexpectedly different entries for the same set")
	}
	if got, want := entry1, cache.forGameServerSet(gsSet1b); got == want {
		t.Errorf("unexpectedly same entries for different sets 1 and 1b")
	}
	if got, want := entry1, cache.forGameServerSet(gsSet2); got == want {
		t.Errorf("unexpectedly same entries for different sets 1 and 2")
	}
	if got, want := entry1, cache.forGameServerSet(gsSet1); got != want {
		t.Errorf("unexpectedly different entries for the same set")
	}
	cache.deleteGameServerSet(gsSet1)
	if got, want := entry1, cache.forGameServerSet(gsSet1); got == want {
		t.Errorf("unexpectedly same entries for sets 1  before and after deletion")
	}
}

func TestGameServerSetCacheEntry(t *testing.T) {
	gs1 := makeGameServer("gs-1")
	gs2 := makeGameServer("gs-2")

	cases := []struct {
		desc                     string
		setup                    func(c *gameServerSetCacheEntry)
		list                     []*agonesv1.GameServer
		expected                 []*agonesv1.GameServer
		expectedPendingCreations int
		expectedPendingDeletions int
	}{
		{
			desc:     "EmptyList",
			setup:    func(_ *gameServerSetCacheEntry) {},
			list:     nil,
			expected: nil,
		},
		{
			desc:                     "LocallyAddedGameServerNotInServerResults",
			setup:                    func(c *gameServerSetCacheEntry) { c.created(gs1) },
			list:                     nil,
			expected:                 []*agonesv1.GameServer{gs1},
			expectedPendingCreations: 1,
		},
		{
			desc:                     "LocallyAddedGameServerAnotherOneFoundOnServer",
			setup:                    func(c *gameServerSetCacheEntry) { c.created(gs1) },
			list:                     []*agonesv1.GameServer{gs2},
			expected:                 []*agonesv1.GameServer{gs1, gs2},
			expectedPendingCreations: 1,
		},
		{
			desc:     "LocallyAddedGameServerAlsoFoundOnServer",
			setup:    func(c *gameServerSetCacheEntry) { c.created(gs1) },
			list:     []*agonesv1.GameServer{gs1},
			expected: []*agonesv1.GameServer{gs1},
		},
		{
			desc:                     "LocallyDeletedStillFoundOnServer",
			setup:                    func(c *gameServerSetCacheEntry) { c.deleted(gs1) },
			list:                     []*agonesv1.GameServer{gs1},
			expected:                 []*agonesv1.GameServer{deleted(gs1)},
			expectedPendingDeletions: 1,
		},
		{
			desc:     "LocallyDeletedNotFoundOnServer",
			setup:    func(c *gameServerSetCacheEntry) { c.deleted(gs1) },
			list:     []*agonesv1.GameServer{},
			expected: []*agonesv1.GameServer{},
		},
		{
			desc: "LocallyCreatedAndDeletedFoundOnServer",
			setup: func(c *gameServerSetCacheEntry) {
				c.created(gs1)
				c.deleted(gs1)
			},
			list:                     []*agonesv1.GameServer{gs1},
			expected:                 []*agonesv1.GameServer{deleted(gs1)},
			expectedPendingDeletions: 1,
		},
		{
			desc: "LocallyCreatedAndDeletedFoundDeletedOnServer",
			setup: func(c *gameServerSetCacheEntry) {
				c.created(gs1)
				c.deleted(gs1)
			},
			list:     []*agonesv1.GameServer{deleted(gs1)},
			expected: []*agonesv1.GameServer{deleted(gs1)},
		},
		{
			desc: "LocallyCreatedAndDeletedNotFoundOnServer",
			setup: func(c *gameServerSetCacheEntry) {
				c.created(gs1)
				c.deleted(gs1)
			},
			list:     []*agonesv1.GameServer{},
			expected: []*agonesv1.GameServer{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			entry := &gameServerSetCacheEntry{}

			tc.setup(entry)
			result := entry.reconcileWithUpdatedServerList(tc.list)
			assert.Equal(t, sortedStatusInfo(result), sortedStatusInfo(tc.expected))

			if got, want := len(entry.pendingCreation), tc.expectedPendingCreations; got != want {
				t.Errorf("unexpected # of pending creations %v, wanted %v", got, want)
			}
			if got, want := len(entry.pendingDeletion), tc.expectedPendingDeletions; got != want {
				t.Errorf("unexpected # of pending deletions %v, wanted %v", got, want)
			}

			result2 := entry.reconcileWithUpdatedServerList(result)
			assert.Equal(t, sortedStatusInfo(result2), sortedStatusInfo(result))

			// now both pending creations and deletions must be zero
			if got, want := len(entry.pendingCreation), 0; got != want {
				t.Errorf("unexpected # of pending creations %v, wanted %v", got, want)
			}
			if got, want := len(entry.pendingDeletion), 0; got != want {
				t.Errorf("unexpected # of pending deletions %v, wanted %v", got, want)
			}
		})
	}
}

func makeGameServer(s string) *agonesv1.GameServer {
	return &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: s,
		},
	}
}

func deleted(gs *agonesv1.GameServer) *agonesv1.GameServer {
	gs2 := gs.DeepCopy()
	gs2.ObjectMeta.DeletionTimestamp = &deletionTime
	return gs2
}

type gameServerStatusInfo struct {
	name    string
	status  string
	deleted bool
}

func sortedStatusInfo(list []*agonesv1.GameServer) []gameServerStatusInfo {
	var result []gameServerStatusInfo
	for _, gs := range list {
		result = append(result, gameServerStatusInfo{
			gs.Name,
			string(gs.Status.State),
			gs.DeletionTimestamp != nil,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].name < result[j].name
	})
	return result
}
