# Simple Unity Example

This is a very simple "unity server" that doesn't do much other than show how the SDK works in Unity.

## Prerequisites
This example is working on
```
Unity Editor: Unity 6000.6.7f1 or later
OS: Windows, macOS or Linux
```

Building the server requires the **Linux Dedicated Server Build Support** module for the Unity Editor.

## Running a Server
There are a few steps.

### Install and configure Agones on Kubernetes
Check out [these instructions](https://agones.dev/site/docs/installation/).

### Building a Server and Docker Image
`make build-image` builds the server with the `unity` CLI (installed with Unity Hub) and then builds the Docker image:
```
$ make build-image
```

To build only the server, run `make build-server`, or open this folder with the Unity Editor and click on the
`Build Tool/Build Server` menu item in the menu bar. The builds are created in a `Builds/Server` folder.

### Running the Server
```
$ kubectl create -f gameserver.yaml
```

## Calling an SDK API via a Client

### Building a Client
* Open this folder with UnityEditor.
* Click on the `Build Tool/Build Client` menu item in the menu bar.

### How to use a Client
* Run `Builds/Client/UnitySimpleClient.exe`.
* Set `Address` and `Port` text fields to GameServer's one. You can see these with the following command.
    ```
    $ kubectl get gs 
    NAME                        STATE   ADDRESS         PORT   NODE       AGE
    unity-simple-server-z7nln   Ready   192.168.*.*     7854   node-name  1m
    ```
* Click on the `Change Server` Button.
* Set any text to a center text filed and click the `Send` button.
  * The Client will send the text to the Server.

  When a Server receives a text, it will send back "Echo : $text" as an echo.
  And an SDK API will be executed in a Server by the following rules.

    | Sending Text(Client) | SDK API(Server)       |
    |----------------------|-----------------------|
    | Allocate             | Allocate()            |
    | Label $1 $2          | SetLabel($1, $2)      |
    | Annotation $1 $2     | SetAnnotation($1, $2) |
    | Shutdown             | Shutdown()            |
    | GameServer           | GameServer()          |
