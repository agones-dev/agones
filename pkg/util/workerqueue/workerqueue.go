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

// Package workerqueue extends client-go's workqueue
// functionality into an opinionated queue + worker model that
// is reusable
package workerqueue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	workFx = time.Second
)

// traceError is a marker type for errors that that should only be logged at a Trace level.
// Useful if you want a Handler to be retried, but not logged at an Error level.
type traceError struct {
	err error
}

// NewTraceError returns a traceError wrapper around an error.
func NewTraceError(err error) error {
	return &traceError{err: err}
}

// Error returns the error string
func (l *traceError) Error() string {
	if l.err == nil {
		return "<nil>"
	}
	return l.err.Error()
}

// isTraceError returns if the error is a trace error or not
func isTraceError(err error) bool {
	cause := errors.Cause(err)
	_, ok := cause.(*traceError)
	return ok
}

// Handler is the handler for processing the work queue
// This is usually a syncronisation handler for a controller or related
type Handler func(context.Context, string) error

// WorkerQueue is an opinionated queue + worker for use
// with controllers and related and processing Kubernetes watched
// events and synchronising resources
//
//nolint:govet // ignore fieldalignment, singleton
type WorkerQueue struct {
	logger  *logrus.Entry
	keyName string
	queue   workqueue.TypedRateLimitingInterface[any]
	// SyncHandler is exported to make testing easier (hack)
	SyncHandler Handler

	mu      sync.Mutex
	workers int
	running int
}

// FastRateLimiter returns a rate limiter without exponential back-off, with specified maximum per-item retry delay.
func FastRateLimiter(maxDelay time.Duration) workqueue.TypedRateLimiter[any] {
	const numFastRetries = 5
	const fastDelay = 200 * time.Millisecond // first few retries up to 'numFastRetries' are fast

	return workqueue.NewTypedItemFastSlowRateLimiter[any](fastDelay, maxDelay, numFastRetries)
}

// ConstantRateLimiter returns a rate limiter without exponential back-off, with constant retry delay.
func ConstantRateLimiter(maxDelay time.Duration) workqueue.TypedRateLimiter[any] {
	const numFastRetries = 0                 // only constant delay
	const fastDelay = 200 * time.Millisecond // not needed

	return workqueue.NewTypedItemFastSlowRateLimiter[any](fastDelay, maxDelay, numFastRetries)
}

// NewWorkerQueue returns a new worker queue for a given name
func NewWorkerQueue(handler Handler, logger *logrus.Entry, keyName logfields.ResourceType, queueName string) *WorkerQueue {
	return NewWorkerQueueWithRateLimiter(handler, logger, keyName, queueName, workqueue.DefaultTypedControllerRateLimiter[any]())
}

// NewWorkerQueueWithRateLimiter returns a new worker queue for a given name and a custom rate limiter.
func NewWorkerQueueWithRateLimiter(handler Handler, logger *logrus.Entry, keyName logfields.ResourceType, queueName string, rateLimiter workqueue.TypedRateLimiter[any]) *WorkerQueue {
	return &WorkerQueue{
		keyName:     string(keyName),
		logger:      logger.WithField("queue", queueName),
		queue:       workqueue.NewNamedRateLimitingQueue(rateLimiter, queueName),
		SyncHandler: handler,
	}
}

// Enqueue puts the name of the runtime.Object in the
// queue to be processed. If you need to send through an
// explicit key, use an cache.ExplicitKey
func (wq *WorkerQueue) Enqueue(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		err = errors.Wrap(err, "Error creating key for object")
		runtime.HandleError(wq.logger.WithField("obj", obj), err)
		return
	}
	wq.logger.WithField(wq.keyName, key).Trace("Enqueuing")
	wq.queue.AddRateLimited(key)
}

// EnqueueImmediately performs Enqueue but without rate-limiting.
// This should be used to continue partially completed work after giving other
// items in the queue a chance of running.
func (wq *WorkerQueue) EnqueueImmediately(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		err = errors.Wrap(err, "Error creating key for object")
		runtime.HandleError(wq.logger.WithField("obj", obj), err)
		return
	}
	wq.logger.WithField(wq.keyName, key).Trace("Enqueuing immediately")
	wq.queue.Add(key)
}

// EnqueueAfter delays an enqueue operation by duration
func (wq *WorkerQueue) EnqueueAfter(obj interface{}, duration time.Duration) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		err = errors.Wrap(err, "Error creating key for object")
		runtime.HandleError(wq.logger.WithField("obj", obj), err)
		return
	}

	wq.logger.WithField(wq.keyName, key).WithField("duration", duration).Trace("Enqueueing after duration")
	wq.queue.AddAfter(key, duration)
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (wq *WorkerQueue) runWorker(ctx context.Context) {
	for wq.processNextWorkItem(ctx) {
	}
}

// processNextWorkItem processes the next work item.
// pretty self explanatory :)
func (wq *WorkerQueue) processNextWorkItem(ctx context.Context) bool {
	obj, quit := wq.queue.Get()
	if quit {
		return false
	}
	defer wq.queue.Done(obj)

	wq.logger.WithField(wq.keyName, obj).Debug("Processing")

	var key string
	var ok bool
	if key, ok = obj.(string); !ok {
		runtime.HandleError(wq.logger.WithField(wq.keyName, obj), errors.Errorf("expected string in queue, but got %T", obj))
		// this is a bad entry, we don't want to reprocess
		wq.queue.Forget(obj)
		return true
	}

	if err := wq.SyncHandler(ctx, key); err != nil {
		// Conflicts are expected, so only show them in debug operations.
		// Also check is traceError for other expected errors.
		if k8serror.IsConflict(errors.Cause(err)) || isTraceError(err) {
			wq.logger.WithField(wq.keyName, obj).Trace(err)
		} else {
			runtime.HandleError(wq.logger.WithField(wq.keyName, obj), err)
		}

		// we don't forget here, because we want this to be retried via the queue
		wq.queue.AddRateLimited(obj)
		return true
	}

	wq.queue.Forget(obj)
	return true
}

// Run the WorkerQueue processing via the Handler. Will block until stop is closed.
// Runs a certain number workers to process the rate limited queue
func (wq *WorkerQueue) Run(ctx context.Context, workers int) {
	wq.setWorkerCount(workers)
	wq.logger.WithField("workers", workers).Info("Starting workers...")
	for i := 0; i < workers; i++ {
		go wq.run(ctx)
	}

	<-ctx.Done()
	wq.logger.Info("...shutting down workers")
	wq.queue.ShutDown()
}

func (wq *WorkerQueue) run(ctx context.Context) {
	wq.inc()
	defer wq.dec()
	wait.Until(func() { wq.runWorker(ctx) }, workFx, ctx.Done())
}

// Healthy reports whether all the worker goroutines are running.
func (wq *WorkerQueue) Healthy() error {
	wq.mu.Lock()
	defer wq.mu.Unlock()
	want := wq.workers
	got := wq.running

	if want != got {
		return fmt.Errorf("want %d worker goroutine(s), got %d", want, got)
	}
	return nil
}

// RunCount reports the number of running worker goroutines started by Run.
func (wq *WorkerQueue) RunCount() int {
	wq.mu.Lock()
	defer wq.mu.Unlock()
	return wq.running
}

func (wq *WorkerQueue) setWorkerCount(n int) {
	wq.mu.Lock()
	defer wq.mu.Unlock()
	wq.workers = n
}

func (wq *WorkerQueue) inc() {
	wq.mu.Lock()
	defer wq.mu.Unlock()
	wq.running++
}

func (wq *WorkerQueue) dec() {
	wq.mu.Lock()
	defer wq.mu.Unlock()
	wq.running--
}
