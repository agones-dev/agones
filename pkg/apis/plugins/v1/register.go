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

package v1

import "k8s.io/apimachinery/pkg/runtime/schema"

// SchemeGroupVersion is the group version of the plugin types. They are only embedded in other
// resources, so nothing is registered with a scheme, but the generated apply configurations refer to it.
var SchemeGroupVersion = schema.GroupVersion{Group: "plugins.agones.dev", Version: "v1"}
