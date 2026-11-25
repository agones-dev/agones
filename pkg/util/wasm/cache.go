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

// Package wasm loads and runs Wasm plugins for Agones.
package wasm

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	pluginsv1 "agones.dev/agones/pkg/apis/plugins/v1"
	"agones.dev/agones/pkg/util/errors"
	extism "github.com/extism/go-sdk"
	lru "github.com/hashicorp/golang-lru"
	"github.com/tetratelabs/wazero"
	"golang.org/x/sync/singleflight"
)

const (
	// fetchTimeout is how long to wait when downloading a Wasm module.
	fetchTimeout = 15 * time.Second
	// callTimeoutMilliseconds is how long a single call to a Wasm function can run for, including
	// creating the instance that it runs on.
	callTimeoutMilliseconds = 5000
)

// Cache is a fixed size, least recently used cache of compiled Wasm modules, keyed by where the
// module was loaded from, and its optional hash. Modules are downloaded and compiled the first time
// they are called, and are closed when they are evicted from the cache.
type Cache struct {
	modules *lru.Cache
	// runtimeConfig is the wazero configuration that modules are compiled with. Nil uses the default.
	runtimeConfig wazero.RuntimeConfig
	errs          *errors.Errors
	// loading ensures that each module is only downloaded and compiled once at a time, while
	// different modules load in parallel.
	loading singleflight.Group
}

// module is a compiled Wasm module that is stored in the Cache.
type module struct {
	compiled *extism.CompiledPlugin
	// mu is read locked while the module is being called, so that it is not closed mid-call.
	mu     sync.RWMutex
	closed bool
}

// NewCache returns a new Cache that holds at most size compiled Wasm modules.
func NewCache(size int) (*Cache, error) {
	c := &Cache{}
	c.errs = errors.FromStruct(c)

	modules, err := lru.NewWithEvict(size, func(_, value any) {
		// close in the background, as the module may still be mid-call, and we don't want to block the cache.
		go value.(*module).close()
	})
	if err != nil {
		return nil, c.errs.Wrap(err, "could not create wasm module cache")
	}
	c.modules = modules

	return c, nil
}

// Call calls the Function of the Wasm module in the WasmPolicy with the given input, and returns its output.
// The module is downloaded and compiled if it is not already in the cache.
func (c *Cache) Call(ctx context.Context, wp *pluginsv1.WasmPolicy, input []byte) ([]byte, error) {
	if wp == nil || wp.From.URL == nil {
		return nil, c.errs.New("wasm policy does not have a module to load")
	}

	for {
		// a cached module does not wait on ctx, so check it here, so that reloading an evicted module
		// stops once the caller has given up.
		if err := ctx.Err(); err != nil {
			return nil, c.errs.Wrap(err, "stopped calling wasm module")
		}

		m, err := c.load(ctx, wp)
		if err != nil {
			return nil, err
		}

		output, closed, err := c.call(ctx, m, wp, input)
		// if the module was evicted and closed between loading it and calling it, load it again.
		if closed {
			continue
		}
		return output, err
	}
}

// Len returns the number of compiled Wasm modules in the cache.
func (c *Cache) Len() int {
	return c.modules.Len()
}

// call calls the Function on a new instance of the module. Returns true if the module has been closed.
func (c *Cache) call(ctx context.Context, m *module, wp *pluginsv1.WasmPolicy, input []byte) ([]byte, bool, error) {
	// close is the only writer, so if the lock is held or waited on, the module has been evicted. Don't
	// wait for close, as it waits for the calls in progress to finish.
	if !m.mu.TryRLock() {
		return nil, true, nil
	}
	defer m.mu.RUnlock()

	if m.closed {
		return nil, true, nil
	}

	// creating an instance runs the module's start section, so the timeout has to cover it, as well as the call.
	ctx, cancel := context.WithTimeout(ctx, callTimeoutMilliseconds*time.Millisecond)
	defer cancel()

	// instances are not thread safe, so each call gets its own.
	instance, err := m.compiled.Instance(ctx, extism.PluginInstanceConfig{})
	if err != nil {
		return nil, false, c.errs.Wrap(err, "could not create instance of wasm module")
	}
	defer instance.Close(ctx) //nolint:errcheck
	instance.Config = wp.Config

	if !instance.FunctionExists(wp.Function) {
		return nil, false, c.errs.Errorf("wasm function %q does not exist", wp.Function)
	}

	_, output, err := instance.CallWithContext(ctx, wp.Function, input)
	if err != nil {
		return nil, false, c.errs.Wrapf(err, "failed to call wasm function %q", wp.Function)
	}

	return output, false, nil
}

// load returns the compiled module for the WasmPolicy from the cache,
// downloading and compiling it first if it is not there.
func (c *Cache) load(ctx context.Context, wp *pluginsv1.WasmPolicy) (*module, error) {
	u, err := wp.From.URL.BuildURL()
	if err != nil {
		return nil, err
	}
	// quoted, so that a separator in the URL or hash can't make two policies share a key.
	key := fmt.Sprintf("%q|%q|%x", u.String(), wp.Hash, sha256.Sum256(wp.From.URL.CABundle))

	if m, ok := c.modules.Get(key); ok {
		return m.(*module), nil
	}

	// the load is shared by every call for this module, so it is not cancelled when this call is.
	loadCtx := context.WithoutCancel(ctx)
	ch := c.loading.DoChan(key, func() (any, error) {
		// check again, in case it finished loading after the check above.
		if m, ok := c.modules.Get(key); ok {
			return m, nil
		}

		m, err := c.compile(loadCtx, wp, u)
		if err != nil {
			return nil, err
		}
		c.modules.Add(key, m)
		return m, nil
	})

	select {
	case res := <-ch:
		if res.Err != nil {
			return nil, res.Err
		}
		return res.Val.(*module), nil
	case <-ctx.Done():
		return nil, c.errs.Wrapf(ctx.Err(), "stopped waiting for wasm module from %s", u.String())
	}
}

// compile downloads the Wasm module for the WasmPolicy from u, and compiles it.
func (c *Cache) compile(ctx context.Context, wp *pluginsv1.WasmPolicy, u *url.URL) (*module, error) {
	client, err := wp.From.URL.HTTPClient(fetchTimeout)
	if err != nil {
		return nil, err
	}
	defer client.CloseIdleConnections()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, c.errs.Wrapf(err, "failed to build request for wasm module at %s", u.String())
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, c.errs.Wrapf(err, "failed to fetch wasm module from %s", u.String())
	}
	defer res.Body.Close() //nolint:errcheck

	if res.StatusCode != http.StatusOK {
		return nil, c.errs.Errorf("bad status code %d fetching wasm module from %s", res.StatusCode, u.String())
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, c.errs.Wrapf(err, "failed to read wasm module from %s", u.String())
	}

	manifest := extism.Manifest{
		Wasm:    []extism.Wasm{extism.WasmData{Data: b, Hash: wp.Hash}},
		Timeout: callTimeoutMilliseconds,
	}

	// the compiled module outlives this request, so it is not tied to its context.
	compiled, err := extism.NewCompiledPlugin(context.Background(), manifest, extism.PluginConfig{EnableWasi: true, RuntimeConfig: c.runtimeConfig}, []extism.HostFunction{})
	if err != nil {
		return nil, c.errs.Wrapf(err, "failed to compile wasm module from %s", u.String())
	}

	return &module{compiled: compiled}, nil
}

// close closes the compiled module, once any calls that are in progress have completed.
func (m *module) close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	_ = m.compiled.Close(context.Background())
}
