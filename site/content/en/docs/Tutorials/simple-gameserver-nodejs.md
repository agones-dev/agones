---
title: "Tutorial Build and Run a Simple Gameserver (node.js)"
linkTitle: "Build and Run a Simple Gameserver (node.js)"
date: 2019-07-25T17:34:37Z
publishDate: 2019-08-01T10:00:00Z
description: >
  This tutorial describes how to use the Agones node.js SDK in a simple node.js gameserver.
---

## Objectives
- Run a simple gameserver
- Understand how the simple gameserver uses the Agones node.js SDK
- Build a customized version of the simple gameserver
- Run your customized simple gameserver

## Prerequisites
1. [Docker](https://www.docker.com/get-started/)
2. Agones installed on GKE
3. kubectl properly configured
4. A local copy of the [Agones repository](https://github.com/agones-dev/agones/tree/{{< release-branch >}})
5. A repository for Docker images, such as [Docker Hub](https://hub.docker.com/) or [GC Container Registry](https://cloud.google.com/container-registry/)

To install on GKE, follow the install instructions (if you haven't already) at
[Setting up a Google Kubernetes Engine (GKE) cluster]({{< ref "/docs/Installation/Creating Cluster/gke.md" >}}).
Also complete the "Installing Agones" instructions on the same page.

While not required, you may wish to review the [Create a Game Server]({{< relref "../Getting Started/create-gameserver.md" >}}),
[Create a Game Server Fleet]({{< relref "../Getting Started/create-fleet.md" >}}), and/or [Edit a Game Server]({{< relref "../Getting Started/edit-first-gameserver-go.md" >}}) quickstarts.

### 1. Run the simple gameserver

First, run the pre-built version of the simple gameserver and take note of the name that was created:

```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/nodejs-simple/gameserver.yaml
GAMESERVER_NAME=$(kubectl get gs -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

The game server sets up the Agones SDK, calls `sdk.ready()` to inform Agones that it is ready to serve traffic,
prints a message every 10 seconds, and then calls `sdk.shutdown()` after a minute to indicate that the gameserver
is going to exit.

You can follow along with the lifecycle of the gameserver by running

```bash
kubectl logs ${GAMESERVER_NAME} nodejs-simple -f
```

which should produce output similar to
```
> @ start /home/server/examples/nodejs-simple
> node src/index.js

node.js Game Server has started!
Setting a label
(node:20) [DEP0005] DeprecationWarning: Buffer() is deprecated due to security and usability issues. Please use the Buffer.alloc(), Buffer.allocUnsafe(), or Buffer.from() methods instead.
Setting an annotation
Marking server as ready...
...marked Ready
GameServer Update:
	name: nodejs-simple-9bw4g 
	state: Scheduled
GameServer Update:
	name: nodejs-simple-9bw4g 
	state: RequestReady
GameServer Update:
	name: nodejs-simple-9bw4g 
	state: RequestReady
GameServer Update:
	name: nodejs-simple-9bw4g 
	state: Ready
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 10 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 20 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
GameServer Update:
	name: nodejs-simple-9bw4g 
	state: Ready
Running for 30 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 40 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 50 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
GameServer Update:
	name: nodejs-simple-9bw4g 
	state: Ready
Running for 60 seconds!
Shutting down after 60 seconds...
...marked for Shutdown
GameServer Update:
	name: nodejs-simple-9bw4g 
	state: Shutdown
Health ping sent
GameServer Update:
	name: nodejs-simple-9bw4g 
	state: Shutdown
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 70 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 80 seconds!
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 90 seconds!
```

If everything goes as expected, the gameserver will exit automatically after about a minute. 

In some cases, the gameserver goes into an unhealthy state, in which case it will be restarted indefinitely. 
If this happens, you can manually remove it by running
```bash
kubectl delete gs ${GAMESERVER_NAME}
```

### 2. Build a simple gameserver

Change directories to your local agones/examples/nodejs-simple directory. To experiment with the SDK, open up
`src/index.js` in your favorite editor and change the interval at which the gameserver calls `sdk.health()` from
2 seconds to 20 seconds by modifying the lines in the health ping handler to be

```js
setInterval(() => {
	agonesSDK.health();
	console.log('Health ping sent');
}, 20000);
```

Next build a new docker image by running
```bash
cd examples/nodejs-simple
REPOSITORY=<your-repository> # e.g. gcr.io/agones-images
make build REPOSITORY=${REPOSITORY}
```

Once the container has been built, push it to your repository
```bash
docker push ${REPOSITORY}/nodejs-simple-server:0.1
```

### 3. Run the customized gameserver

Now it is time to deploy your newly created gameserver container into your Agones cluster. 

First, you need to edit `examples/nodejs-simple/gameserver.yaml` to point to your new image:

```yaml
containers:
- name: nodejs-simple
  image: $(REPOSITORY)/nodejs-simple-server:0.1
  imagePullPolicy: Always
```

Then, deploy your gameserver

```bash
kubectl create -f gameserver.yaml
GAMESERVER_NAME=$(kubectl get gs -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

Again, follow along with the lifecycle of the gameserver by running

```bash
kubectl logs ${GAMESERVER_NAME} nodejs-simple -f
```

which should produce output similar to
```
> @ start /home/server/examples/nodejs-simple
> node src/index.js

node.js Game Server has started!
Setting a label
(node:20) [DEP0005] DeprecationWarning: Buffer() is deprecated due to security and usability issues. Please use the Buffer.alloc(), Buffer.allocUnsafe(), or Buffer.from() methods instead.
Setting an annotation
Marking server as ready...
...marked Ready
GameServer Update:
	name: nodejs-simple-qkpqn 
	state: Scheduled
GameServer Update:
	name: nodejs-simple-qkpqn 
	state: Scheduled
GameServer Update:
	name: nodejs-simple-qkpqn 
	state: RequestReady
GameServer Update:
	name: nodejs-simple-qkpqn 
	state: Ready
Running for 10 seconds!
GameServer Update:
	name: nodejs-simple-qkpqn 
	state: Unhealthy
Health ping sent
Running for 20 seconds!
GameServer Update:
	name: nodejs-simple-qkpqn 
	state: Unhealthy
Running for 30 seconds!
Health ping sent
Running for 40 seconds!
Running for 50 seconds!
GameServer Update:
	name: nodejs-simple-qkpqn 
	state: Unhealthy
Health ping sent
Shutting down after 60 seconds...
...marked for Shutdown
Running for 60 seconds!
Running for 70 seconds!
Health ping sent
Running for 80 seconds!
GameServer Update:
	name: nodejs-simple-qkpqn 
	state: Unhealthy
Running for 90 seconds!
```

with the slower healthcheck interval, the gameserver gets automatically marked an `Unhealthy` by Agones. 

To finish, clean up the gameserver by manually removing it
```bash
kubectl delete gs ${GAMESERVER_NAME}
```
