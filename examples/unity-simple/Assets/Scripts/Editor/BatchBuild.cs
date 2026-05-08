// Copyright Contributors to Agones a Series of LF Projects, LLC.
// All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

using System.IO;
using UnityEditor;

namespace AgonesExample.Editor
{
    public class BatchBuild
    {
        [MenuItem("Build Tool/Build Server")]
        public static void BuildServer()
        {
            string[] scenes = new[] { "Assets/Scenes/AgonesUnitySimple.unity" };
            string dir = "Builds/Server";

            Directory.CreateDirectory(dir);

            BuildPlayerOptions option = new BuildPlayerOptions
            {
                scenes = scenes,
                locationPathName = dir + "/UnitySimpleServer.x86_64",
                target = BuildTarget.StandaloneLinux64,
                options = BuildOptions.EnableHeadlessMode
            };
            BuildPipeline.BuildPlayer(option);
        }

        [MenuItem("Build Tool/Build Client")]
        public static void BuildClient()
        {
            string[] scenes = new[] { "Assets/Scenes/AgonesUnitySimple.unity" };
            string dir = "Builds/Client";

            Directory.CreateDirectory(dir);

            var target = BuildTarget.StandaloneWindows64;
#if UNITY_EDITOR_OSX
            target = BuildTarget.StandaloneOSX;
#elif !UNITY_EDITOR_WIN && !UNITY_EDITOR_OSX
            target = BuildTarget.StandaloneLinux64;
#endif
            BuildPlayerOptions option = new BuildPlayerOptions
            {
                scenes = scenes,
                locationPathName = dir + "/UnitySimpleClient.exe",
                target = target,
                options = BuildOptions.None
            };
            BuildPipeline.BuildPlayer(option);
        }
    }
}

