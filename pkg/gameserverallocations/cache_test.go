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
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGameServerCacheEntry(t *testing.T) {
	gs1 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1"}}
	gs2 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs2"}}
	gs3 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs3"}}

	cache := gameServerCache{}

	gs, ok := cache.Load("gs1")
	assert.Nil(t, gs)
	assert.False(t, ok)

	cache.Store("gs1", gs1)
	gs, ok = cache.Load("gs1")

	assert.Equal(t, gs, gs1)
	assert.True(t, ok)

	cache.Store("gs2", gs2)
	cache.Store("gs3", gs3)

	count := 0
	cache.Range(func(_ string, _ *agonesv1.GameServer) bool {
		count++
		return count != 2
	})

	assert.Equal(t, 2, count, "Should only process one item")

	cache.Delete("gs1")
	gs, ok = cache.Load("gs1")
	assert.Nil(t, gs)
	assert.False(t, ok)
}

func TestGameServerCacheEntryGeneration(t *testing.T) {
	// Test case 1: If there's no existing value for a key, the incoming GameServer is stored regardless of its Generation.
	cache := gameServerCache{}
	gs1 := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Generation: 1}}

	cache.Store("gs1", gs1)
	gs, ok := cache.Load("gs1")
	assert.True(t, ok)
	assert.Equal(t, gs1, gs)

	// Test case 2: If there's an existing value for a key and the incoming GameServer's Generation is less than the existing one, the existing value is not replaced.
	gs1Lower := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Generation: 0}}
	cache.Store("gs1", gs1Lower)
	gs, ok = cache.Load("gs1")
	assert.True(t, ok)
	assert.Equal(t, int64(1), gs.Generation, "Should not replace with lower Generation")

	// Test case 3: If there's an existing value for a key and the incoming GameServer's Generation is equal to the existing one and ResourceVersion is the same, the existing value is not replaced.
	gs1Equal := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Generation: 1, ResourceVersion: ""}}
	cache.Store("gs1", gs1Equal)
	gs, ok = cache.Load("gs1")
	assert.True(t, ok)
	assert.Equal(t, int64(1), gs.Generation, "Should not replace with equal Generation and same ResourceVersion")

	// Test case 4: If there's an existing value for a key and the incoming GameServer's Generation is greater than the existing one, the existing value is replaced.
	gs1Higher := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Generation: 2, ResourceVersion: "3"}}
	cache.Store("gs1", gs1Higher)
	gs, ok = cache.Load("gs1")
	assert.True(t, ok)
	assert.Equal(t, int64(2), gs.Generation, "Should replace with higher Generation")
	assert.Equal(t, "3", gs.ResourceVersion, "Should replace with higher Generation")

	// Test case 5: If there's an existing value for a key and the incoming GameServer's Generation is equal to the existing one but ResourceVersion is different, the existing value is replaced.
	gs1SameGenDiffRV := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{Name: "gs1", Generation: 2, ResourceVersion: "4"}}
	cache.Store("gs1", gs1SameGenDiffRV)
	gs, ok = cache.Load("gs1")
	assert.True(t, ok)
	assert.Equal(t, int64(2), gs.Generation, "Generation should remain the same")
	assert.Equal(t, "4", gs.ResourceVersion, "Should replace with different ResourceVersion")
}
