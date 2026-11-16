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
	"slices"
	"sync"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
)

const (
	unhealthyBackoffBase = 10 * time.Second
	unhealthyBackoffMax  = 5 * time.Minute
)

// unhealthyBackoff tracks, per GameServerSet, how long to keep a GameServer that became Unhealthy
// before Ready before it is replaced. The state is in memory only.
type unhealthyBackoff struct {
	entries  map[string]*unhealthyBackoffEntry
	mu       sync.Mutex
	maxDelay time.Duration
}

type unhealthyBackoffEntry struct {
	usableAtLastFailure map[string]bool
	lastStep            time.Time
	lastFailure         time.Time
	delay               time.Duration
}

func newUnhealthyBackoff(maxDelay time.Duration) *unhealthyBackoff {
	return &unhealthyBackoff{maxDelay: maxDelay, entries: map[string]*unhealthyBackoffEntry{}}
}

// observe updates the backoff state of the GameServerSet and returns the current delay.
// There is no delay for the first unhealthyBackoffBase after the first failure. After that the delay starts at
// unhealthyBackoffBase and doubles, at most once per delay period, up to maxDelay.
func (b *unhealthyBackoff) observe(gsSet *agonesv1.GameServerSet, list []*agonesv1.GameServer, now time.Time) time.Duration {
	failed := slices.ContainsFunc(list, func(gs *agonesv1.GameServer) bool {
		_, ok := unhealthyBeforeReadyAt(gs)
		return ok && !gs.IsBeingDeleted()
	})

	b.mu.Lock()
	defer b.mu.Unlock()

	key := gsSet.ObjectMeta.Namespace + "/" + gsSet.ObjectMeta.Name
	entry := b.entries[key]
	if entry == nil && !failed {
		return 0
	}

	usable := map[string]bool{}
	var latestFailure time.Time
	for _, gs := range list {
		if gs.IsBeingDeleted() {
			continue
		}
		if gs.IsActive() {
			usable[gs.ObjectMeta.Name] = true
		}
		if failedAt, ok := unhealthyBeforeReadyAt(gs); ok && failedAt.After(latestFailure) {
			latestFailure = failedAt
		}
	}

	if entry != nil {
		recovered := false
		for name := range usable {
			if !entry.usableAtLastFailure[name] {
				recovered = true
				break
			}
		}
		resetAfter := 2 * b.maxDelay
		if recovered || (latestFailure.IsZero() && now.Sub(entry.lastFailure) > resetAfter) {
			delete(b.entries, key)
			return 0
		}
		if latestFailure.Sub(entry.lastFailure) > resetAfter {
			// Start again, with this as the first failure.
			delete(b.entries, key)
			entry = nil
		}
	}

	if entry == nil {
		entry = &unhealthyBackoffEntry{lastStep: now}
		b.entries[key] = entry
	}
	if latestFailure.After(entry.lastFailure) {
		if now.Sub(entry.lastStep) >= max(entry.delay, unhealthyBackoffBase) {
			entry.delay = min(max(entry.delay*2, unhealthyBackoffBase), b.maxDelay)
			entry.lastStep = now
		}
		entry.lastFailure = latestFailure
		entry.usableAtLastFailure = usable
	}
	return entry.delay
}

func (b *unhealthyBackoff) deleteGameServerSet(gsSet *agonesv1.GameServerSet) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.entries, gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name)
}

func unhealthyBeforeReadyAt(gs *agonesv1.GameServer) (time.Time, bool) {
	if gs.Status.State != agonesv1.GameServerStateUnhealthy {
		return time.Time{}, false
	}
	failedAt, err := time.Parse(time.RFC3339, gs.ObjectMeta.Annotations[agonesv1.GameServerUnhealthyBeforeReadyAtAnnotation])
	if err != nil {
		return time.Time{}, false
	}
	return failedAt, true
}

// unhealthyDelayRemaining returns how much longer to keep the GameServer. Zero or less means replace it now.
func unhealthyDelayRemaining(gs *agonesv1.GameServer, delay time.Duration, now time.Time) time.Duration {
	failedAt, ok := unhealthyBeforeReadyAt(gs)
	if !ok || delay <= 0 {
		return 0
	}
	return failedAt.Add(delay).Sub(now)
}

// nextUnhealthyDelayEnd returns the time until the first kept GameServer is due to be replaced.
func nextUnhealthyDelayEnd(list []*agonesv1.GameServer, delay time.Duration, now time.Time) (time.Duration, bool) {
	var next time.Duration
	for _, gs := range list {
		if gs.IsBeingDeleted() {
			continue
		}
		if remaining := unhealthyDelayRemaining(gs, delay, now); remaining > 0 && (next == 0 || remaining < next) {
			next = remaining
		}
	}
	return next, next > 0
}
