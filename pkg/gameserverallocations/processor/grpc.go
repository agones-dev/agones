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

package processor

import (
	"net/http"

	"agones.dev/agones/pkg/util/runtime"
	"google.golang.org/grpc/codes"
)

var grpcLogger = runtime.NewLoggerWithSource("processor")

// GRPCCodeFromHTTPStatus converts an HTTP status code to the corresponding gRPC status code
func GRPCCodeFromHTTPStatus(httpStatusCode int) codes.Code {
	switch httpStatusCode {
	case http.StatusOK:
		return codes.OK
	case 499:
		return codes.Canceled
	case http.StatusInternalServerError:
		return codes.Internal
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusGatewayTimeout:
		return codes.DeadlineExceeded
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusConflict:
		return codes.AlreadyExists
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusTooManyRequests:
		return codes.ResourceExhausted
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusUnprocessableEntity:
		return codes.InvalidArgument
	case http.StatusServiceUnavailable:
		return codes.Unavailable
	default:
		grpcLogger.WithField("httpStatusCode", httpStatusCode).Warn("received unknown http status code, defaulting to codes.ResourceExhausted / 429")
		return codes.ResourceExhausted
	}
}
