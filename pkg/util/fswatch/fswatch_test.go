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

package fswatch

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/fsnotify.v1"
)

func TestBatchWatch(t *testing.T) {
	eventChan := make(chan fsnotify.Event)
	errorChan := make(chan error)
	cancelChan := make(chan struct{})
	defer close(cancelChan)

	eventOut := make(chan struct{}, 1) // only allow one event
	errorCount := 0

	go batchWatch(time.Second, eventChan, errorChan, cancelChan, func() {
		select {
		case eventOut <- struct{}{}:
			// capacity
		default:
			assert.FailNow(t, "second event written - did not want")
		}
	}, func(error) {
		errorCount++
	})

	drainEventAndErrors := func(wantErrors int) {
		timeout := time.NewTimer(2 * time.Second)
		select {
		case <-eventOut:
		case <-timeout.C:
			assert.FailNow(t, "no event in 2s")
		}
		assert.Equal(t, wantErrors, errorCount)
	}

	for i := 0; i < 10; i++ {
		eventChan <- fsnotify.Event{}
	}
	drainEventAndErrors(0)

	for i := 0; i < 10; i++ {
		errorChan <- errors.New("some error")
		eventChan <- fsnotify.Event{}
	}
	drainEventAndErrors(10)
}
