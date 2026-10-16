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

#include "Classes.h"

#if WITH_DEV_AUTOMATION_TESTS

#include "Misc/AutomationTest.h"
#include "Serialization/JsonReader.h"
#include "Serialization/JsonSerializer.h"

IMPLEMENT_SIMPLE_AUTOMATION_TEST(FAgonesObjectMetaKeysTest, "Agones.Json.ObjectMetaKeys",
	EAutomationTestFlags::EditorContext | EAutomationTestFlags::ClientContext |
	EAutomationTestFlags::ServerContext | EAutomationTestFlags::CommandletContext |
	EAutomationTestFlags::EngineFilter)

bool FAgonesObjectMetaKeysTest::RunTest(const FString& Parameters)
{
	const FString Json = TEXT("{\"annotations\":{\"agones.dev/display-name\":\"caf\\u00e9\",\"agones.dev/missing\":null},")
		TEXT("\"labels\":{\"app.kubernetes.io/name\":\"demo\",\"agones.dev/missing\":null}}");
	TSharedPtr<FJsonObject> JsonObject;
	const TSharedRef<TJsonReader<>> Reader = TJsonReaderFactory<>::Create(Json);
	if (!TestTrue(TEXT("Metadata JSON deserializes"), FJsonSerializer::Deserialize(Reader, JsonObject)))
	{
		return false;
	}

	const FObjectMeta Metadata(JsonObject);
	TestEqual(TEXT("Annotation count excludes null values"), Metadata.Annotations.Num(), 1);
	TestEqual(TEXT("Label count excludes null values"), Metadata.Labels.Num(), 1);
	TestFalse(TEXT("Null annotation is omitted"), Metadata.Annotations.Contains(TEXT("agones.dev/missing")));
	TestFalse(TEXT("Null label is omitted"), Metadata.Labels.Contains(TEXT("agones.dev/missing")));

	const FString* Annotation = Metadata.Annotations.Find(TEXT("agones.dev/display-name"));
	const FString* Label = Metadata.Labels.Find(TEXT("app.kubernetes.io/name"));
	if (!TestNotNull(TEXT("Annotation key is preserved"), Annotation) ||
		!TestNotNull(TEXT("Label key is preserved"), Label))
	{
		return false;
	}
	TestEqual(TEXT("Annotation value is preserved"), *Annotation, FString(TEXT("caf\u00e9")));
	TestEqual(TEXT("Label value is preserved"), *Label, FString(TEXT("demo")));
	return true;
}

IMPLEMENT_SIMPLE_AUTOMATION_TEST(FAgonesStatusCounterKeysTest, "Agones.Json.StatusCounterKeys",
	EAutomationTestFlags::EditorContext | EAutomationTestFlags::ClientContext |
	EAutomationTestFlags::ServerContext | EAutomationTestFlags::CommandletContext |
	EAutomationTestFlags::EngineFilter)

bool FAgonesStatusCounterKeysTest::RunTest(const FString& Parameters)
{
	const FString Json = TEXT("{\"counters\":{\"players\":{\"count\":3,\"capacity\":10},")
		TEXT("\"caf\\u00e9\":{\"count\":2,\"capacity\":5},\"missing\":null}}");
	TSharedPtr<FJsonObject> JsonObject;
	const TSharedRef<TJsonReader<>> Reader = TJsonReaderFactory<>::Create(Json);
	if (!TestTrue(TEXT("Status JSON deserializes"), FJsonSerializer::Deserialize(Reader, JsonObject)))
	{
		return false;
	}

	const FStatus Status(JsonObject);
	TestEqual(TEXT("Counter count excludes null values"), Status.Counters.Num(), 2);
	TestFalse(TEXT("Null counter is omitted"), Status.Counters.Contains(TEXT("missing")));

	const FCounter* Players = Status.Counters.Find(TEXT("players"));
	const FCounter* UnicodeCounter = Status.Counters.Find(TEXT("caf\u00e9"));
	if (!TestNotNull(TEXT("ASCII counter key is preserved"), Players) ||
		!TestNotNull(TEXT("Unicode counter key is preserved"), UnicodeCounter))
	{
		return false;
	}
	TestEqual(TEXT("ASCII counter count is preserved"), Players->Count, int64(3));
	TestEqual(TEXT("ASCII counter capacity is preserved"), Players->Capacity, int64(10));
	TestEqual(TEXT("Unicode counter count is preserved"), UnicodeCounter->Count, int64(2));
	TestEqual(TEXT("Unicode counter capacity is preserved"), UnicodeCounter->Capacity, int64(5));
	return true;
}

#endif
