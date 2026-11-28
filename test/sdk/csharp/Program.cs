// Copyright Contributors to Agones a Series of LF Projects, LLC.
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
using Agones;
using Grpc.Core;

using var sdk = new AgonesSDK();
{
    sdk.WatchGameServer((gameServer) =>
    {
        Console.WriteLine("Received GameServer update");
        Console.WriteLine(gameServer);
    });
}

{
    var status = await sdk.ReadyAsync();
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Could not send ready message. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

{
    var status = await sdk.ReserveAsync(5);
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Could not send reserve command. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

{
    var status = await sdk.AllocateAsync();
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Err sending allocate request. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

{
    var status = await sdk.HealthAsync();
    if (status.StatusCode != StatusCode.OK)
    {
        Console.Error.WriteLine(
            $"Could not send health check. StatusCode={status.StatusCode}, Detail={status.Detail}");
        Environment.Exit(1);
    }
}

{
    var gameServer = await sdk.GetGameServerAsync();
    Console.WriteLine("Successfully GameServer");
    Console.WriteLine(gameServer);
    {
        var status = await sdk.SetLabelAsync("creationTimestamp",
            gameServer.ObjectMeta.CreationTimestamp.ToString());
        if (status.StatusCode != StatusCode.OK)
        {
            Console.Error.WriteLine(
                $"Could not set label. StatusCode={status.StatusCode}, Detail={status.Detail}");
            Environment.Exit(1);
        }
    }
    {
        var status = await sdk.SetAnnotationAsync("UID", gameServer.ObjectMeta.Uid);
        if (status.StatusCode != StatusCode.OK)
        {
            Console.Error.WriteLine(
                $"Could not set annotation. StatusCode={status.StatusCode}, Detail={status.Detail}");
            Environment.Exit(1);
        }
    }
}

// CSHARP_SDK_BETA=true runs the Counters and Lists tests against the deprecated Beta SDK interface.
// Remove it along with the Beta SDK.
var countsAndLists = Environment.GetEnvironmentVariable("CSHARP_SDK_BETA") == "true"
    ? CountsAndListsClient.From(sdk.Beta())
    : CountsAndListsClient.From(sdk);

// Tests are expected to run sequentially on the same pre-defined Counter in the localsdk server
{
    var client = countsAndLists;
    var key = "rooms";

    {
        var wantCount = 1;
        var task = client.GetCounterCountAsync(key);
        task.Wait();
        var gotCount = task.Result;
        if (wantCount != gotCount)
        {
            Console.Error.WriteLine($"Counter count should be {wantCount}, but is {gotCount}");
            Environment.Exit(1);
        }
    }

    {
        var wantCount = 10;
        var increment = 9;
        try
        {
            var task = client.IncrementCounterAsync(key, increment);
            task.Wait();
        }
        catch (Exception e)
        {
            Console.Error.WriteLine($"IncrementCounterAsync for Counter {key} did not increment. Error: {e.ToString()}");
            Environment.Exit(1);
        }

        var getTask = client.GetCounterCountAsync(key);
        getTask.Wait();
        var gotCount = getTask.Result;
        if (wantCount != gotCount)
        {
            Console.Error.WriteLine($"Counter count should be {wantCount}, but is {gotCount}");
            Environment.Exit(1);
        }
    }

    {
        var wantCount = 5;
        var decrement = 5;
        try
        {
            var task = client.DecrementCounterAsync(key, decrement);
            task.Wait();
        }
        catch (Exception e)
        {
            Console.Error.WriteLine($"DecrementCounterAsync for Counter {key} did not decrement. Error: {e.ToString()}");
            Environment.Exit(1);
        }

        var getTask = client.GetCounterCountAsync(key);
        getTask.Wait();
        var gotCount = getTask.Result;
        if (wantCount != gotCount)
        {
            Console.Error.WriteLine($"Counter count should be {wantCount}, but is {gotCount}");
            Environment.Exit(1);
        }
    }

    {
        var wantCount = 3;
        try
        {
            var task = client.SetCounterCountAsync(key, wantCount);
            task.Wait();
        }
        catch (Exception e)
        {
            Console.Error.WriteLine($"SetCounterCountAsync for Counter {key} did not set. Error: {e.ToString()}");
            Environment.Exit(1);
        }

        var getTask = client.GetCounterCountAsync(key);
        getTask.Wait();
        var gotCount = getTask.Result;
        if (wantCount != gotCount)
        {
            Console.Error.WriteLine($"Counter count should be {wantCount}, but is {gotCount}");
            Environment.Exit(1);
        }
    }

    {
        var wantCapacity = 10;
        var task = client.GetCounterCapacityAsync(key);
        task.Wait();
        var gotCapacity = task.Result;
        if (wantCapacity != gotCapacity)
        {
            Console.Error.WriteLine($"Counter capacity should be {wantCapacity}, but is {gotCapacity}");
            Environment.Exit(1);
        }
    }


    {
        var wantCapacity = 0;
        try
        {
            var task = client.SetCounterCapacityAsync(key, wantCapacity);
            task.Wait();
        }
        catch (Exception e)
        {
            Console.Error.WriteLine($"SetCounterCapacityAsync for Counter {key} did not set. Error: {e.ToString()}");
            Environment.Exit(1);
        }

        var getTask = client.GetCounterCapacityAsync(key);
        getTask.Wait();
        var gotCapacity = getTask.Result;
        if (wantCapacity != gotCapacity)
        {
            Console.Error.WriteLine($"Counter capacity should be {wantCapacity}, but is {gotCapacity}");
            Environment.Exit(1);
        }
    }
}

// Tests are expected to run sequentially on the same pre-defined List in the localsdk server
{
    var client = countsAndLists;
    var key = "players";

    {
        var wantCapacity = 100;
        var task = client.GetListCapacityAsync(key);
        task.Wait();
        var gotCapacity = task.Result;
        if (wantCapacity != gotCapacity)
        {
            Console.Error.WriteLine($"List capacity should be {wantCapacity}, but is {gotCapacity}");
            Environment.Exit(1);
        }
    }

    {
        var wantCapacity = 10;
        try
        {
            var task = client.SetListCapacityAsync(key, wantCapacity);
            task.Wait();
        }
        catch (Exception e)
        {
            Console.Error.WriteLine($"SetListCapacityAsync for List {key} did not set. Error: {e.ToString()}");
            Environment.Exit(1);
        }

        var getTask = client.GetListCapacityAsync(key);
        getTask.Wait();
        var gotCapacity = getTask.Result;
        if (wantCapacity != gotCapacity)
        {
            Console.Error.WriteLine($"List capacity should be {wantCapacity}, but is {gotCapacity}");
            Environment.Exit(1);
        }
    }

    {
        var value = "foo";
        var want = false;
        var task = client.ListContainsAsync(key, value);
        task.Wait();
        var got = task.Result;
        if (want != got)
        {
            Console.Error.WriteLine($"ListContains expected {want} for value {value}, but got {got}");
            Environment.Exit(1);
        }
        value = "test1";
        want = true;
        task = client.ListContainsAsync(key, value);
        task.Wait();
        got = task.Result;
        if (want != got)
        {
            Console.Error.WriteLine($"ListContains expected {want} for value {value}, but got {got}");
            Environment.Exit(1);
        }
    }

    {
        IList<string> wantValues = new List<string> { "test0", "test1", "test2" };
        var task = client.GetListValuesAsync(key);
        task.Wait();
        var gotValues = task.Result;
        var equal = Enumerable.SequenceEqual(wantValues, gotValues);
        if (!equal)
        {
            var wantStr = String.Join(" ", wantValues);
            var gotStr = String.Join(" ", gotValues);
            Console.Error.WriteLine($"List values should be {wantStr}, but is {gotStr}");
            Environment.Exit(1);
        }
    }

    {
        var addValue = "test3";
        IList<string> wantValues = new List<string> { "test0", "test1", "test2", "test3" };
        try
        {
            var task = client.AppendListValueAsync(key, addValue);
            task.Wait();
        }
        catch (Exception e)
        {
            Console.Error.WriteLine($"AppendListValueAsync for List {key} did not append {addValue}. Error: {e.ToString()}");
            Environment.Exit(1);
        }

        var getTask = client.GetListValuesAsync(key);
        getTask.Wait();
        var gotValues = getTask.Result;
        var equal = Enumerable.SequenceEqual(wantValues, gotValues);
        if (!equal)
        {
            var wantStr = String.Join(" ", wantValues);
            var gotStr = String.Join(" ", gotValues);
            Console.Error.WriteLine($"List values should be {wantStr}, but is {gotStr}");
            Environment.Exit(1);
        }
    }

    {
        var removeValue = "test2";
        IList<string> wantValues = new List<string> { "test0", "test1", "test3" };
        try
        {
            var task = client.DeleteListValueAsync(key, removeValue);
            task.Wait();
        }
        catch (Exception e)
        {
            Console.Error.WriteLine($"DeleteListValueAsync for List {key} did not delete {removeValue}. Error: {e.ToString()}");
            Environment.Exit(1);
        }

        var getTask = client.GetListValuesAsync(key);
        getTask.Wait();
        var gotValues = getTask.Result;
        var equal = Enumerable.SequenceEqual(wantValues, gotValues);
        if (!equal)
        {
            var wantStr = String.Join(" ", wantValues);
            var gotStr = String.Join(" ", gotValues);
            Console.Error.WriteLine($"List values should be {wantStr}, but is {gotStr}");
            Environment.Exit(1);
        }
    }
}

var shutDownStatus = await sdk.ShutDownAsync();
if (shutDownStatus.StatusCode != StatusCode.OK)
{
    Console.Error.WriteLine(
        $"Could not shutdown GameServer. StatusCode={shutDownStatus.StatusCode}, Detail={shutDownStatus.Detail}");
    Environment.Exit(1);
}

Console.WriteLine("Finish all tests");

// CountsAndListsClient lets the Counters and Lists tests run against either the Stable SDK or the
// deprecated Beta SDK, which share the same method signatures.
sealed class CountsAndListsClient
{
    public required Func<string, Task<long>> GetCounterCountAsync { get; init; }
    public required Func<string, long, Task> IncrementCounterAsync { get; init; }
    public required Func<string, long, Task> DecrementCounterAsync { get; init; }
    public required Func<string, long, Task> SetCounterCountAsync { get; init; }
    public required Func<string, Task<long>> GetCounterCapacityAsync { get; init; }
    public required Func<string, long, Task> SetCounterCapacityAsync { get; init; }
    public required Func<string, Task<long>> GetListCapacityAsync { get; init; }
    public required Func<string, long, Task> SetListCapacityAsync { get; init; }
    public required Func<string, string, Task<bool>> ListContainsAsync { get; init; }
    public required Func<string, Task<int>> GetListLengthAsync { get; init; }
    public required Func<string, Task<IList<string>>> GetListValuesAsync { get; init; }
    public required Func<string, string, Task> AppendListValueAsync { get; init; }
    public required Func<string, string, Task> DeleteListValueAsync { get; init; }

    public static CountsAndListsClient From(IAgonesSDK sdk) => new()
    {
        GetCounterCountAsync = sdk.GetCounterCountAsync,
        IncrementCounterAsync = sdk.IncrementCounterAsync,
        DecrementCounterAsync = sdk.DecrementCounterAsync,
        SetCounterCountAsync = sdk.SetCounterCountAsync,
        GetCounterCapacityAsync = sdk.GetCounterCapacityAsync,
        SetCounterCapacityAsync = sdk.SetCounterCapacityAsync,
        GetListCapacityAsync = sdk.GetListCapacityAsync,
        SetListCapacityAsync = sdk.SetListCapacityAsync,
        ListContainsAsync = sdk.ListContainsAsync,
        GetListLengthAsync = sdk.GetListLengthAsync,
        GetListValuesAsync = sdk.GetListValuesAsync,
        AppendListValueAsync = sdk.AppendListValueAsync,
        DeleteListValueAsync = sdk.DeleteListValueAsync,
    };

#pragma warning disable CS0618 // The Beta Counters and Lists methods are deprecated.
    public static CountsAndListsClient From(IAgonesBetaSDK beta) => new()
    {
        GetCounterCountAsync = beta.GetCounterCountAsync,
        IncrementCounterAsync = beta.IncrementCounterAsync,
        DecrementCounterAsync = beta.DecrementCounterAsync,
        SetCounterCountAsync = beta.SetCounterCountAsync,
        GetCounterCapacityAsync = beta.GetCounterCapacityAsync,
        SetCounterCapacityAsync = beta.SetCounterCapacityAsync,
        GetListCapacityAsync = beta.GetListCapacityAsync,
        SetListCapacityAsync = beta.SetListCapacityAsync,
        ListContainsAsync = beta.ListContainsAsync,
        GetListLengthAsync = beta.GetListLengthAsync,
        GetListValuesAsync = beta.GetListValuesAsync,
        AppendListValueAsync = beta.AppendListValueAsync,
        DeleteListValueAsync = beta.DeleteListValueAsync,
    };
#pragma warning restore CS0618
}
