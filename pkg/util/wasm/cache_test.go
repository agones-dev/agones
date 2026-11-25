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

package wasm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pluginsv1 "agones.dev/agones/pkg/apis/plugins/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tetratelabs/wazero"
)

// compilationCache is shared by every Cache in these tests, so the example plugin is only compiled
// once, rather than every time a test loads it.
var compilationCache = wazero.NewCompilationCache()

// newCache returns a Cache that shares compiled code with every other Cache in these tests.
func newCache(t *testing.T, size int) *Cache {
	t.Helper()

	c, err := NewCache(size)
	require.NoError(t, err)
	c.runtimeConfig = wazero.NewRuntimeConfig().WithCompilationCache(compilationCache)
	return c
}

const gameServerJSON = `{"metadata":{"name":"gs"},"status":{"state":"Allocated","lists":{"freeSessions":{"capacity":3,"values":["session-1","session-2","session-3"]}}}}`

// wasmServer serves the allocation-wasm example plugin, and counts how many times it has been requested.
func wasmServer(t *testing.T) (*httptest.Server, *atomic.Int32, string) {
	t.Helper()
	return serveWasm(t, func(http.ResponseWriter, *http.Request) {})
}

// serveWasm is wasmServer, with before called at the start of every request.
func serveWasm(t *testing.T, before http.HandlerFunc) (*httptest.Server, *atomic.Int32, string) {
	t.Helper()

	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "examples", "allocation-wasm"))
	require.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(dir, "plugin.wasm"))
	require.NoError(t, err, "wasm plugin not found in %s", dir)
	sum := sha256.Sum256(b)

	requests := &atomic.Int32{}
	fs := http.FileServer(http.Dir(dir))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		before(w, r)
		requests.Add(1)
		fs.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)

	return srv, requests, hex.EncodeToString(sum[:])
}

// gatedWasmServer serves the allocation-wasm example plugin, but holds every request to paths
// under /gated/ until release is called. Each held request is signalled on requested.
func gatedWasmServer(t *testing.T) (srv *httptest.Server, requests *atomic.Int32, requested <-chan struct{}, release func()) {
	t.Helper()

	gate := make(chan struct{})
	once := sync.Once{}
	release = func() { once.Do(func() { close(gate) }) }
	signal := make(chan struct{}, 10)

	srv, requests, _ = serveWasm(t, func(_ http.ResponseWriter, r *http.Request) {
		if path, ok := strings.CutPrefix(r.URL.Path, "/gated"); ok {
			signal <- struct{}{}
			<-gate
			r.URL.Path = path
		}
	})
	// cleanups run last in first out, so held requests are released before the server is closed.
	t.Cleanup(release)

	return srv, requests, signal, release
}

func wasmPolicy(url, function string) *pluginsv1.WasmPolicy {
	return &pluginsv1.WasmPolicy{
		Function: function,
		Config:   map[string]string{"list": "freeSessions"},
		From:     pluginsv1.WasmFrom{URL: &pluginsv1.URLConfiguration{URL: &url}},
	}
}

// annotations returns the annotations of the GameServer JSON that is returned from the plugin.
func annotations(t *testing.T, output []byte) map[string]string {
	t.Helper()

	gs := struct {
		Metadata struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
	}{}
	require.NoError(t, json.Unmarshal(output, &gs))
	return gs.Metadata.Annotations
}

func TestCacheCall(t *testing.T) {
	t.Parallel()

	srv, requests, hash := wasmServer(t)
	url := srv.URL + "/plugin.wasm"
	ctx := context.Background()

	c := newCache(t, 10)

	output, err := c.Call(ctx, wasmPolicy(url, "allocate"), []byte(gameServerJSON))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"session": "session-1"}, annotations(t, output))
	assert.Equal(t, 1, c.Len())

	// config is applied on each call, and the module is only downloaded the once.
	wp := wasmPolicy(url, "allocate")
	wp.Config["annotation"] = "other"
	output, err = c.Call(ctx, wp, []byte(gameServerJSON))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"other": "session-1"}, annotations(t, output))

	output, err = c.Call(ctx, wasmPolicy(url, "allocateNone"), []byte(gameServerJSON))
	require.NoError(t, err)
	assert.Empty(t, annotations(t, output))
	assert.Contains(t, string(output), `"values":["session-1","session-2","session-3"]`)
	assert.Equal(t, 1, c.Len())
	assert.Equal(t, int32(1), requests.Load())

	// a hash is a different module in the cache.
	wp = wasmPolicy(url, "allocate")
	wp.Hash = hash
	_, err = c.Call(ctx, wp, []byte(gameServerJSON))
	require.NoError(t, err)
	assert.Equal(t, 2, c.Len())
	assert.Equal(t, int32(2), requests.Load())
}

func TestCacheCallErrors(t *testing.T) {
	t.Parallel()

	srv, _, hash := wasmServer(t)
	url := srv.URL + "/plugin.wasm"
	ctx := context.Background()

	c := newCache(t, 10)

	_, err := NewCache(0)
	require.Error(t, err)

	_, err = c.Call(ctx, nil, []byte(gameServerJSON))
	require.ErrorContains(t, err, "wasm policy does not have a module to load")

	_, err = c.Call(ctx, wasmPolicy(srv.URL+"/missing.wasm", "allocate"), []byte(gameServerJSON))
	require.ErrorContains(t, err, "bad status code 404")

	_, err = c.Call(ctx, wasmPolicy(srv.URL+"/main.go", "allocate"), []byte(gameServerJSON))
	require.ErrorContains(t, err, "failed to compile wasm module")

	wp := wasmPolicy(url, "allocate")
	wp.Hash = "00" + hash[2:]
	_, err = c.Call(ctx, wp, []byte(gameServerJSON))
	require.ErrorContains(t, err, "failed to compile wasm module")
	assert.Equal(t, 0, c.Len())

	_, err = c.Call(ctx, wasmPolicy(url, "missing"), []byte(gameServerJSON))
	require.ErrorContains(t, err, `wasm function "missing" does not exist`)

	// a separator in the URL or hash does not make the policies share a module, so the bad hash fails.
	_, err = c.Call(ctx, wasmPolicy(url+"?x=a|", "allocate"), []byte(gameServerJSON))
	require.NoError(t, err)
	wp = wasmPolicy(url+"?x=a", "allocate")
	wp.Hash = "|"
	_, err = c.Call(ctx, wp, []byte(gameServerJSON))
	require.ErrorContains(t, err, "failed to compile wasm module")

	// the plugin returns an error, as the list is not there.
	wp = wasmPolicy(url, "allocate")
	wp.Config["list"] = "missing"
	_, err = c.Call(ctx, wp, []byte(gameServerJSON))
	require.ErrorContains(t, err, `failed to call wasm function "allocate"`)
}

func TestCacheEviction(t *testing.T) {
	t.Parallel()

	srv, requests, _ := wasmServer(t)
	first := srv.URL + "/plugin.wasm"
	second := srv.URL + "/plugin.wasm?second"
	ctx := context.Background()

	c := newCache(t, 1)

	for i, url := range []string{first, second, first} {
		output, err := c.Call(ctx, wasmPolicy(url, "allocate"), []byte(gameServerJSON))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"session": "session-1"}, annotations(t, output))
		assert.Equal(t, 1, c.Len())
		assert.Equal(t, int32(i+1), requests.Load())
	}
}

func TestCacheConcurrentCalls(t *testing.T) {
	t.Parallel()

	srv, _, _ := wasmServer(t)
	urls := []string{srv.URL + "/plugin.wasm", srv.URL + "/plugin.wasm?second"}
	ctx := context.Background()

	// only room for one module, so modules are evicted while they are being called.
	c := newCache(t, 1)

	wg := sync.WaitGroup{}
	for i := range 8 {
		wg.Go(func() {
			output, err := c.Call(ctx, wasmPolicy(urls[i%len(urls)], "allocate"), []byte(gameServerJSON))
			if assert.NoError(t, err) {
				assert.Contains(t, string(output), `"session":"session-1"`)
			}
		})
	}
	wg.Wait()
}

func TestCacheSlowModuleDoesNotBlockOthers(t *testing.T) {
	t.Parallel()

	srv, requests, requested, release := gatedWasmServer(t)
	ctx := context.Background()

	c := newCache(t, 10)

	wg := sync.WaitGroup{}
	wg.Go(func() {
		_, err := c.Call(ctx, wasmPolicy(srv.URL+"/gated/plugin.wasm", "allocate"), []byte(gameServerJSON))
		assert.NoError(t, err)
	})
	<-requested

	wg.Go(func() {
		output, err := c.Call(ctx, wasmPolicy(srv.URL+"/plugin.wasm", "allocate"), []byte(gameServerJSON))
		if assert.NoError(t, err) {
			assert.Equal(t, map[string]string{"session": "session-1"}, annotations(t, output))
		}
	})

	// another module is downloaded while the slow one is still downloading. Held requests are not
	// counted until they are released. If loads were serialised, this request would not be made
	// until the slow download timed out.
	require.Eventually(t, func() bool { return requests.Load() == 1 }, 5*time.Second, 10*time.Millisecond)

	release()
	wg.Wait()
	assert.Equal(t, 2, c.Len())
}

func TestCacheConcurrentLoadsDownloadOnce(t *testing.T) {
	t.Parallel()

	srv, requests, requested, release := gatedWasmServer(t)
	url := srv.URL + "/gated/plugin.wasm"
	ctx := context.Background()

	c := newCache(t, 10)

	wg := sync.WaitGroup{}
	for range 8 {
		wg.Go(func() {
			output, err := c.Call(ctx, wasmPolicy(url, "allocate"), []byte(gameServerJSON))
			if assert.NoError(t, err) {
				assert.Contains(t, string(output), `"session":"session-1"`)
			}
		})
	}
	<-requested
	release()
	wg.Wait()

	assert.Equal(t, int32(1), requests.Load())
	assert.Equal(t, 1, c.Len())
}

func TestCacheCallCancelledWhileLoading(t *testing.T) {
	t.Parallel()

	srv, requests, requested, release := gatedWasmServer(t)
	url := srv.URL + "/gated/plugin.wasm"

	c := newCache(t, 10)

	// the first call starts the load, and gives up waiting for it.
	first, cancel := context.WithCancel(context.Background())
	firstDone := make(chan error, 1)
	go func() {
		_, err := c.Call(first, wasmPolicy(url, "allocate"), []byte(gameServerJSON))
		firstDone <- err
	}()
	<-requested
	cancel()
	err := <-firstDone
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "stopped waiting for wasm module")

	// a call that waits on the same load stops waiting when its context is done.
	waiting, cancelWaiting := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelWaiting()
	_, err = c.Call(waiting, wasmPolicy(url, "allocate"), []byte(gameServerJSON))
	require.ErrorIs(t, err, context.DeadlineExceeded)

	// the load carries on, even though the call that started it was cancelled.
	release()
	_, err = c.Call(context.Background(), wasmPolicy(url, "allocate"), []byte(gameServerJSON))
	require.NoError(t, err)
	assert.Equal(t, int32(1), requests.Load())
	assert.Equal(t, 1, c.Len())
}

func TestCacheCallContextDone(t *testing.T) {
	t.Parallel()

	srv, requests, _ := wasmServer(t)
	url := srv.URL + "/plugin.wasm"

	c := newCache(t, 10)

	_, err := c.Call(context.Background(), wasmPolicy(url, "allocate"), []byte(gameServerJSON))
	require.NoError(t, err)

	// the module is cached, so the call would not otherwise wait on the context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.Call(ctx, wasmPolicy(url, "allocate"), []byte(gameServerJSON))
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorContains(t, err, "stopped calling wasm module")
	assert.Equal(t, int32(1), requests.Load())
}

func TestCacheCallTimeoutCoversStartSection(t *testing.T) {
	t.Parallel()

	// a module whose start section loops forever.
	loop := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, // magic and version
		0x01, 0x04, 0x01, 0x60, 0x00, 0x00, // type section: func() -> ()
		0x03, 0x02, 0x01, 0x00, // function section: one function of type 0
		0x08, 0x01, 0x00, // start section: function 0
		0x0a, 0x09, 0x01, 0x07, 0x00, 0x03, 0x40, 0x0c, 0x00, 0x0b, 0x0b, // code section: loop { br 0 }
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(loop)
	}))
	t.Cleanup(srv.Close)

	c := newCache(t, 10)

	done := make(chan error, 1)
	go func() {
		_, err := c.Call(context.Background(), wasmPolicy(srv.URL+"/loop.wasm", "allocate"), []byte(gameServerJSON))
		done <- err
	}()

	select {
	case err := <-done:
		require.ErrorContains(t, err, "could not create instance of wasm module")
	case <-time.After(time.Minute):
		require.Fail(t, "creating an instance of the module did not time out")
	}
}
