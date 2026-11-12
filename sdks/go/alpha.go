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

package sdk

import (
	"google.golang.org/grpc"

	"agones.dev/agones/pkg/sdk/alpha"
	"agones.dev/agones/pkg/util/errors"
)

// Alpha is the struct for Alpha SDK functionality.
type Alpha struct {
	client alpha.SDKClient
	errs   *errors.Errors
}

// newAlpha creates a new Alpha SDK with the passed in connection.
func newAlpha(conn *grpc.ClientConn) *Alpha {
	a := &Alpha{client: alpha.NewSDKClient(conn)}
	a.errs = errors.FromStruct(a)
	return a
}
