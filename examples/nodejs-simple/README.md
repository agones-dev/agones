# Simple node.js Example

This is a very simple "server" that doesn't do much other than show how the SDK works in node.js.

It will:
- Set up the Agones SDK and connect to the SDK-server
- Add a watch to display status updates from the SDK-server
- Set labels, annotations
- Manage a fake lifecycle from `Ready` through `Allocated` and `Reserved`
- Every 10 seconds, write a log showing how long it has been running for
- Every 20 seconds, mark as healthy
- After the shutdown duration (default 60 seconds), shut the server down
- Parse options to get help or set the shutdown timeout duration

To learn how to deploy this example service to GKE, please see the tutorial [Build and Run a Simple Gameserver (node.js)](https://agones.dev/site/docs/tutorials/simple-gameserver-nodejs/).

## Building

If you want to modify the source code and/or build an updated container image, run `make build` from this directory.
This will run the `docker build` command with the correct context.

This example uses a Docker container to host the SDK and example it inside a container so that no special build
tools need to be installed on your system.

## Testing locally with Docker

If you want to run the example locally, you need to start an instance of the SDK-server. To run an SDK-server for
120 seconds, run
```bash
$ cd ../../build; make run-sdk-conformance-local TIMEOUT=120 TESTS=ready,watch,health,gameserver
```

In a separate terminal, while the SDK-server is still running, build and start a container with the example gameserver:
```bash
$ make build
$ make run
```

The example can also be run via docker:
```
$ docker run --network=host us-docker.pkg.dev/agones-images/examples/nodejs-simple-server:0.10
```
Or directly via npm:
```
$ npm start
```

You will see the output like the following:
```
docker run --network=host us-docker.pkg.dev/agones-images/examples/nodejs-simple-server:0.10

> @ start /home/server/examples/nodejs-simple
> node src/index.js

Connecting to the SDK server...
...connected to SDK server
```

To see help, pass `--help` as the argument (use the preferred command below, all are equivalent):
```
$ make args="--help" run
$ docker run --network=host us-docker.pkg.dev/agones-images/examples/nodejs-simple-server:0.10 --help
$ npm start -- --help
```

You can optionally specify how long the server will stay up once the basic tests are complete with the `--timeout` option.
To do this pass arguments through, e.g. to increase the shutdown duration to 120 seconds:
```
$ make args="--timeout=120" run
$ docker run --network=host us-docker.pkg.dev/agones-images/examples/nodejs-simple-server:0.10 --timeout=120
$ npm start -- --timeout=120
```

To make run indefinitely use the special timeout value of 0:
```
$ make args="--timeout=0" run
$ docker run --network=host us-docker.pkg.dev/agones-images/examples/nodejs-simple-server:0.10 --timeout=0
$ npm start -- --timeout=0
```
