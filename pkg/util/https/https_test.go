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

package https

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestFourZeroFour(t *testing.T) {
	b := bytes.NewBuffer(nil)
	r, err := http.NewRequest(http.MethodGet, "/", b)
	assert.NoError(t, err)
	w := httptest.NewRecorder()

	l := logrus.WithField("source", "test")

	FourZeroFour(l, w, r)

	resp := w.Result()
	defer resp.Body.Close() // nolint: errcheck
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
