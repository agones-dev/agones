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

// Package runtime handles runtime errors
// Wraps and reconfigures functionality in apimachinery/pkg/runtime
package runtime

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/runtime"
)

func TestHandleError(t *testing.T) {
	old := runtime.ErrorHandlers
	defer func() { runtime.ErrorHandlers = old }()
	var result error
	runtime.ErrorHandlers = []runtime.ErrorHandler{
		func(_ context.Context, err error, _ string, _ ...interface{}) {
			result = err
		},
	}
	HandleError(nil, nil)
	assert.Nil(t, result, "No Errors for now")

	err := fmt.Errorf("test")
	// test nil logger
	logger := NewLoggerWithSource("test")
	HandleError(logger.WithError(err), err)
	if result != err {
		t.Errorf("did not receive custom handler")
	}
}
