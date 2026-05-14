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

const DEFAULT_TIMEOUT = 60;
const MAX_TIMEOUT = 2147483;

const connect = async (timeout, enableAlpha, enableBeta) => {
	let agonesSDK = new AgonesSDK();

	let lifetimeInterval;
	let healthInterval;
	
	try {
		console.log(`Connecting to the SDK server...`);
		await agonesSDK.connect();
		console.log('...connected to SDK server');

		let connectTime = Date.now();
		lifetimeInterval = setInterval(() => {
			let lifetimeSeconds = Math.floor((Date.now() - connectTime) / 1000);
			console.log(`Running for ${lifetimeSeconds} seconds`);
		}, 10000);
		healthInterval = setInterval(() => {
			agonesSDK.health();
			console.log('Health ping sent');
		}, 20000);
		agonesSDK.watchGameServer((result) => {
			let output = `GameServer Update:
	name: ${result.objectMeta.name}
	state: ${result.status.state}
	labels: ${result.objectMeta.labelsMap.join(' & ')}
	annotations: ${result.objectMeta.annotationsMap.join(' & ')}`;
			if (enableAlpha) {
				output += `
	players: ${result.status.players.count}/${result.status.players.capacity} [${result.status.players.idsList}]`;
			}
			console.log(output);
		}, (error) => {
			console.error('Watch ERROR', error);
			clearInterval(healthInterval);
			clearInterval(lifetimeInterval);
			process.exit(0);
		});

		await setTimeout(5000);
		console.log('Setting a label');
		await agonesSDK.setLabel('test-label', 'test-value');

		await setTimeout(5000);
		console.log('Setting an annotation');
		await agonesSDK.setAnnotation('test-annotation', 'test value');

		await setTimeout(5000);
		console.log('Marking server as ready...');
		await agonesSDK.ready();
		
		await setTimeout(5000);
		console.log('Allocating');
		await agonesSDK.allocate();

		await setTimeout(5000);
		console.log('Reserving for 10 seconds');
		await agonesSDK.reserve(10);
		await setTimeout(15000);

		if (enableAlpha) {
			console.log('Running alpha suite');
			await runAlphaSuite(agonesSDK);
		}

		if (enableBeta) {
			console.log('Running beta suite');
			await runBetaSuite(agonesSDK);
		}

		if (timeout === 0) {
			do {
				await setTimeout(MAX_TIMEOUT);
			} while (true);
		}

		console.log(`Shutting down after timeout of ${timeout} seconds...`);
		await setTimeout(timeout * 1000);
		console.log('Shutting down...');
		agonesSDK.shutdown();

		await setTimeout(5000);
		console.log('Closing connection to SDK server');
		clearInterval(healthInterval);
		agonesSDK.close();

		await setTimeout(5000);
		console.log('Exiting');
		clearInterval(lifetimeInterval);

		// Some parts of the SDK are still using the event loop so must exit manually until fixed
		process.exit(0);
	} catch (error) {
		console.error(error);
		clearInterval(healthInterval);
		clearInterval(lifetimeInterval);
		process.exit(0);
	}
};

const runAlphaSuite = async (agonesSDK) => {
	await setTimeout(5000);
	console.log('Setting capacity');
	await agonesSDK.alpha.setPlayerCapacity(64);

	await setTimeout(5000);
	console.log('Getting capacity');
	let result = await agonesSDK.alpha.getPlayerCapacity();
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Connecting a player');
	result = await agonesSDK.alpha.playerConnect('firstPlayerID');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Connecting a duplicate player');
	result = await agonesSDK.alpha.playerConnect('firstPlayerID');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Connecting another player');
	await agonesSDK.alpha.playerConnect('secondPlayerID');

	await setTimeout(5000);
	console.log('Getting player count');
	result = await agonesSDK.alpha.getPlayerCount();
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Finding if firstPlayerID connected');
	result = await agonesSDK.alpha.isPlayerConnected('firstPlayerID');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Getting connected players');
	result = await agonesSDK.alpha.getConnectedPlayers();
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Disconnecting a player');
	result = await agonesSDK.alpha.playerDisconnect('firstPlayerID');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Disconnecting the same player');
	result = await agonesSDK.alpha.playerDisconnect('firstPlayerID');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Setting counter capacity');
	result = await agonesSDK.alpha.setCounterCapacity('testCounter', 10);
	console.log(`result: ${result}`);
};

const runBetaSuite = async (agonesSDK) => {
	let result;

	await setTimeout(5000);
	console.log('Getting counter count');
	result = await agonesSDK.beta.getCounterCount('rooms');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Incrementing counter');
	await agonesSDK.beta.incrementCounter('rooms', 1);

	await setTimeout(5000);
	console.log('Decrementing counter');
	await agonesSDK.beta.decrementCounter('rooms', 1);

	await setTimeout(5000);
	console.log('Setting counter count');
	await agonesSDK.beta.setCounterCount('rooms', 2);

	await setTimeout(5000);
	console.log('Getting counter capacity');
	result = await agonesSDK.beta.getCounterCapacity('rooms');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Setting counter capacity');
	await agonesSDK.beta.setCounterCapacity('rooms', 200);

	await setTimeout(5000);
	console.log('Getting list capacity');
	result = await agonesSDK.beta.getListCapacity('players');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Setting list capacity');
	await agonesSDK.beta.setListCapacity('players', 10);

	await setTimeout(5000);
	console.log('Getting list contains');
	result = await agonesSDK.beta.listContains('players', 'test0');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Getting list length');
	result = await agonesSDK.beta.getListLength('players');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Getting list values');
	result = await agonesSDK.beta.getListValues('players');
	console.log(`result: ${result}`);

	await setTimeout(5000);
	console.log('Appending list value');
	await agonesSDK.beta.appendListValue('players', 'test3');

	await setTimeout(5000);
	console.log('Deleting list value');
	await agonesSDK.beta.deleteListValue('players', 'test3');
};

let args = process.argv.slice(2);
let timeout = DEFAULT_TIMEOUT;
let enableAlpha = false;
let enableBeta = false;

for (let arg of args) {
	let [argName, argValue] = arg.split('=');
	if (argName === '--help') {
		console.log(`Example to call each SDK feature in turn. Once complete will call shutdown and close after a default timeout of ${DEFAULT_TIMEOUT} seconds.
		
Options:
	--timeout=...\t\tshutdown timeout in seconds. Use 0 to never shut down
	--alpha\t\t\tenable alpha features
	--beta\t\t\tenable beta features`);
		return;
	}
	if (argName === '--timeout') {
		timeout = Number(argValue);
		if (Number.isNaN(timeout)) {
			timeout = DEFAULT_TIMEOUT;
		} else if (timeout < 0 || timeout > MAX_TIMEOUT) {
			timeout = 0;
		}
		if (timeout === 0) {
			console.log(`Using shutdown timeout of never`);
		} else {
			console.log(`Using shutdown timeout of ${timeout} seconds`);
		}
	}

	if (argName === '--alpha') {
		console.log('Enabling alpha features!');
		enableAlpha = true;
	}

	if (argName === '--beta') {
		console.log('Enabling beta features!');
		enableBeta = true;
	}
}

connect(timeout, enableAlpha, enableBeta);
