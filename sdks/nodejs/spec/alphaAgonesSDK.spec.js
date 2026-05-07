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

const messages = require("../lib/alpha/alpha_pb");
const Alpha = require("../src/alpha");

describe("Alpha", () => {
	let alpha;

	beforeEach(() => {
		const address = "localhost:9357";
		const credentials = grpc.credentials.createInsecure();
		alpha = new Alpha(address, credentials);
	});

	describe("playerConnect", () => {
		it("calls the server and handles success when the player is not connected", async () => {
			spyOn(alpha.client, "playerConnect").and.callFake(
				(_request, callback) => {
					const result = new messages.Bool();
					result.setBool(true);
					callback(undefined, result);
				},
			);

			const result = await alpha.playerConnect("playerID");
			expect(alpha.client.playerConnect).toHaveBeenCalled();
			expect(result).toEqual(true);
		});

		it("calls the server and handles success when the player is already connected", async () => {
			spyOn(alpha.client, "playerConnect").and.callFake(
				(_request, callback) => {
					const result = new messages.Bool();
					result.setBool(false);
					callback(undefined, result);
				},
			);

			const result = await alpha.playerConnect("playerID");
			expect(alpha.client.playerConnect).toHaveBeenCalled();
			expect(result).toEqual(false);
		});

		it("calls the server and handles failure", async () => {
			spyOn(alpha.client, "playerConnect").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await alpha.playerConnect("playerID");
				fail();
			} catch (error) {
				expect(alpha.client.playerConnect).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("playerDisconnect", () => {
		it("calls the server and handles success when the player is connected", async () => {
			spyOn(alpha.client, "playerDisconnect").and.callFake(
				(_request, callback) => {
					const result = new messages.Bool();
					result.setBool(true);
					callback(undefined, result);
				},
			);

			const result = await alpha.playerDisconnect("playerID");
			expect(alpha.client.playerDisconnect).toHaveBeenCalled();
			expect(result).toEqual(true);
		});

		it("calls the server and handles success when the player is not connected", async () => {
			spyOn(alpha.client, "playerDisconnect").and.callFake(
				(_request, callback) => {
					const result = new messages.Bool();
					result.setBool(false);
					callback(undefined, result);
				},
			);

			const result = await alpha.playerDisconnect("playerID");
			expect(alpha.client.playerDisconnect).toHaveBeenCalled();
			expect(result).toEqual(false);
		});

		it("calls the server and handles failure", async () => {
			spyOn(alpha.client, "playerDisconnect").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await alpha.playerDisconnect("playerID");
				fail();
			} catch (error) {
				expect(alpha.client.playerDisconnect).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("setPlayerCapacity", () => {
		it("passes arguments to the server and handles success", async () => {
			spyOn(alpha.client, "setPlayerCapacity").and.callFake(
				(_request, callback) => {
					const result = new messages.Empty();
					callback(undefined, result);
				},
			);

			const result = await alpha.setPlayerCapacity(64);
			expect(result).toEqual({});
			expect(alpha.client.setPlayerCapacity).toHaveBeenCalled();
			const request = alpha.client.setPlayerCapacity.calls.argsFor(0)[0];
			expect(request.getCount()).toEqual(64);
		});

		it("calls the server and handles failure", async () => {
			spyOn(alpha.client, "setPlayerCapacity").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await alpha.setPlayerCapacity(64);
				fail();
			} catch (error) {
				expect(alpha.client.setPlayerCapacity).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("getPlayerCapacity", () => {
		it("calls the server and handles the response", async () => {
			spyOn(alpha.client, "getPlayerCapacity").and.callFake(
				(_request, callback) => {
					const capacity = new messages.Count();
					capacity.setCount(64);
					callback(undefined, capacity);
				},
			);

			const capacity = await alpha.getPlayerCapacity();
			expect(alpha.client.getPlayerCapacity).toHaveBeenCalled();
			expect(capacity).toEqual(64);
		});

		it("calls the server and handles failure", async () => {
			spyOn(alpha.client, "getPlayerCapacity").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await alpha.getPlayerCapacity();
				fail();
			} catch (error) {
				expect(alpha.client.getPlayerCapacity).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("getPlayerCount", () => {
		it("calls the server and handles the response", async () => {
			spyOn(alpha.client, "getPlayerCount").and.callFake(
				(_request, callback) => {
					const capacity = new messages.Count();
					capacity.setCount(16);
					callback(undefined, capacity);
				},
			);

			const capacity = await alpha.getPlayerCount();
			expect(alpha.client.getPlayerCount).toHaveBeenCalled();
			expect(capacity).toEqual(16);
		});

		it("calls the server and handles failure", async () => {
			spyOn(alpha.client, "getPlayerCount").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await alpha.getPlayerCount();
				fail();
			} catch (error) {
				expect(alpha.client.getPlayerCount).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("isPlayerConnected", () => {
		it("calls the server and handles success when the player is connected", async () => {
			spyOn(alpha.client, "isPlayerConnected").and.callFake(
				(_request, callback) => {
					const result = new messages.Bool();
					result.setBool(true);
					callback(undefined, result);
				},
			);

			const result = await alpha.isPlayerConnected("playerID");
			expect(alpha.client.isPlayerConnected).toHaveBeenCalled();
			expect(result).toEqual(true);
		});

		it("calls the server and handles success when the player is not connected", async () => {
			spyOn(alpha.client, "isPlayerConnected").and.callFake(
				(_request, callback) => {
					const result = new messages.Bool();
					result.setBool(false);
					callback(undefined, result);
				},
			);

			const result = await alpha.isPlayerConnected("playerID");
			expect(alpha.client.isPlayerConnected).toHaveBeenCalled();
			expect(result).toEqual(false);
		});

		it("calls the server and handles failure", async () => {
			spyOn(alpha.client, "isPlayerConnected").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await alpha.isPlayerConnected("playerID");
				fail();
			} catch (error) {
				expect(alpha.client.isPlayerConnected).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("getConnectedPlayers", () => {
		it("calls the server and handles the response", async () => {
			spyOn(alpha.client, "getConnectedPlayers").and.callFake(
				(_request, callback) => {
					const connectedPlayers = new messages.PlayerIDList();
					connectedPlayers.setListList(["firstPlayerID", "secondPlayerID"]);
					callback(undefined, connectedPlayers);
				},
			);

			const connectedPlayers = await alpha.getConnectedPlayers();
			expect(alpha.client.getConnectedPlayers).toHaveBeenCalled();
			expect(connectedPlayers).toEqual(["firstPlayerID", "secondPlayerID"]);
		});

		it("calls the server and handles failure", async () => {
			spyOn(alpha.client, "getConnectedPlayers").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await alpha.getConnectedPlayers();
				fail();
			} catch (error) {
				expect(alpha.client.getConnectedPlayers).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});
});
