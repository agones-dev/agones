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

const Alpha = require("./alpha");
const Beta = require("./beta");

const messages = require("../lib/sdk_pb");
const servicesPackageDefinition = require("../lib/sdk_grpc_pb");

class AgonesSDK {
	constructor() {
		const services = grpc.loadPackageDefinition(servicesPackageDefinition);
		const address = `localhost:${this.port}`;
		const credentials = grpc.credentials.createInsecure();
		this.client = new services.agones.dev.sdk.SDK(address, credentials);
		this.healthStream = undefined;
		this.streams = [];
		this.alpha = new Alpha(address, credentials);
		this.beta = new Beta(address, credentials);
	}

	get port() {
		return process.env.AGONES_SDK_GRPC_PORT || "9357";
	}

	async connect() {
		return new Promise((resolve, reject) => {
			this.client.waitForReady(Date.now() + 30000, (error) => {
				if (error) {
					reject(error);
				} else {
					resolve();
				}
			});
		});
	}

	close() {
		if (this.healthStream !== undefined) {
			this.healthStream.end();
		}
		for (const stream of this.streams) {
			stream.destroy();
		}
		this.client.close();
	}

	async ready() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this.client.ready(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	async allocate() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this.client.allocate(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	async shutdown() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this.client.shutdown(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	health(errorCallback) {
		if (this.healthStream === undefined) {
			this.healthStream = this.client.health(() => {
				// Ignore error as this can't be caught
			});
			this.healthStream.on("error", () => {
				// ignore error, prevent from being uncaught
			});
		}
		const request = new messages.Empty();
		this.healthStream.write(request, null, (error) => {
			if (error) {
				if (errorCallback) {
					errorCallback(error);
					return;
				}
				throw error;
			}
		});
	}

	async getGameServer() {
		const request = new messages.Empty();
		return new Promise((resolve, reject) => {
			this.client.getGameServer(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	watchGameServer(callback, errorCallback) {
		const request = new messages.Empty();
		const stream = this.client.watchGameServer(request);
		stream.on("data", (data) => {
			callback(data.toObject());
		});
		if (errorCallback) {
			stream.on("error", (error) => {
				errorCallback(error);
			});
		}
		this.streams.push(stream);
	}

	async setLabel(key, value) {
		const request = new messages.KeyValue();
		request.setKey(key);
		request.setValue(value);
		return new Promise((resolve, reject) => {
			this.client.setLabel(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	async setAnnotation(key, value) {
		const request = new messages.KeyValue();
		request.setKey(key);
		request.setValue(value);
		return new Promise((resolve, reject) => {
			this.client.setAnnotation(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}

	async reserve(duration) {
		const request = new messages.Duration();
		request.setSeconds(duration);
		return new Promise((resolve, reject) => {
			this.client.reserve(request, (error, response) => {
				if (error) {
					reject(error);
				} else {
					resolve(response.toObject());
				}
			});
		});
	}
}

module.exports = AgonesSDK;
