# Agones Python SDK

A Python SDK for the [Agones](https://agones.dev) game server platform.

## Prerequisites

- Python >= 3.10
- A running Agones GameServer sidecar (the SDK communicates via gRPC on `localhost:9357`)

## Installation

```bash
pip install agones
```

Or install from source:

```bash
pip install -e sdks/python
```

## Usage

```python
import threading
import time
from agones import AgonesSDK

sdk = AgonesSDK()
sdk.connect()

# Start health checking in a background thread
def health_loop():
    while True:
        sdk.health()
        time.sleep(2)

threading.Thread(target=health_loop, daemon=True).start()

# Mark server as ready
sdk.ready()

# Watch for GameServer updates
sdk.watch_game_server(lambda gs: print(f"State: {gs.status.state}"))

# Set metadata
sdk.set_label("map", "dungeon")

# Alpha: Player tracking
sdk.alpha.player_connect("player-1")
print(sdk.alpha.get_player_count())

# Beta: Counters and lists
sdk.beta.increment_counter("rounds", 1)
sdk.beta.append_list_value("items", "sword")

# Shutdown
sdk.shutdown()
sdk.close()
```

Context manager is also supported:

```python
with AgonesSDK() as sdk:
    sdk.ready()
    # ...
```

## Configuration

| Environment Variable | Default | Description |
|---|---|---|
| `AGONES_SDK_GRPC_HOST` | `localhost` | gRPC server host |
| `AGONES_SDK_GRPC_PORT` | `9357` | gRPC server port |

Or pass `host`/`port` to the constructor:

```python
sdk = AgonesSDK(host="10.0.0.1", port=8080)
```

## Development

### Regenerate gRPC code

```bash
pip install grpcio-tools
bash generate.sh
```

### Run tests

```bash
pip install -e ".[dev]"
pytest tests/
```
