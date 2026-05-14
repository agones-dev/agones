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

const stream = require("node:stream");

const grpc = require("@grpc/grpc-js");

const messages = require("../lib/sdk_pb");

const AgonesSDK = require("../src/agonesSDK");
const Alpha = require("../src/alpha");

describe("AgonesSDK", () => {
	let agonesSDK;

	beforeEach(() => {
		agonesSDK = new AgonesSDK();
	});

	describe("port", () => {
		it("returns the default port if $AGONES_SDK_GRPC_PORT is not defined", async () => {
			const port = agonesSDK.port;
			expect(port).toEqual("9357");
		});

		it("returns a valid port set in $AGONES_SDK_GRPC_PORT", async () => {
			process.env.AGONES_SDK_GRPC_PORT = "6789";
			const port = agonesSDK.port;
			expect(port).toEqual("6789");
			delete process.env.AGONES_SDK_GRPC_PORT;
		});

		it("returns an invalid port set in $AGONES_SDK_GRPC_PORT", async () => {
			process.env.AGONES_SDK_GRPC_PORT = "foo";
			const port = agonesSDK.port;
			expect(port).toEqual("foo");
			delete process.env.AGONES_SDK_GRPC_PORT;
		});
	});

	describe("connect", () => {
		it("calls the server and handles success", async () => {
			spyOn(agonesSDK.client, "waitForReady").and.callFake(
				(_deadline, callback) => {
					const result = new messages.Empty();
					callback(undefined, result);
				},
			);
			const result = await agonesSDK.connect();
			expect(agonesSDK.client.waitForReady).toHaveBeenCalled();
			expect(result).toEqual(undefined);
		});

		it("calls the server and handles failure", async () => {
			spyOn(agonesSDK.client, "waitForReady").and.callFake(
				(_deadline, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await agonesSDK.connect();
				fail();
			} catch (error) {
				expect(agonesSDK.client.waitForReady).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("ready", () => {
		it("calls the server and handles success", async () => {
			spyOn(agonesSDK.client, "ready").and.callFake((_request, callback) => {
				const result = new messages.Empty();
				callback(undefined, result);
			});
			const result = await agonesSDK.ready();
			expect(agonesSDK.client.ready).toHaveBeenCalled();
			expect(result).toEqual({});
		});

		it("calls the server and handles failure", async () => {
			spyOn(agonesSDK.client, "ready").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await agonesSDK.ready();
				fail();
			} catch (error) {
				expect(agonesSDK.client.ready).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("allocate", () => {
		it("calls the server and handles success", async () => {
			spyOn(agonesSDK.client, "allocate").and.callFake((_request, callback) => {
				const result = new messages.Empty();
				callback(undefined, result);
			});
			const result = await agonesSDK.allocate();
			expect(agonesSDK.client.allocate).toHaveBeenCalled();
			expect(result).toEqual({});
		});

		it("calls the server and handles failure", async () => {
			spyOn(agonesSDK.client, "allocate").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await agonesSDK.allocate();
				fail();
			} catch (error) {
				expect(agonesSDK.client.allocate).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("shutdown", () => {
		it("calls the server and handles success", async () => {
			spyOn(agonesSDK.client, "shutdown").and.callFake((_request, callback) => {
				const result = new messages.Empty();
				callback(undefined, result);
			});

			const result = await agonesSDK.shutdown();
			expect(agonesSDK.client.shutdown).toHaveBeenCalled();
			expect(result).toEqual({});
		});

		it("calls the server and handles failure", async () => {
			spyOn(agonesSDK.client, "shutdown").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await agonesSDK.shutdown();
				fail();
			} catch (error) {
				expect(agonesSDK.client.shutdown).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("health", () => {
		it("calls the server and passes calls to stream", async () => {
			const stream = jasmine.createSpyObj("stream", ["write", "on"]);
			spyOn(agonesSDK.client, "health").and.callFake(() => {
				return stream;
			});

			agonesSDK.health();
			expect(agonesSDK.client.health).toHaveBeenCalled();
			expect(stream.write).toHaveBeenCalled();
		});

		it("uses the same stream for subsequent calls", async () => {
			const stream = jasmine.createSpyObj("stream", ["write", "on"]);
			spyOn(agonesSDK.client, "health").and.callFake(() => {
				return stream;
			});

			agonesSDK.health();
			agonesSDK.health();
			expect(agonesSDK.client.health.calls.count()).toEqual(1);
			expect(stream.write.calls.count()).toEqual(2);
		});

		it("calls the server and silently handles the internal error message", async () => {
			spyOn(agonesSDK.client, "health").and.callFake((callback) => {
				callback("error", undefined);
			});
			try {
				agonesSDK.health();
				fail();
			} catch (error) {
				expect(agonesSDK.client.health).toHaveBeenCalled();
				expect(error).not.toEqual("error");
			}
		});

		it("calls the server and handles stream write error if callback provided", async () => {
			const stream = jasmine.createSpyObj("stream", ["write", "on"]);
			stream.write.and.callFake((_chunk, _encoding, callback) => {
				callback("error");
			});
			spyOn(agonesSDK.client, "health").and.callFake(() => {
				return stream;
			});
			try {
				agonesSDK.health((error) => {
					expect(error).toEqual("error");
				});
			} catch (error) {
				fail(error);
			}
		});

		it("calls the server and re throws stream write error if no callback", async () => {
			const stream = jasmine.createSpyObj("stream", ["write", "on"]);
			stream.write.and.callFake((_chunk, _encoding, callback) => {
				callback("error");
			});
			spyOn(agonesSDK.client, "health").and.callFake(() => {
				return stream;
			});
			try {
				agonesSDK.health();
				fail();
			} catch (error) {
				expect(agonesSDK.client.health).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});

		it("does not call error callback if there was no stream error", async () => {
			const stream = jasmine.createSpyObj("stream", ["write", "on"]);
			stream.write.and.callFake((_chunk, _encoding, callback) => {
				callback();
			});
			spyOn(agonesSDK.client, "health").and.callFake(() => {
				return stream;
			});
			agonesSDK.health(() => {
				fail();
			});
		});

		it("calls the server and handles stream completing", async () => {
			const stream = jasmine.createSpyObj("stream", ["write", "on"]);
			spyOn(agonesSDK.client, "health").and.callFake((callback) => {
				const result = new messages.Empty();
				callback(undefined, result);
				return stream;
			});

			agonesSDK.health();
			expect(agonesSDK.client.health).toHaveBeenCalled();
		});
	});

	describe("getGameServer", () => {
		it("calls the server and handles the response", async () => {
			spyOn(agonesSDK.client, "getGameServer").and.callFake(
				(_request, callback) => {
					const status = new messages.GameServer.Status();
					status.setState("up");
					const gameServer = new messages.GameServer();
					gameServer.setStatus(status);
					callback(undefined, gameServer);
				},
			);

			const gameServer = await agonesSDK.getGameServer();
			expect(agonesSDK.client.getGameServer).toHaveBeenCalled();
			expect(gameServer).toBeDefined();
			expect(gameServer.status).toBeDefined();
			expect(gameServer.status.state).toEqual("up");
		});

		it("calls the server and handles failure", async () => {
			spyOn(agonesSDK.client, "getGameServer").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await agonesSDK.getGameServer();
				fail();
			} catch (error) {
				expect(agonesSDK.client.getGameServer).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("watchGameServer", () => {
		it("calls the server and passes events to the callback", async () => {
			const serverStream = stream.Readable({ read: () => undefined });
			spyOn(agonesSDK.client, "watchGameServer").and.callFake(() => {
				return serverStream;
			});

			const callback = jasmine.createSpy("callback");
			agonesSDK.watchGameServer(callback);
			expect(agonesSDK.client.watchGameServer).toHaveBeenCalled();

			const status = new messages.GameServer.Status();
			status.setState("up");
			const gameServer = new messages.GameServer();
			gameServer.setStatus(status);
			serverStream.emit("data", gameServer);

			expect(callback).toHaveBeenCalled();
			const callbackArgs = callback.calls.argsFor(0)[0];
			expect(callbackArgs.status).toBeDefined();
			expect(callbackArgs.status.state).toEqual("up");
		});
		it("calls the server and passes errors to the optional error callback", async () => {
			const serverStream = stream.Readable({ read: () => undefined });
			spyOn(agonesSDK.client, "watchGameServer").and.callFake(() => {
				return serverStream;
			});

			const callback = jasmine.createSpy("callback");
			const errorCallback = jasmine.createSpy("errorCallback");
			agonesSDK.watchGameServer(callback, errorCallback);

			const error = {
				code: grpc.status.CANCELLED,
			};
			serverStream.emit("error", error);
			expect(errorCallback).toHaveBeenCalled();
			const errorCallbackArgs = errorCallback.calls.argsFor(0)[0];
			expect(errorCallbackArgs).toEqual(error);
		});
	});

	describe("setLabel", () => {
		it("calls the server and handles success", async () => {
			spyOn(agonesSDK.client, "setLabel").and.callFake((_request, callback) => {
				const result = new messages.Empty();
				callback(undefined, result);
			});

			const result = await agonesSDK.setLabel("key", "value");
			expect(agonesSDK.client.setLabel).toHaveBeenCalled();
			expect(result).toEqual({});
		});

		it("passes arguments to the server", async () => {
			spyOn(agonesSDK.client, "setLabel").and.callFake((_request, callback) => {
				const result = new messages.Empty();
				callback(undefined, result);
			});

			await agonesSDK.setLabel("key", "value");
			expect(agonesSDK.client.setLabel).toHaveBeenCalled();
			const request = agonesSDK.client.setLabel.calls.argsFor(0)[0];
			expect(request.getKey()).toEqual("key");
			expect(request.getValue()).toEqual("value");
		});

		it("calls the server and handles failure", async () => {
			spyOn(agonesSDK.client, "setLabel").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await agonesSDK.setLabel("key", "value");
				fail();
			} catch (error) {
				expect(agonesSDK.client.setLabel).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("setAnnotation", () => {
		it("calls the server and handles success", async () => {
			spyOn(agonesSDK.client, "setAnnotation").and.callFake(
				(_request, callback) => {
					const result = new messages.Empty();
					callback(undefined, result);
				},
			);

			const result = await agonesSDK.setAnnotation("key", "value");
			expect(agonesSDK.client.setAnnotation).toHaveBeenCalled();
			expect(result).toEqual({});
		});

		it("passes arguments to the server", async () => {
			spyOn(agonesSDK.client, "setAnnotation").and.callFake(
				(_request, callback) => {
					const result = new messages.Empty();
					callback(undefined, result);
				},
			);

			await agonesSDK.setAnnotation("key", "value");
			expect(agonesSDK.client.setAnnotation).toHaveBeenCalled();
			const request = agonesSDK.client.setAnnotation.calls.argsFor(0)[0];
			expect(request.getKey()).toEqual("key");
			expect(request.getValue()).toEqual("value");
		});

		it("calls the server and handles failure", async () => {
			spyOn(agonesSDK.client, "setAnnotation").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await agonesSDK.setAnnotation("key", "value");
				fail();
			} catch (error) {
				expect(agonesSDK.client.setAnnotation).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("close", () => {
		it("closes the client connection when called", async () => {
			spyOn(agonesSDK.client, "close");
			await agonesSDK.close();
			expect(agonesSDK.client.close).toHaveBeenCalled();
		});
		it("ends the health stream if set", async () => {
			const stream = jasmine.createSpyObj("stream", ["end", "write", "on"]);
			spyOn(agonesSDK.client, "health").and.callFake(() => {
				return stream;
			});
			agonesSDK.health();
			spyOn(agonesSDK.client, "close").and.callFake(() => {});
			await agonesSDK.close();
			expect(stream.end).toHaveBeenCalled();
		});
		it("cancels any watchers", async () => {
			const serverStream = stream.Readable({ read: () => undefined });
			spyOn(serverStream, "destroy").and.callThrough();
			spyOn(agonesSDK.client, "watchGameServer").and.callFake(() => {
				return serverStream;
			});

			const callback = jasmine.createSpy("callback");
			agonesSDK.watchGameServer(callback);

			spyOn(agonesSDK.client, "close");
			await agonesSDK.close();
			expect(serverStream.destroy).toHaveBeenCalled();
		});
	});

	describe("reserve", () => {
		it("calls the server with duration parameter and handles success", async () => {
			spyOn(agonesSDK.client, "reserve").and.callFake((_request, callback) => {
				const result = new messages.Empty();
				callback(undefined, result);
			});

			const result = await agonesSDK.reserve(10);
			expect(agonesSDK.client.reserve).toHaveBeenCalled();
			expect(result).toEqual({});

			const request = agonesSDK.client.reserve.calls.argsFor(0)[0];
			expect(request.getSeconds()).toEqual(10);
		});

		it("calls the server and handles failure", async () => {
			spyOn(agonesSDK.client, "reserve").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await agonesSDK.reserve(10);
				fail();
			} catch (error) {
				expect(agonesSDK.client.reserve).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("alpha", () => {
		it("returns the alpha features class", () => {
			expect(agonesSDK.alpha).toBeInstanceOf(Alpha);
		});
	});
});
