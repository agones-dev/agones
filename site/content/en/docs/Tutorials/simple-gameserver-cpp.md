---
title: "Tutorial Build and Run a Simple Gameserver (C++)"
linkTitle: "Build and Run a Simple Gameserver (C++)"
date: 2019-07-24T05:52:12Z
publishDate: 2019-08-01T10:00:00Z
description: >
  This tutorial describes how to use the Agones C++ SDK in a simple C++ gameserver.
---

## Objectives
- Run a simple gameserver
- Understand how the simple gameserver uses the Agones C++ SDK
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
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/cpp-simple/gameserver.yaml
GAMESERVER_NAME=$(kubectl get gs -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

The game server sets up the Agones SDK, calls `SDK::Ready()` to inform Agones that it is ready to serve traffic,
prints a message every 10 seconds, and then calls `SDK::Shutdown()` after a minute to indicate that the gameserver
is going to exit.

You can follow along with the lifecycle of the gameserver by running

```bash
kubectl logs ${GAMESERVER_NAME} cpp-simple -f
```

which should produce output similar to


```
C++ Game Server has started!
Getting the instance of the SDK!
Attempting to connect...
...handshake complete.
Setting a label
Starting to watch GameServer updates...
Health ping sent
Setting an annotation
Marking server as ready...
...marked Ready
Getting GameServer details...
GameServer name: cpp-simple-tlgzp
Running for 0 seconds !
GameServer Update:
	name: cpp-simple-tlgzp
	state: Scheduled
GameServer Update:
	name: cpp-simple-tlgzp
	state: RequestReady
GameServer Update:
	name: cpp-simple-tlgzp
	state: RequestReady
GameServer Update:
	name: cpp-simple-tlgzp
	state: Ready
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 10 seconds !
Health ping sent
...
GameServer Update:
	name: cpp-simple-2mtdc
	state: Ready
Shutting down after 60 seconds...
...marked for Shutdown
Running for 60 seconds !
Health ping sent
GameServer Update:
	name: cpp-simple-2mtdc
	state: Shutdown
GameServer Update:
	name: cpp-simple-2mtdc
	state: Shutdown
Health ping failed
Health ping failed
Health ping failed
Health ping failed
Running for 70 seconds !
Health ping failed
Health ping failed
Health ping failed
Health ping failed
Health ping failed
Running for 80 seconds !
Health ping failed
Health ping failed
Health ping failed
Health ping failed
Health ping failed
```

If everything goes as expected, the gameserver will exit automatically after about a minute. 

In some cases, the gameserver goes into an unhealthy state, in which case it will be restarted indefinitely. 
If this happens, you can manually remove it by running
```bash
kubectl delete gs ${GAMESERVER_NAME}
```

### 2. Run a fleet of simple gameservers

Next, run a fleet of gameservers 
```bash
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/cpp-simple/fleet.yaml
FLEET_NAME=$(kubectl get fleets -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

You can again inspect the output of an individual gameserver (which will look the same as above), but what is more
interesting is to watch the set of all gameservers over time. Each gameserver exits after about a minute, but a fleet
is responsible for keeping a sufficient number of gameservers in the `Ready` state. So as each gameserver exits, it
is replaced by a new one. You can see this in action by running

```bash
watch "kubectl get gameservers"
```

which should show how gameservers are constantly transitioning from `Scheduled` to `Ready` to `Shutdown` before
disappearing.  

When you are finished watching the fleet produce new gameservers you should remove the fleet by running
```bash
kubectl delete fleet ${FLEET_NAME}
```

### 3. Build a simple gameserver

Change directories to your local agones/examples/cpp-simple directory. To experiment with the SDK, open up `server.cc`
in your favorite editor and change the interval at which the gameserver calls `SDK::Health` from 2 seconds to 20
seconds by modifying the line in `DoHealth` to be

```c++
std::this_thread::sleep_for(std::chrono::seconds(20));
```

Next build a new docker image by running
```bash
cd examples/cpp-simple
REPOSITORY=<your-repository> # e.g. gcr.io/agones-images
make build REPOSITORY=${REPOSITORY}
```

The multi-stage Dockerfile will pull down all of the dependencies needed to build the image. Note that it is normal
for this to take several minutes to complete.

Once the container has been built, push it to your repository
```bash
docker push ${REPOSITORY}/cpp-simple-server:0.6
```

### 4. Run the customized gameserver

Now it is time to deploy your newly created gameserver container into your Agones cluster. 

First, you need to edit `examples/cpp-simple/gameserver.yaml` to point to your new image:

```yaml
containers:
- name: cpp-simple
  image: $(REPOSITORY)/cpp-simple-server:0.6
  imagePullPolicy: Always # add for development
```

Then, deploy your gameserver

```bash
kubectl create -f gameserver.yaml
GAMESERVER_NAME=$(kubectl get gs -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

Again, follow along with the lifecycle of the gameserver by running

```bash
kubectl logs ${GAMESERVER_NAME} cpp-simple -f
```

which should produce output similar to

```
C++ Game Server has started!
Getting the instance of the SDK!
Attempting to connect...
...handshake complete.
Setting a label
Health ping sent
Starting to watch GameServer updates...
Setting an annotation
Marking server as ready...
...marked Ready
Getting GameServer details...
GameServer name: cpp-simple-f255n
Running for 0 seconds !
GameServer Update:
	name: cpp-simple-f255n
	state: Scheduled
GameServer Update:
	name: cpp-simple-f255n
	state: Scheduled
GameServer Update:
	name: cpp-simple-f255n
	state: RequestReady
GameServer Update:
	name: cpp-simple-f255n
	state: Ready
Running for 10 seconds !
GameServer Update:
	name: cpp-simple-f255n
	state: Unhealthy
Health ping sent
Running for 20 seconds !
GameServer Update:
	name: cpp-simple-f255n
	state: Unhealthy
Running for 30 seconds !
Health ping sent
Running for 40 seconds !
Running for 50 seconds !
GameServer Update:
	name: cpp-simple-f255n
	state: Unhealthy
Health ping sent
Shutting down after 60 seconds...
...marked for Shutdown
Running for 60 seconds !
Running for 70 seconds !
Health ping sent
Running for 80 seconds !
GameServer Update:
	name: cpp-simple-f255n
	state: Unhealthy
Running for 90 seconds !
Health ping sent
```

with the slower healthcheck interval, the gameserver gets automatically marked an `Unhealthy` by Agones. 

To finish, clean up the gameserver by manually removing it
```bash
kubectl delete gs ${GAMESERVER_NAME}
```
