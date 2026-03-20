# Agones Unity Client SDK

## Contributing
Thank you for your interest in contributing to the Agones Unity client library. Setting up a local workspace to help develop the Agones Game Server Unity Client SDK involves cloning the Agones source code, having a host Unity project, and allowing the Unity project to run the external tests in the Unity test runner.

### A copy of Agones source code
If you don't already have a local copy of the Agones source code you can either (1) clone the official [googleforgames/agones repository](https://github.com/agones-dev/agones) for quicker access or (2) create a fork then clone your fork. Make note of where you clone the project to for the next setup steps!

### Setting up the Test Agones Unity SDK Project
In the directory `test/sdk/unity` there is a Test Agones Unity SDK project. This is the project you should add to your Unity Hub, to be able to develop the Agones Unity SDK. You can then open up the project in the Unity Editor application. Once open in the Unity Editor you can run the Agones Unity SDK test suite.

Please be aware that when making contributions to the Agones Unity SDK you will be making modifications to directly to files under `Packages/Agones Unity SDK/` in Unity. The way this works is because the Test Agones Unity SDK project's `Package/manifest.json` uses a relative path to pull in the code directly from `sdks/unity`. For those new to Unity package development this may not be intuitive!

Tests covering contributions is always encouraged. You are now ready to develop the Agones Unity SDK using the Test Agones Unity SDK project!
