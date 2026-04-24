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

package apis

const (
	// Packed scheduling strategy will prioritise allocating GameServers
	// on Nodes with the most Allocated, and then Ready GameServers
	// to bin pack as many Allocated GameServers on a single node.
	// This is most useful for dynamic Kubernetes clusters - such as on Cloud Providers.
	// In future versions, this will also impact Fleet scale down, and Pod Scheduling.
	Packed SchedulingStrategy = "Packed"

	// Distributed scheduling strategy will prioritise allocating GameServers
	// on Nodes with the least Allocated, and then Ready GameServers
	// to distribute Allocated GameServers across many nodes.
	// This is most useful for statically sized Kubernetes clusters - such as on physical hardware.
	// In future versions, this will also impact Fleet scale down, and Pod Scheduling.
	Distributed SchedulingStrategy = "Distributed"
)

// SchedulingStrategy is the strategy that a Fleet & GameServers will use
// when scheduling GameServers' Pods across a cluster.
type SchedulingStrategy string
