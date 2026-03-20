---
title: "Build and Run a Simple Game Server that Connects to an Inference Server"
linkTitle: "GenAI Game Server"
date:
publishDate:
description: >
  This example shows how to set up a simple game server that integrates with your inference server's GenAI endpoints. You can either interact with one GenAI endpoint via TCP, or set up two endpoints to "chat" to each other.
---

## Setting up the GenAI Inference Server

You will need a separate GenAI inference server. This example uses the
[Google for Games GenAI](https://github.com/googleforgames/GenAI-quickstart) as its inference server.
This particular inference server has the request structure for the /chat endpoint:

```
type GenAIRequest struct {
	Context         string  `json:"context,omitempty"`
	Prompt          string  `json:"prompt"`
}
```

### (Optional) Modify the GenAIRequest Structure

If you need a different request structure for your GenAI endpoint, you will need to clone or fork the Agones repository and
modify the above `GenAIRequest struct` in [main.go](https://github.com/agones-dev/agones/blob/main/examples/simple-genai-server/main.go). Update `REPOSITORY` in the
[agones/examples/simple-genai-server/Makefile](https://github.com/agones-dev/agones/blob/main/examples/simple-genai-server/Makefile)
to your own container registry and run `make build && make push` from within the
`agones/examples/simple-genai-server` path. Then modify the `gameserver_*.yaml` to pull the image from
your container registry. If you are making a series of changes you may also want to add
`imagePullPolicy: Always` to the container image in `gameserver_*.yaml`.

## Setting up Agones

To set up the Game Servers on a different cluster than the GenAI inference server follow the
instructions for [creating a cluster](https://agones.dev/site/docs/installation/creating-cluster/).

If you set up a separate cluster for your game servers [install Agones](https://agones.dev/site/docs/installation/install-agones/).
on that cluster, otherwise install Agones into your GenAI inference server cluster.

## Setting up the Game Server

To manually interact with `GenAiEndpoint` via netcat, change the `GEN_AI_ENDPOINT` value in the 
[gameserver_manualchat.yaml](https://github.com/agones-dev/agones/blob/main/examples/simple-genai-server/gameserver_manualchat.yaml) file 
to your inference server’s endpoint. Optionally, include the `GEN_AI_CONTEXT` in the same file that will include the context with 
each chat (post request) that you make to the GenAI endpoint.

To make two clients "chat" to each other, change the `GEN_AI_ENDPOINT` and `SIM_ENDPOINT` values in the [gameserver_autochat.yaml](https://github.com/agones-dev/agones/blob/main/examples/simple-genai-server/gameserver_autochat.yaml) file 
to your inference server's address. You could also create a basic http server that receives requests in the structure noted in the above section,
and returns a predetermined set of responses for the chat. When sending requests, `GEN_AI_CONTEXT` is sent to the `GEN_AI_ENDPOINT`
and `SIM_CONTEXT` is sent to the `SIM_ENDPOINT` as part of the GenAIRequest structure. The default values for `GEN_AI_CONTEXT` and `SIM_CONTEXT` 
are empty strings. The conversation starts with a `PROMPT` that is sent to the GenAI endpoint, which is also an empty string by default. 
`NUM_CHATS` determines how many times messages are sent back and forth, with a default value of 1. 
It's important to note that the chat between two clients is a rolling conversation, meaning that after completing a conversation of length set by `NUM_CHATS`, a new conversation starts. This cycle continues until the game server is deleted using the `kubectl delete` command.

If you want to set up the chat with the npc-chat-api from the [Google for Games GenAI](https://github.com/googleforgames/GenAI-quickstart/tree/main/genai/api/npc_chat_api), 
update the [gameserver_npcchat.yaml](https://github.com/agones-dev/agones/blob/main/examples/simple-genai-server/gameserver_npcchat.yaml) file.
Choose between `GEN_AI_ENDPOINT` or `SIM_ENDPOINT` and set it to `http://genai-api.genai.svc/genai/npc_chat` for the NPC service. 
Mark either `GEN_AI_NPC` or `SIM_NPC` as "true" to indicate it's connected to the NPC service. The `NPCRequest` to the NPC endpoint only sends 
the message (prompt), so any additional context outside of the prompt is ignored. FROM_ID is the entity sending messages to NPC, and 
TO_ID is the entity receiving the message (the NPC ID). A new conversation begins either when it reaches the length `NUM_CHATS` or when it reaches the `STOP_PHRASE`, whichever happens first.
```
type NPCRequest struct {
	Msg    string `json:"message,omitempty"`
	FromId int `json:"from_id,omitempty"`
	ToId   int `json:"to_id,omitempty"`
}
```

## Running the Game Server

Once you have modified the `gameserver_*.yaml` to use your
endpoint(s), apply to your Agones cluster with `kubectl apply -f gameserver_autochat.yaml`,
`kubectl apply -f gameserver_manualchat.yaml`, or `kubectl apply -f gameserver_npcchat`.

Note that if your inference server is in a different cluster you'll want to make sure you're using
the kubectl context that points to your Agones cluster and not the inference cluster.

If you set up the `gameserver_autochat.yaml` or `gameserver_npcchat` the chat will be in the game server logs:

```bash
kubectl logs -f gen-ai-server-auto -c simple-genai-game-server
```

In autochat mode, the game server will stay running forever until the game server is deleted.
While running, we keep `--ConcurrentPlayers` slots of players running - each simulated player
will initiate a chat and then go until they send `--StopPhrase` or until `--NumChats`, whichever
comes first, after which a new player will fill the slot.

If you set up the `gameserver_manualchat.yaml` you can manually send requests to the GenAI endpoint.
Retreive the IP address and port:

```bash
kubectl get gs gen-ai-server-manual -o jsonpath='{.status.address}:{.status.ports[0].port}'
```

You can now send requests to the GenAI endpoint:

{{% alert title="Note" color="info" %}}
If you do not have netcat installed (i.e. you get a response of `nc: command not found`), you can
install netcat by running `sudo apt install netcat`.
{{% /alert %}}


```
nc {IP} {PORT}
Enter your prompt for the GenAI server in the same terminal, and the response will appear here too.
```

After you're done you will need to manually delete the game server `kubectl delete gs gen-ai-server-manual`.
