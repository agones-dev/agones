---
title: "Go Game Server Client SDK"
linkTitle: "Go"
date: 2019-05-17T10:17:50Z
weight: 30
description: "This is the Go version of the Agones Game Server Client SDK. "
---

Check the [Client SDK Documentation]({{< relref "_index.md" >}}) for more details on each of the SDK functions and how to run the SDK locally.

## SDK Functionality

| Area            | Action              | Implemented |
|-----------------|---------------------|-------------|
| Lifecycle       | Ready               | ✔️          |
| Lifecycle       | Health              | ✔️          |
| Lifecycle       | Reserve             | ✔️          |
| Lifecycle       | Allocate            | ✔️          |
| Lifecycle       | Shutdown            | ✔️          |
| Configuration   | GameServer          | ✔️          |
| Configuration   | Watch               | ✔️          |
| Metadata        | SetAnnotation       | ✔️          |
| Metadata        | SetLabel            | ✔️          |
| Counters        | GetCounterCount     | ✔️          |
| Counters        | SetCounterCount     | ✔️          |
| Counters        | IncrementCounter    | ✔️          |
| Counters        | DecrementCounter    | ✔️          |
| Counters        | SetCounterCapacity  | ✔️          |
| Counters        | GetCounterCapacity  | ✔️          |
| Lists           | AppendListValue     | ✔️          |
| Lists           | DeleteListValue     | ✔️          |
| Lists           | SetListCapacity     | ✔️          |
| Lists           | GetListCapacity     | ✔️          |
| Lists           | ListContains        | ✔️          |
| Lists           | GetListLength       | ✔️          |
| Lists           | GetListValues       | ✔️          |

## Installation

`go get` the source, {{< ghlink href="sdks/go" >}}directly from GitHub{{< /ghlink >}}

## Usage

Review the [GoDoc](https://pkg.go.dev/agones.dev/agones/sdks/go) for usage instructions
