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
	"sync"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
)

// gameserver cache to keep the Ready state gameserver.
//
//nolint:govet // ignore fieldalignment, singleton embedded in AllocationCache
type gameServerCache struct {
	mu    sync.RWMutex
	cache map[string]*agonesv1.GameServer
}

// Store saves the data in the cache.
// If there is already a value for this key, the incoming gs object is ignored
// if its Generation metadata is less than the existing one, or if its Generation
// is equal and its ResourceVersion is the same as the existing one.
// Otherwise, the incoming gs object replaces the existing one.
func (e *gameServerCache) Store(key string, gs *agonesv1.GameServer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cache == nil {
		e.cache = map[string]*agonesv1.GameServer{}
	}

	// Check if there's already a value for this key
	if existingGS, ok := e.cache[key]; ok {
		// If the incoming gs object's Generation is less than the existing one, ignore it, but also check the ResourceVersion
		// saves us a deep copy if we don't need to
		if gs.ObjectMeta.Generation < existingGS.ObjectMeta.Generation || gs.ObjectMeta.ResourceVersion == existingGS.ObjectMeta.ResourceVersion {
			return
		}
	}
	// If the incoming gs object's Generation is greater, store it
	e.cache[key] = gs.DeepCopy()
}

// Delete deletes the data. If it exists returns true.
func (e *gameServerCache) Delete(key string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	ret := false
	if e.cache != nil {
		if _, ok := e.cache[key]; ok {
			delete(e.cache, key)
			ret = true
		}
	}

	return ret
}

// Load returns the data from cache. It return true if the value exists in the cache
func (e *gameServerCache) Load(key string) (*agonesv1.GameServer, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	val, ok := e.cache[key]

	return val, ok
}

// Range extracts data from the cache based on provided function f.
func (e *gameServerCache) Range(f func(key string, gs *agonesv1.GameServer) bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for k, v := range e.cache {
		if !f(k, v) {
			break
		}
	}
}

// Len returns the current length of the cache
func (e *gameServerCache) Len() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.cache)
}
