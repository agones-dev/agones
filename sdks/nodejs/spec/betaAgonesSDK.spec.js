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

const messages = require("../lib/beta/beta_pb");
const Beta = require("../src/beta");

describe("Beta", () => {
	let beta;

	beforeEach(() => {
		const address = "localhost:9357";
		const credentials = grpc.credentials.createInsecure();
		beta = new Beta(address, credentials);
	});

	describe("getCounterCount", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "getCounter").and.callFake((_request, callback) => {
				const counter = new messages.Counter();
				counter.setCount(10);
				callback(undefined, counter);
			});

			const count = await beta.getCounterCount("key");
			expect(beta.client.getCounter).toHaveBeenCalled();
			expect(count).toEqual(10);
			const request = beta.client.getCounter.calls.argsFor(0)[0];
			expect(request.getName()).toEqual("key");
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "getCounter").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.getCounterCount("key");
				fail();
			} catch (error) {
				expect(beta.client.getCounter).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("incrementCounter", () => {
		it("calls the server and handles the response while under capacity", async () => {
			spyOn(beta.client, "updateCounter").and.callFake((_request, callback) => {
				const counter = new messages.Counter();
				callback(undefined, counter);
			});

			const response = await beta.incrementCounter("key", 5);
			expect(beta.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(undefined);
			const request = beta.client.updateCounter.calls.argsFor(0)[0];
			expect(request.getCounterupdaterequest().getName()).toEqual("key");
			expect(request.getCounterupdaterequest().getCountdiff()).toEqual(5);
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "updateCounter").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.incrementCounter("key", 5);
				fail();
			} catch (error) {
				expect(beta.client.updateCounter).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("decrementCounter", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "updateCounter").and.callFake((_request, callback) => {
				const counter = new messages.Counter();
				callback(undefined, counter);
			});

			const response = await beta.decrementCounter("key", 5);
			expect(beta.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(undefined);
			const request = beta.client.updateCounter.calls.argsFor(0)[0];
			expect(request.getCounterupdaterequest().getName()).toEqual("key");
			expect(request.getCounterupdaterequest().getCountdiff()).toEqual(-5);
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "updateCounter").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.decrementCounter("key", 5);
				fail();
			} catch (error) {
				expect(beta.client.updateCounter).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("setCounterCount", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "updateCounter").and.callFake((_request, callback) => {
				const counter = new messages.Counter();
				callback(undefined, counter);
			});

			const response = await beta.setCounterCount("key", 5);
			expect(beta.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(undefined);
			const request = beta.client.updateCounter.calls.argsFor(0)[0];
			expect(request.getCounterupdaterequest().getName()).toEqual("key");
			expect(request.getCounterupdaterequest().getCount().getValue()).toEqual(
				5,
			);
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "updateCounter").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.setCounterCount("key", 5);
				fail();
			} catch (error) {
				expect(beta.client.updateCounter).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("getCounterCapacity", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "getCounter").and.callFake((_request, callback) => {
				const counter = new messages.Counter();
				counter.setCapacity(10);
				callback(undefined, counter);
			});

			const capacity = await beta.getCounterCapacity("key");
			expect(beta.client.getCounter).toHaveBeenCalled();
			expect(capacity).toEqual(10);
			const request = beta.client.getCounter.calls.argsFor(0)[0];
			expect(request.getName()).toEqual("key");
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "getCounter").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.getCounterCapacity("key");
				fail();
			} catch (error) {
				expect(beta.client.getCounter).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("setCounterCapacity", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "updateCounter").and.callFake((_request, callback) => {
				const counter = new messages.Counter();
				callback(undefined, counter);
			});

			const response = await beta.setCounterCapacity("key", 5);
			expect(beta.client.updateCounter).toHaveBeenCalled();
			expect(response).toEqual(undefined);
			const request = beta.client.updateCounter.calls.argsFor(0)[0];
			expect(request.getCounterupdaterequest().getName()).toEqual("key");
			expect(
				request.getCounterupdaterequest().getCapacity().getValue(),
			).toEqual(5);
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "updateCounter").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.setCounterCapacity("key", 5);
				fail();
			} catch (error) {
				expect(beta.client.updateCounter).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("getListCapacity", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "getList").and.callFake((_request, callback) => {
				const list = new messages.List();
				list.setCapacity(10);
				callback(undefined, list);
			});

			const capacity = await beta.getListCapacity("key");
			expect(beta.client.getList).toHaveBeenCalled();
			expect(capacity).toEqual(10);
			const request = beta.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual("key");
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "getList").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.getListCapacity("key");
				fail();
			} catch (error) {
				expect(beta.client.getList).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("setListCapacity", () => {
		it("calls the server and handles the response while in range", async () => {
			spyOn(beta.client, "updateList").and.callFake((_request, callback) => {
				const list = new messages.List();
				callback(undefined, list);
			});

			const response = await beta.setListCapacity("key", 5);
			expect(beta.client.updateList).toHaveBeenCalled();
			expect(response).toEqual(undefined);
			const request = beta.client.updateList.calls.argsFor(0)[0];
			expect(request.getList().getName()).toEqual("key");
			expect(request.getList().getCapacity()).toEqual(5);
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "updateList").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.setListCapacity("key", 5);
				fail();
			} catch (error) {
				expect(beta.client.updateList).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("listContains", () => {
		it("calls the server and handles the response when the value is contained", async () => {
			spyOn(beta.client, "getList").and.callFake((_request, callback) => {
				const list = new messages.List();
				list.setValuesList(["firstValue", "secondValue"]);
				callback(undefined, list);
			});

			const contains = await beta.listContains("key", "secondValue");
			expect(beta.client.getList).toHaveBeenCalled();
			expect(contains).toEqual(true);
			const request = beta.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual("key");
		});

		it("calls the server and handles the response when the value is not contained", async () => {
			spyOn(beta.client, "getList").and.callFake((_request, callback) => {
				const list = new messages.List();
				list.setValuesList(["firstValue", "secondValue"]);
				callback(undefined, list);
			});

			const contains = await beta.listContains("key", "thirdValue");
			expect(beta.client.getList).toHaveBeenCalled();
			expect(contains).toEqual(false);
			const request = beta.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual("key");
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "getList").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.listContains("key", "value");
				fail();
			} catch (error) {
				expect(beta.client.getList).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("getListLength", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "getList").and.callFake((_request, callback) => {
				const list = new messages.List();
				list.setValuesList(["firstValue", "secondValue"]);
				callback(undefined, list);
			});

			const length = await beta.getListLength("key");
			expect(beta.client.getList).toHaveBeenCalled();
			expect(length).toEqual(2);
			const request = beta.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual("key");
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "getList").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.getListLength("key");
				fail();
			} catch (error) {
				expect(beta.client.getList).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("getListValues", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "getList").and.callFake((_request, callback) => {
				const list = new messages.List();
				list.setValuesList(["firstValue", "secondValue"]);
				callback(undefined, list);
			});

			const values = await beta.getListValues("key");
			expect(beta.client.getList).toHaveBeenCalled();
			expect(values).toEqual(["firstValue", "secondValue"]);
			const request = beta.client.getList.calls.argsFor(0)[0];
			expect(request.getName()).toEqual("key");
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "getList").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.getListValues("key");
				fail();
			} catch (error) {
				expect(beta.client.getList).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("appendListValue", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "addListValue").and.callFake((_request, callback) => {
				const list = new messages.List();
				callback(undefined, list);
			});

			const response = await beta.appendListValue("key", "value");
			expect(beta.client.addListValue).toHaveBeenCalled();
			expect(response).toEqual(undefined);
			const request = beta.client.addListValue.calls.argsFor(0)[0];
			expect(request.getName()).toEqual("key");
			expect(request.getValue()).toEqual("value");
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "addListValue").and.callFake((_request, callback) => {
				callback("error", undefined);
			});
			try {
				await beta.appendListValue("key", "value");
				fail();
			} catch (error) {
				expect(beta.client.addListValue).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});

	describe("deleteListValue", () => {
		it("calls the server and handles the response", async () => {
			spyOn(beta.client, "removeListValue").and.callFake(
				(_request, callback) => {
					const list = new messages.List();
					callback(undefined, list);
				},
			);

			const response = await beta.deleteListValue("key", "value");
			expect(beta.client.removeListValue).toHaveBeenCalled();
			expect(response).toEqual(undefined);
			const request = beta.client.removeListValue.calls.argsFor(0)[0];
			expect(request.getName()).toEqual("key");
			expect(request.getValue()).toEqual("value");
		});

		it("calls the server and handles failure", async () => {
			spyOn(beta.client, "removeListValue").and.callFake(
				(_request, callback) => {
					callback("error", undefined);
				},
			);
			try {
				await beta.deleteListValue("key", "value");
				fail();
			} catch (error) {
				expect(beta.client.removeListValue).toHaveBeenCalled();
				expect(error).toEqual("error");
			}
		});
	});
});
