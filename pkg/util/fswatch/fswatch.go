// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fswatch provies Watch(), a utility function to watch a filesystem path.
package fswatch

import (
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

// Watch watches the filesystem path `path`. When anything changes, changes are
// batched for the period `batchFor`, then `processEvent` is called.
//
// Returns a cancel() function to terminate the watch.
func Watch(logger *logrus.Entry, path string, batchFor time.Duration, processEvent func()) (func(), error) {
	logger = logger.WithField("path", path)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	cancelChan := make(chan struct{})
	cancel := func() {
		close(cancelChan)
		_ = watcher.Close()
	}
	if err := watcher.Add(path); err != nil {
		cancel()
		return nil, err
	}

	go batchWatch(batchFor, watcher.Events, watcher.Errors, cancelChan, processEvent, func(error) {
		logger.WithError(err).Errorf("error watching path")
	})
	return cancel, nil
}

// batchWatch: watch for events; when an event occurs, keep draining events for duration `batchFor`, then call processEvent().
// Intended for batching of rapid-fire events where we want to process the batch once, like filesystem update notifications.
func batchWatch(batchFor time.Duration, events chan fsnotify.Event, errors chan error, cancelChan chan struct{}, processEvent func(), onError func(error)) {
	// Pattern shamelessly stolen from https://blog.gopheracademy.com/advent-2013/day-24-channel-buffering-patterns/
	timer := time.NewTimer(0)
	var timerCh <-chan time.Time

	for {
		select {
		// start a timer when an event occurs, otherwise ignore event
		case <-events:
			if timerCh == nil {
				timer.Reset(batchFor)
				timerCh = timer.C
			}

		// on timer, run the batch; nil channels are silently ignored
		case <-timerCh:
			processEvent()
			timerCh = nil

		// handle errors
		case err := <-errors:
			onError(err)

		// on cancel, abort
		case <-cancelChan:
			return
		}
	}
}
