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

// Package https provides HTTPS helpers.
package https

import (
	"io"
	"net/http"

	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ErrorHandlerFunc is a http handler that can return an error
// for standard logging and a 500 response
type ErrorHandlerFunc func(http.ResponseWriter, *http.Request) error

// FourZeroFour is the standard 404 handler.
func FourZeroFour(logger *logrus.Entry, w http.ResponseWriter, r *http.Request) {
	f := ErrorHTTPHandler(logger, func(_ http.ResponseWriter, _ *http.Request) error {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return errors.Wrap(err, "error in default handler")
		}
		defer r.Body.Close() // nolint: errcheck

		LogRequest(logger, r).WithField("body", string(body)).Warn("404")
		http.NotFound(w, r)

		return nil
	})

	f(w, r)
}

// ErrorHTTPHandler is a conversion function that sets up a http.StatusInternalServerError
// if an error is returned
func ErrorHTTPHandler(logger *logrus.Entry, f ErrorHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			runtime.HandleError(LogRequest(logger, r), err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// LogRequest logs all the JSON parsable fields in a request
// as otherwise, the request is not marshable
func LogRequest(logger *logrus.Entry, r *http.Request) *logrus.Entry {
	return logger.WithField("method", r.Method).
		WithField("url", r.URL).
		WithField("host", r.Host).
		WithField("headers", r.Header).
		WithField("requestURI", r.RequestURI)
}
