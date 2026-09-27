# Simple Game Server

A very simple "toy" game server created to demo and test running a UDP and/or
TCP game server on Agones.

To learn how to deploy your edited version of go server to gcp, please check
out this link: [Edit Your First Game Server (Go)](https://agones.dev/site/docs/getting-started/edit-first-gameserver-go/),
or also look at the [`Makefile`](./Makefile).

## Interacting with the Server

When the server receives a text packet, it will send back "ACK:<text content>"
for UDP as an echo or "ACK TCP:<text content>" for TCP.

There are some text commands you can send the server to affect its behavior:

| Command                | Behavior                                                                                 |
|------------------------|------------------------------------------------------------------------------------------|
| "EXIT"                 | Causes the game server to exit cleanly calling `os.Exit(0)`                              |
| "UNHEALTHY"            | Stopping sending health checks                                                           |
| "GAMESERVER"           | Sends back the game server name                                                          |
| "READY"                | Marks the server as Ready                                                                |
| "ALLOCATE"             | Allocates the game server                                                                |
| "RESERVE"              | Reserves the game server after the specified duration                                    |
| "WATCH"                | Instructs the game server to log changes to the resource                                 |
| "LABEL"                | Sets the specified label on the game server resource                                     |
| "CRASH"                | Causes the game server to exit / crash immediately                                       |
| "ANNOTATION"           | Sets the specified annotation on the game server resource                                |
| "GET_COUNTER_COUNT"    | Returns a count of a given Counter                                                       |
| "INCREMENT_COUNTER"    | Increases the count of the given Counter by the given nonnegative integer amount         |
| "DECREMENT_COUNTER"    | Decreases the count of the given Counter by the given nonnegative integer amount         |
| "SET_COUNTER_COUNT"    | Sets a count of the given Counter to the given amount                                    |
| "GET_COUNTER_CAPACITY" | Returns the Capacity of the given Counter                                                |
| "SET_COUNTER_CAPACITY" | Sets the Capacity of the given Counter to the given amount                               |
| "GET_LIST_CAPACITY"    | Returns the Capacity of the given List                                                   |
| "SET_LIST_CAPACITY"    | Returns if the List was set to a new Capacity successfully (true) or not (false)         |
| "LIST_CONTAINS"        | Returns true if the given value is in the given List, false otherwise                    |
| "GET_LIST_LENGTH"      | Returns the length (number of values) of the given List as a string                      |
| "GET_LIST_VALUES"      | Return the values in the given List as a comma delineated string                         |
| "APPEND_LIST_VALUE"    | Returns if the given value was successfully added to the List (true) or not (false)      |
| "DELETE_LIST_VALUE"    | Rreturns if the given value was successfully deleted from the List (true) or not (false) |

## Configuration

The server has a few configuration options that can be set via command line
flags. Some can also be set using environment variables.

| Flag                                   | Environment Variable | Default |
|----------------------------------------|----------------------|---------|
| port                                   | PORT                 | 7654    |
| passthrough                            | PASSTHROUGH          | false   |
| ready                                  | READY                | true    |
| automaticShutdownDelaySec              | _n/a_                | 0       |
| automaticShutdownDelayMin (deprecated) | _n/a_                | 0       |
| readyDelaySec                          | _n/a_                | 0       |
| readyIterations                        | _n/a_                | 0       |
| udp                                    | UDP                  | true    |
| tcp                                    | TCP                  | false   |

