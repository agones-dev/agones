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

const grpc = require("@grpc/grpc-js");

const servicesPackageDefinition = require("../lib/alpha/alpha_grpc_pb");

class Alpha {
	constructor(address, credentials) {
		const services = grpc.loadPackageDefinition(servicesPackageDefinition);
		this.client = new services.agones.dev.sdk.alpha.SDK(address, credentials);
	}
}

module.exports = Alpha;
