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

const AgonesSDK = require('@google-cloud/agones-sdk');
const {setTimeout} = require('timers/promises');

const agonesSDK = new AgonesSDK();

const connect = async () => {
	let UID = '';
	try {
		console.log("attempting to connect");
		await agonesSDK.connect();
		console.log("connected!");
		let once = true;
		agonesSDK.watchGameServer((result) => {
			console.log('watch', result);
			UID = result.objectMeta.uid.toString();
			if (once) {
				console.log('Setting annotation ', UID);
				agonesSDK.setAnnotation('annotation', UID);
				once = false;
			}
		}, (error) => {
			console.error('Watch ERROR', error);
		});
		await agonesSDK.ready();

		let result = await agonesSDK.getGameServer();
		await agonesSDK.setLabel('label', result.objectMeta.creationTimestamp.toString());
		console.log('gameServer', result);
		console.log('creation Timestamp', result.objectMeta.creationTimestamp);
		result = await agonesSDK.health();
		console.log('health', result);

		await agonesSDK.reserve(5);

		await setTimeout(1000);		
		console.log('send allocate request');
		agonesSDK.allocate();

		await testCounts(agonesSDK);
		await testLists(agonesSDK);

		await setTimeout(1000);
		console.log('send shutdown request');
		agonesSDK.shutdown();
		
		await setTimeout(2000);
		console.log('closing agones SDK');
		// Closing Agones SDK and all event emitters
		agonesSDK.close()
		
		await setTimeout(2000);
		process.exit(0);
	} catch (error) {
		console.error(error);
	}
};

const testCounts = async(sdk) => {
	// LocalSDKServer starting "rooms": {Count: 1, Capacity: 10}
	const counter = "rooms";

	try {
		let count = await sdk.beta.getCounterCount(counter);
		if (count !== 1) {
			throw new Error(`Counter count should be 1, but is ${count}`);
		}
	} catch (error) {
		throw new Error(`Error getting Counter count: ${error}`);
	}

	try {
		await sdk.beta.incrementCounter(counter, 9);
	} catch (error) {
		throw new Error(`Error incrementing Counter: ${error}`);
	}

	try {
		await sdk.beta.decrementCounter(counter, 10);
	} catch (error) {
		throw new Error(`Error decrementing Counter: ${error}`);
	}

	try {
		await sdk.beta.setCounterCount(counter, 10);
	} catch (error) {
		throw new Error(`Error setting Counter count: ${error}`);
	}

	try {
		let capacity = await sdk.beta.getCounterCapacity(counter);
		if (capacity !== 10) {
			throw new Error(`Counter capacity should be 10, but is ${capacity}`);
		}
	} catch (error) {
		throw new Error(`Error getting Counter capacity: ${error}`);
	}

	try {
		await sdk.beta.setCounterCapacity(counter, 1);
	} catch (error) {
		throw new Error(`Error setting Counter capacity: ${error}`);
	}
}

const testLists = async(sdk) => {
	// LocalSDKServer starting "players": {Values: []string{"test0", "test1", "test2"}, Capacity: 100}}
	const list = "players"
	const listValues = ["test0", "test1", "test2"]

	let contains = await sdk.beta.listContains(list, "test1");
	if (!contains) {
		throw new Error("List should contain value \"test1\"");
	}

	try {
		let length = await sdk.beta.getListLength(list);
		if (length !== 3) {
			throw new Error(`List length should be 3, but is ${length}`);
		}
	} catch (error) {
		throw new Error(`Error getting List length: ${error}`);
	}

	try {
		let values = await sdk.beta.getListValues(list);
		if (JSON.stringify(values) !== JSON.stringify(listValues)) {
			throw new Error(`List values should be ${listValues}, but is ${values}`);
		}
	} catch (error) {
		throw new Error(`Error getting List values: ${error}`);
	}

	await sdk.beta.appendListValue(list, "test3");

	await sdk.beta.deleteListValue(list, "test2");

	try {
		let capacity = await sdk.beta.getListCapacity(list);
		if (capacity !== 100) {
			throw new Error(`List capacity should be 100, but is ${capacity}`);
		}
	} catch (error) {
		throw new Error(`Error getting List capacity: ${error}`);
	}

	await sdk.beta.setListCapacity(list, 2);
}

connect();
