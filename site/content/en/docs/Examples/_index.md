---
title: "Examples"
date: 2019-01-03T05:05:47Z
description: "List of available code examples"
weight: 120
---

## Configuration files

These are full examples for each of the resource types of Agones

- {{< ghlink href="examples/gameserver.yaml" >}}Full GameServer Configuration{{< /ghlink >}}
- {{< ghlink href="examples/fleet.yaml" >}}Full Fleet Configuration{{< /ghlink >}}
- {{< ghlink href="examples/gameserverallocation.yaml" >}}Full GameServer Allocation Configuration{{< /ghlink >}}
- {{< ghlink href="examples/fleetautoscaler.yaml" >}}Full Autoscaler Configuration with Buffer Strategy{{< /ghlink >}}
- {{< ghlink href="examples/webhookfleetautoscaler.yaml" >}}Full Autoscaler Configuration with Webhook Strategy{{< /ghlink >}}
- {{< ghlink href="examples/webhookfleetautoscalertls.yaml" >}}Full Autoscaler Configuration with Webhook Strategy + TLS{{< /ghlink >}}

## Game server implementations

These are all examples of simple game server implementations, that integrate the Agones game server SDK.

* {{< ghlink href="examples/simple-game-server" >}}Simple gameserver{{< /ghlink >}} (Go) - simple server that
  responds to TCP connections or UDP packets on the same port.
* {{< ghlink href="examples/cpp-simple" >}}CPP Simple{{< /ghlink >}} (C++) - C++ example that starts up, stays
  healthy and then shuts down after 60 seconds.
* {{< ghlink href="examples/nodejs-simple" >}}Node.js Simple{{< /ghlink >}} (Node.js) -
  A simple Node.js example that marks itself as ready, sets some labels and then shutsdown.
* {{< ghlink href="examples/rust-simple" >}}Rust Simple{{< /ghlink >}} (Rust) -
  A simple Rust example that marks itself as ready, sets some labels and then shutsdown.
* {{< ghlink href="examples/unity-simple" >}}Unity Simple{{< /ghlink >}} (Unity3d)  -
  This is a very simple "unity server" that doesn't do much other than show how the SDK works in Unity.
* {{< ghlink href="examples/xonotic" >}}Xonotic{{< /ghlink >}} - Wraps the SDK around the open source FPS game
  [Xonotic](http://www.xonotic.org) and hosts it on Agones.
* {{< ghlink href="examples/supertuxkart" >}}SuperTuxKart{{< /ghlink >}} \- Wraps the SDK around the open source
  racing game [SuperTuxKart](https://supertuxkart.net/), and hosts it on Agones.
 - {{< ghlink href="examples/custom-controller" >}}Custom-Controller{{< /ghlink >}} (Go) - A custom controller for Agones designed to log GameServer details on insert, update, or delete, demonstrating how to build your own controller. 

## Building on top of Agones

- {{< ghlink href="examples/crd-client" >}}Agones API Usage Example{{< /ghlink >}} (Go) -
  This service provides an example of using the [Agones API](https://pkg.go.dev/agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1)
  to create a GameServer.

## Game Server Allocation

- {{< ghlink href="examples/allocation-endpoint" >}}Allocation Endpoint Example{{< /ghlink >}} - An Agones clusters 
  aware proxy, built on Google Cloud.

## Integrations with other projects

* [googleforgames/space-agon](https://github.com/googleforgames/space-agon) - Space Agon is a demo of Agones and 
  Open Match with a browser based game.
* [googleforgames/global-multiplayer-demo](https://github.com/googleforgames/global-multiplayer-demo) - A demo of a global scale multiplayer game using Agones, Open Match, Unreal Engine 5 and multiple Google Cloud products.
