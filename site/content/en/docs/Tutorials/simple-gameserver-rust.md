---
title: "Tutorial Build and Run a Simple Gameserver (Rust)"
linkTitle: "Build and Run a Simple Gameserver (Rust)"
date: 2019-07-30T07:47:45Z
publishDate: 2019-08-01T10:00:00Z
description: >
  This tutorial describes how to use the Agones Rust SDK in a simple Rust gameserver.
---

## Objectives
- Run a simple gameserver
- Understand how the simple gameserver uses the Agones Rust SDK
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
kubectl create -f https://raw.githubusercontent.com/googleforgames/agones/{{< release-branch >}}/examples/rust-simple/gameserver.yaml
GAMESERVER_NAME=$(kubectl get gs -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

The game server sets up the Agones SDK, calls `sdk.ready()` to inform Agones that it is ready to serve traffic,
prints a message every 10 seconds, and then calls `sdk.shutdown()` after a minute to indicate that the gameserver
is going to exit.

You can follow along with the lifecycle of the gameserver by running

```bash
kubectl logs ${GAMESERVER_NAME} rust-simple -f
```

which should produce output similar to
```
Rust Game Server has started!
Creating SDK instance
Setting a label
Starting to watch GameServer updates...
Health ping sent
Setting an annotation
Marking server as ready...
...marked Ready
Getting GameServer details...
GameServer name: rust-simple-txsc6
Running for 0 seconds
GameServer Update, name: rust-simple-txsc6
GameServer Update, state: Scheduled
GameServer Update, name: rust-simple-txsc6
GameServer Update, state: Scheduled
GameServer Update, name: rust-simple-txsc6
GameServer Update, state: RequestReady
GameServer Update, name: rust-simple-txsc6
GameServer Update, state: Ready
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Health ping sent
Running for 10 seconds
GameServer Update, name: rust-simple-txsc6
GameServer Update, state: Ready
...
Shutting down after 60 seconds...
...marked for Shutdown
Running for 60 seconds
Health ping sent
GameServer Update, name: rust-simple-txsc6
GameServer Update, state: Shutdown
GameServer Update, name: rust-simple-txsc6
GameServer Update, state: Shutdown
...
```

If everything goes as expected, the gameserver will exit automatically after about a minute. 

In some cases, the gameserver goes into an unhealthy state, in which case it will be restarted indefinitely. 
If this happens, you can manually remove it by running
```bash
kubectl delete gs ${GAMESERVER_NAME}
```

### 2. Build a simple gameserver

Change directories to your local agones/examples/rust-simple directory. To experiment with the SDK, open up `main.rs`
in your favorite editor and change the interval at which the gameserver calls `sdk.health()` from 2 seconds to 20
seconds by modifying the line in the thread assigned to `let _health` to be

```rust
thread::sleep(Duration::from_secs(20));
```

Next build a new docker image by running
```bash
cd examples/rust-simple
REPOSITORY=<your-repository> # e.g. gcr.io/agones-images
make build-image REPOSITORY=${REPOSITORY}
```

The multi-stage Dockerfile will pull down all of the dependencies needed to build the image. Note that it is normal
for this to take several minutes to complete.

Once the container has been built, push it to your repository
```bash
docker push ${REPOSITORY}/rust-simple-server:0.4
```

### 3. Run the customized gameserver

Now it is time to deploy your newly created gameserver container into your Agones cluster. 

First, you need to edit `examples/rust-simple/gameserver.yaml` to point to your new image:

```yaml
containers:
- name: rust-simple
  image: $(REPOSITORY)/rust-simple-server:0.4
  imagePullPolicy: Always
```

Then, deploy your gameserver

```bash
kubectl create -f gameserver.yaml
GAMESERVER_NAME=$(kubectl get gs -o go-template --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

Again, follow along with the lifecycle of the gameserver by running

```bash
kubectl logs ${GAMESERVER_NAME} rust-simple -f
```

which should produce output similar to

```
Rust Game Server has started!
Creating SDK instance
Setting a label
Starting to watch GameServer updates...
Health ping sent
Setting an annotation
Marking server as ready...
...marked Ready
Getting GameServer details...
GameServer name: rust-simple-z6lz8
Running for 0 seconds
GameServer Update, name: rust-simple-z6lz8
GameServer Update, state: Scheduled
GameServer Update, name: rust-simple-z6lz8
GameServer Update, state: RequestReady
GameServer Update, name: rust-simple-z6lz8
GameServer Update, state: RequestReady
GameServer Update, name: rust-simple-z6lz8
GameServer Update, state: Ready
Running for 10 seconds
GameServer Update, name: rust-simple-z6lz8
GameServer Update, state: Ready
GameServer Update, name: rust-simple-z6lz8
GameServer Update, state: Unhealthy
Health ping sent
Running for 20 seconds
Running for 30 seconds
Health ping sent
Running for 40 seconds
GameServer Update, name: rust-simple-z6lz8
GameServer Update, state: Unhealthy
Running for 50 seconds
Health ping sent
Shutting down after 60 seconds...
...marked for Shutdown
Running for 60 seconds
Running for 70 seconds
GameServer Update, name: rust-simple-z6lz8
GameServer Update, state: Unhealthy
Health ping sent
Running for 80 seconds
Running for 90 seconds
Health ping sent
Rust Game Server finished.
```

with the slower healthcheck interval, the gameserver gets automatically marked an `Unhealthy` by Agones. 

To finish, clean up the gameserver by manually removing it
```bash
kubectl delete gs ${GAMESERVER_NAME}
```
