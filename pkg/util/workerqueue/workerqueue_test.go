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

package workerqueue

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
)

func TestWorkerQueueRun(t *testing.T) {
	t.Parallel()

	received := make(chan string)
	defer close(received)

	syncHandler := func(_ context.Context, name string) error {
		assert.Equal(t, "default/test", name)
		received <- name
		return nil
	}

	wq := NewWorkerQueue(syncHandler, logrus.WithField("source", "test"), "testKey", "test")
	stop := make(chan struct{})
	defer close(stop)

	go wq.Run(context.Background(), 1)

	// no change, should be no value
	select {
	case <-received:
		assert.Fail(t, "should not have received value")
	case <-time.After(1 * time.Second):
	}

	wq.Enqueue(cache.ExplicitKey("default/test"))

	select {
	case <-received:
	case <-time.After(5 * time.Second):
		assert.Fail(t, "should have received value")
	}
}

func TestWorkerQueueHealthy(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	handler := func(context.Context, string) error {
		<-done
		return nil
	}
	wq := NewWorkerQueue(handler, logrus.WithField("source", "test"), "testKey", "test")
	wq.Enqueue(cache.ExplicitKey("default/test"))

	ctx, cancel := context.WithCancel(context.Background())
	go wq.Run(ctx, 1)

	// Yield to the scheduler to ensure the worker queue goroutine can run.
	err := wait.PollUntilContextTimeout(context.Background(), 100*time.Millisecond, 3*time.Second, true, func(_ context.Context) (done bool, err error) {
		if (wq.RunCount() == 1) && wq.Healthy() == nil {
			return true, nil
		}

		return false, nil
	})
	assert.Nil(t, err)

	close(done) // Ensure the handler no longer blocks.
	cancel()    // Stop the worker queue.

	// Yield to the scheduler again to ensure the worker queue goroutine can
	// finish.
	err = wait.PollUntilContextTimeout(context.Background(), 100*time.Millisecond, 3*time.Second, true, func(_ context.Context) (done bool, err error) {
		if (wq.RunCount() == 0) && wq.Healthy() != nil {
			return true, nil
		}

		return false, nil
	})
	assert.Nil(t, err)
}

func TestWorkQueueHealthCheck(t *testing.T) {
	t.Parallel()

	health := healthcheck.NewHandler()
	handler := func(context.Context, string) error {
		return nil
	}
	wq := NewWorkerQueue(handler, logrus.WithField("source", "test"), "testKey", "test")
	health.AddLivenessCheck("test", wq.Healthy)

	server := httptest.NewServer(health)
	defer server.Close()

	const workersCount = 1
	ctx, cancel := context.WithCancel(context.Background())
	go wq.Run(ctx, workersCount)

	// Wait for worker to actually start
	err := wait.PollUntilContextTimeout(context.Background(), 100*time.Millisecond, 5*time.Second, true, func(_ context.Context) (bool, error) {
		rc := wq.RunCount()
		logrus.WithField("runcount", rc).Info("Checking run count before liveness check")
		return rc == workersCount, nil
	})
	assert.Nil(t, err)

	f := func(t *testing.T, url string, status int) {
		// sometimes the http server takes a bit to start up
		err := wait.PollUntilContextTimeout(context.Background(), time.Second, 5*time.Second, true, func(_ context.Context) (done bool, err error) {
			resp, err := http.Get(url)
			assert.Nil(t, err)
			defer resp.Body.Close() // nolint: errcheck

			if status != resp.StatusCode {
				return false, nil
			}

			body, err := io.ReadAll(resp.Body)
			assert.Nil(t, err)
			assert.Equal(t, status, resp.StatusCode)
			assert.Equal(t, []byte("{}\n"), body)

			return true, nil
		})

		assert.Nil(t, err)
	}

	url := server.URL + "/live"
	f(t, url, http.StatusOK)

	cancel()
	// closing can take a short while
	err = wait.PollUntilContextTimeout(context.Background(), time.Second, 5*time.Second, true, func(_ context.Context) (bool, error) {
		rc := wq.RunCount()
		logrus.WithField("runcount", rc).Info("Checking run count")
		return rc == 0, nil
	})
	assert.Nil(t, err)

	// gate
	assert.Error(t, wq.Healthy())
	f(t, url, http.StatusServiceUnavailable)
}

func TestWorkerQueueEnqueueAfter(t *testing.T) {
	t.Parallel()

	updated := make(chan bool)
	syncHandler := func(_ context.Context, _ string) error {
		updated <- true
		return nil
	}
	wq := NewWorkerQueue(syncHandler, logrus.WithField("source", "test"), "testKey", "test")
	stop := make(chan struct{})
	defer close(stop)

	go wq.Run(context.Background(), 1)

	wq.EnqueueAfter(cache.ExplicitKey("default/test"), 2*time.Second)

	select {
	case <-updated:
		assert.FailNow(t, "should not be a result in queue yet")
	case <-time.After(time.Second):
	}

	select {
	case <-updated:
	case <-time.After(2 * time.Second):
		assert.Fail(t, "should have got a queue'd message by now")
	}
}

func TestDebugError(t *testing.T) {
	err := errors.New("not a debug error")
	assert.False(t, isTraceError(err))

	err = NewTraceError(err)
	assert.True(t, isTraceError(err))
	assert.EqualError(t, err, "not a debug error")

	err = NewTraceError(nil)
	assert.True(t, isTraceError(err))
	assert.EqualError(t, err, "<nil>")
}
