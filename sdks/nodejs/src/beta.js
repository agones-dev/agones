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
const fieldMask = require("google-protobuf/google/protobuf/field_mask_pb");
const jspbWrappers = require("google-protobuf/google/protobuf/wrappers_pb");

const messages = require("../lib/beta/beta_pb");
const servicesPackageDefinition = require("../lib/beta/beta_grpc_pb");

class Beta {
	constructor(address, credentials) {
		const services = grpc.loadPackageDefinition(servicesPackageDefinition);
		this.client = new services.agones.dev.sdk.beta.SDK(address, credentials);
	}

	async getCounterCount(key) {
		const request = new messages.GetCounterRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getCounter(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCount());
				}
			});
		});
	}

	async incrementCounter(key, amount) {
		const updateRequest = new messages.CounterUpdateRequest();
		updateRequest.setName(key);
		updateRequest.setCountdiff(amount);
		const request = new messages.UpdateCounterRequest();
		request.setCounterupdaterequest(updateRequest);

		return new Promise((resolve, reject) => {
			this.client.updateCounter(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async decrementCounter(key, amount) {
		const updateRequest = new messages.CounterUpdateRequest();
		updateRequest.setName(key);
		updateRequest.setCountdiff(-amount);
		const request = new messages.UpdateCounterRequest();
		request.setCounterupdaterequest(updateRequest);

		return new Promise((resolve, reject) => {
			this.client.updateCounter(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async setCounterCount(key, amount) {
		const count = new jspbWrappers.Int64Value();
		count.setValue(amount);
		const updateRequest = new messages.CounterUpdateRequest();
		updateRequest.setName(key);
		updateRequest.setCount(count);
		const request = new messages.UpdateCounterRequest();
		request.setCounterupdaterequest(updateRequest);

		return new Promise((resolve, reject) => {
			this.client.updateCounter(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async getCounterCapacity(key) {
		const request = new messages.GetCounterRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getCounter(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCapacity());
				}
			});
		});
	}

	async setCounterCapacity(key, amount) {
		const capacity = new jspbWrappers.Int64Value();
		capacity.setValue(amount);
		const updateRequest = new messages.CounterUpdateRequest();
		updateRequest.setName(key);
		updateRequest.setCapacity(capacity);
		const request = new messages.UpdateCounterRequest();
		request.setCounterupdaterequest(updateRequest);

		return new Promise((resolve, reject) => {
			this.client.updateCounter(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async getListCapacity(key) {
		const request = new messages.GetListRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getList(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getCapacity());
				}
			});
		});
	}

	async setListCapacity(key, amount) {
		const list = new messages.List();
		list.setName(key);
		list.setCapacity(amount);
		const updateMask = new fieldMask.FieldMask({ Name: "capacity" });
		const request = new messages.UpdateListRequest();
		request.setList(list);
		request.setUpdateMask(updateMask);

		return new Promise((resolve, reject) => {
			this.client.updateList(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async listContains(key, value) {
		const request = new messages.GetListRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getList(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getValuesList().indexOf(value) >= 0);
				}
			});
		});
	}

	async getListLength(key) {
		const request = new messages.GetListRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getList(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getValuesList().length);
				}
			});
		});
	}

	async getListValues(key) {
		const request = new messages.GetListRequest();
		request.setName(key);

		return new Promise((resolve, reject) => {
			this.client.getList(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.getValuesList());
				}
			});
		});
	}

	async appendListValue(key, value) {
		const request = new messages.AddListValueRequest();
		request.setName(key);
		request.setValue(value);

		return new Promise((resolve, reject) => {
			this.client.addListValue(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	async deleteListValue(key, value) {
		const request = new messages.RemoveListValueRequest();
		request.setName(key);
		request.setValue(value);

		return new Promise((resolve, reject) => {
			this.client.removeListValue(request, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}
}

module.exports = Beta;
