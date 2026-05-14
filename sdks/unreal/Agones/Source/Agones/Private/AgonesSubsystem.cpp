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

#include "AgonesSubsystem.h"

#include "Engine/Engine.h"
#include "Engine/GameInstance.h"
#include "Engine/World.h"
#include "HttpModule.h"
#include "Interfaces/IHttpResponse.h"
#include "JsonObjectConverter.h"
#include "Policies/CondensedJsonPrintPolicy.h"
#include "TimerManager.h"
#include "IWebSocket.h"
#include "Runtime/Launch/Resources/Version.h"
#include "WebSocketsModule.h"

DEFINE_LOG_CATEGORY_STATIC(LogAgones, Log, Log);

#if ENGINE_MAJOR_VERSION > 4
typedef UTF8CHAR UTF8FromType;
#else
typedef ANSICHAR UTF8FromType;
#endif

template <typename CharType = TCHAR, typename PrintPolicy = TCondensedJsonPrintPolicy<TCHAR>>
bool JsonObjectToJsonString(const TSharedRef<FJsonObject>& JsonObject, FString& OutJson, int32 Indent = 0)
{
	TSharedRef<TJsonWriter<CharType, PrintPolicy>> JsonWriter = TJsonWriterFactory<CharType, PrintPolicy>::Create(&OutJson, Indent);
	bool bSuccess = FJsonSerializer::Serialize(JsonObject, JsonWriter);
	JsonWriter->Close();
	return bSuccess;
}

void UAgonesSubsystem::UpdateCounter(const FString& Key, const int64* Count, const int64* Capacity, const int64* CountDiff, FUpdateCounterDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	TSharedRef<FJsonObject> JsonObject = MakeShareable(new FJsonObject());

	if (Count)
	{
		JsonObject->SetNumberField(TEXT("count"), *Count);
	}
	if (Capacity)
	{
		JsonObject->SetNumberField(TEXT("capacity"), *Capacity);
	}
	if (CountDiff)
	{
		JsonObject->SetNumberField(TEXT("countDiff"), *CountDiff);
	}

	FString Json;
	if (!JsonObjectToJsonString(JsonObject, Json))
	{
		ErrorDelegate.ExecuteIfBound({ TEXT("Failed to serializing request") });
		return;
	}

	FHttpRequestRef Request = BuildAgonesRequest(FString::Format(TEXT("v1beta1/counters/{0}"), { Key }), FHttpVerb::Patch, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded)
		{
			if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

FHttpRequestRef UAgonesSubsystem::BuildAgonesRequest(const FString Path, const FHttpVerb Verb, const FString Content)
{
	FHttpModule* Http = &FHttpModule::Get();
	FHttpRequestRef Request = Http->CreateRequest();

	Request->SetURL(FString::Format(
		TEXT("http://localhost:{0}/{1}"), 
{FStringFormatArg(HttpPort), FStringFormatArg(Path)}	
	));
	Request->SetVerb(Verb.ToString());
	Request->SetHeader(TEXT("Content-Type"), TEXT("application/json"));
	Request->SetHeader(TEXT("User-Agent"), TEXT("X-UnrealEngine-Agent"));
	Request->SetHeader(TEXT("Accepts"), TEXT("application/json"));
	Request->SetContentAsString(Content);
	return Request;
}

UAgonesSubsystem* UAgonesSubsystem::Get(const UObject* WorldContext)
{
	UGameInstance* GameInstance{ nullptr };
	auto World = GEngine->GetWorldFromContextObject(WorldContext, EGetWorldErrorMode::LogAndReturnNull);
	if (World)
	{
		GameInstance = World->GetGameInstance();
	}

	return GameInstance ? GameInstance->GetSubsystem<UAgonesSubsystem>() : nullptr;
}

bool UAgonesSubsystem::ShouldCreateSubsystem(UObject *Outer) const 
{
	return UE_SERVER;
}

void UAgonesSubsystem::Initialize(FSubsystemCollectionBase& Collection)
{
	Super::Initialize(Collection);

	const FTickerDelegate TickDelegate = FTickerDelegate::CreateUObject(this, &UAgonesSubsystem::Tick);
	TickHandle = FTSTicker::GetCoreTicker().AddTicker(TickDelegate);
	TimerManager = MakeUnique<FTimerManager>();

	if (!bDisableAutoHealthPing) 
	{
		HealthPing(HealthRateSeconds);
	}

	if (!bDisableAutoConnect)
	{
		Connect();
	}
}

void UAgonesSubsystem::Deinitialize()
{
	Super::Deinitialize();

	FTSTicker::GetCoreTicker().RemoveTicker(TickHandle);

	if (WatchWebSocket != nullptr && WatchWebSocket->IsConnected())
	{
		WatchWebSocket->Close();
	}
}

bool UAgonesSubsystem::Tick(float DeltaTime)
{
	TimerManager->Tick(DeltaTime);
	return true;
}

void UAgonesSubsystem::HealthPing(const float RateSeconds)
{
	if (RateSeconds <= 0.0f)
	{
		return;
	}

	FTimerDelegate TimerDel;
	TimerDel.BindUObject(this, &UAgonesSubsystem::Health, FHealthDelegate(), FAgonesErrorDelegate());
	GetTimerManager()->ClearTimer(HealthTimerHandler);
	GetTimerManager()->SetTimer(HealthTimerHandler, TimerDel, RateSeconds, true);
}

void UAgonesSubsystem::Connect()
{
	FGameServerDelegate SuccessDel;
	SuccessDel.BindUFunction(this, FName("ConnectSuccess"));
	FTimerDelegate ConnectDel;
	ConnectDel.BindUObject(this, &UAgonesSubsystem::GameServer, SuccessDel, FAgonesErrorDelegate());
	GetTimerManager()->ClearTimer(ConnectDelTimerHandle);
	GetTimerManager()->SetTimer(ConnectDelTimerHandle, ConnectDel, 5.f, true);
}

void UAgonesSubsystem::ConnectSuccess(const FGameServerResponse GameServerResponse)
{
	GetTimerManager()->ClearTimer(ConnectDelTimerHandle);
	Ready({}, {});
	ConnectedDelegate.Broadcast(GameServerResponse);
}

void UAgonesSubsystem::Ready(const FReadyDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("ready");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::GameServer(const FGameServerDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("gameserver", FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FGameServerResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::EnsureWebSocketConnection()
{
	if (WatchWebSocket == nullptr)
	{
		if (!FModuleManager::LoadModulePtr<FWebSocketsModule>(TEXT("WebSockets")))
		{
			return;
		}

		TMap<FString, FString> Headers;

		// Make up a WebSocket-Key value. It can be anything!
		Headers.Add(TEXT("Sec-WebSocket-Key"), FGuid::NewGuid().ToString(EGuidFormats::Short));
		Headers.Add(TEXT("Sec-WebSocket-Version"), TEXT("13"));
		Headers.Add(TEXT("User-Agent"), TEXT("X-UnrealEngine-Agent"));

		// Unreal WebSockets are not able to do DNS resolution for localhost for some reason
		// so this is using the IPv4 Loopback Address instead.
		WatchWebSocket = FWebSocketsModule::Get().CreateWebSocket(
			FString::Format(TEXT("ws://127.0.0.1:{0}/watch/gameserver"),
				static_cast<FStringFormatOrderedArguments>(
					TArray<FStringFormatArg, TFixedAllocator<1>>{
						 FStringFormatArg(HttpPort)
					}
				)
			),
			TEXT("")
		);

		WatchWebSocket->OnRawMessage().AddUObject(this, &UAgonesSubsystem::HandleWatchMessage);
	}

	if (WatchWebSocket != nullptr)
	{
		if (!WatchWebSocket->IsConnected())
		{
			WatchWebSocket->Connect();
		}

		// Only start the timer if there is a websocket to check.
		// This timer has nothing to do with health and only matters if the agent is somehow
		// restarted, which would be a failure condition in normal operation.
		if (!EnsureWebSocketTimerHandler.IsValid())
		{
			FTimerDelegate TimerDel;
			TimerDel.BindUObject(this, &UAgonesSubsystem::EnsureWebSocketConnection);
			GetTimerManager()->SetTimer(
				EnsureWebSocketTimerHandler, TimerDel, 15.0f, true);
		}
	}
}

void UAgonesSubsystem::WatchGameServer(const FGameServerDelegate WatchDelegate)
{
	WatchGameServerCallbacks.Add(WatchDelegate);
	EnsureWebSocketConnection();
}

 void UAgonesSubsystem::DeserializeAndBroadcastWatch(FString const& JsonString)
{
	TSharedRef<TJsonReader<TCHAR>> const JsonReader = TJsonReaderFactory<TCHAR>::Create(JsonString);

	TSharedPtr<FJsonObject> JsonObject;
	const TSharedPtr<FJsonObject>* ResultObject = nullptr;

	if (!FJsonSerializer::Deserialize(JsonReader, JsonObject) ||
		!JsonObject.IsValid() ||
		!JsonObject->TryGetObjectField(TEXT("result"), ResultObject) ||
		!ResultObject->IsValid())
	{
		UE_LOG(LogAgones, Error, TEXT("Failed to parse json: %s"), *JsonString);
		return;
	}

	FGameServerResponse const Result = FGameServerResponse(*ResultObject);
	for (FGameServerDelegate const& Callback : WatchGameServerCallbacks)
	{
		if (Callback.IsBound())
		{
			Callback.Execute(Result);
		}
	}
}

void UAgonesSubsystem::HandleWatchMessage(const void* Data, SIZE_T Size, SIZE_T BytesRemaining)
{
	if (BytesRemaining <= 0 && (WatchMessageBuffer.Num() == 0))
	{
		FUTF8ToTCHAR Message(static_cast<const UTF8FromType*>(Data), Size);
		DeserializeAndBroadcastWatch(FString(Message.Length(), Message.Get()));
		return;
	}

	WatchMessageBuffer.Insert(static_cast<const UTF8CHAR*>(Data), Size, WatchMessageBuffer.Num());
	if (BytesRemaining > 0)
	{
		return;
	}

	FUTF8ToTCHAR Message((const UTF8FromType*)WatchMessageBuffer.GetData(), WatchMessageBuffer.Num());
	DeserializeAndBroadcastWatch(FString(Message.Length(), Message.Get()));
	WatchMessageBuffer.Empty();
}

void UAgonesSubsystem::SetLabel(
	const FString& Key, const FString& Value, const FSetLabelDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FKeyValuePair Label = {Key, Value};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Label, Json))
	{
		ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("error serializing key-value pair ({0}: {1}})"),
			static_cast<FStringFormatOrderedArguments>(
			TArray<FStringFormatArg, TFixedAllocator<2>>{
				FStringFormatArg(Key),
				FStringFormatArg(Value)
			})
		)});
		return;
	}

	FHttpRequestRef Request = BuildAgonesRequest("metadata/label", FHttpVerb::Put, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::Health(const FHealthDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("health");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::Shutdown(const FShutdownDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("shutdown");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::SetAnnotation(
	const FString& Key, const FString& Value, const FSetAnnotationDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FKeyValuePair Label = {Key, Value};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Label, Json))
	{
		ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("error serializing key-value pair ({0}: {1}})"),
			static_cast<FStringFormatOrderedArguments>(
				TArray<FStringFormatArg, TFixedAllocator<2>>{
					FStringFormatArg(Key),
					FStringFormatArg(Value)
				}
			)
		)});
		return;
	}

	FHttpRequestRef Request = BuildAgonesRequest("metadata/annotation", FHttpVerb::Put, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::Allocate(const FAllocateDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("allocate");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::Reserve(
	const int64 Seconds, const FReserveDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FDuration Duration = {Seconds};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Duration, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serializing request")});
		return;
	}

	FHttpRequestRef Request = BuildAgonesRequest("reserve", FHttpVerb::Post, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::PlayerConnect(
	const FString PlayerId, const FPlayerConnectDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FAgonesPlayer Player = {PlayerId};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Player, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serializing request")});
		return;
	}

	// TODO(dom) - look at JSON encoding in UE4.
	Json = Json.Replace(TEXT("playerId"), TEXT("playerID"));

	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/connect", FHttpVerb::Post, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FConnectedResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::PlayerDisconnect(
	const FString PlayerId, const FPlayerDisconnectDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FAgonesPlayer Player = {PlayerId};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(Player, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serializing request")});
		return;
	}

	// TODO(dom) - look at JSON encoding in UE4.
	Json = Json.Replace(TEXT("playerId"), TEXT("playerID"));

	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/disconnect", FHttpVerb::Post, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FDisconnectResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::SetPlayerCapacity(
	const int64 Count, const FSetPlayerCapacityDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	const FPlayerCapacity PlayerCapacity = {Count};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(PlayerCapacity, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serializing request")});
		return;
	}

	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/capacity", FHttpVerb::Put, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound({});
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::GetCounter(FString Key, FGetCounterDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest(FString::Format(TEXT("v1beta1/counters/{0}"), {Key}), FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, FHttpResponsePtr HttpResponse, const bool bSucceeded)
		{
			TSharedPtr<FJsonObject> JsonObject;
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FCounterResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::IncrementCounter(FString Key, int64 Amount, FIncrementCounterDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	const auto UpdateSuccessDelegate = FUpdateCounterDelegate::CreateLambda([SuccessDelegate](const FEmptyResponse&)
		{
			SuccessDelegate.ExecuteIfBound({});
		});
	UpdateCounter(Key, nullptr, nullptr, &Amount, UpdateSuccessDelegate, ErrorDelegate);
}

void UAgonesSubsystem::DecrementCounter(FString Key, int64 Amount, FDecrementCounterDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	const int64 NegativeAmount = -Amount;
	const auto UpdateSuccessDelegate = FUpdateCounterDelegate::CreateLambda([SuccessDelegate](const FEmptyResponse&)
		{
			SuccessDelegate.ExecuteIfBound({});
		});
	UpdateCounter(Key, nullptr, nullptr, &NegativeAmount, UpdateSuccessDelegate, ErrorDelegate);
}

void UAgonesSubsystem::SetCounterCount(FString Key, int64 Count, FSetCounterCountDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	const auto UpdateSuccessDelegate = FUpdateCounterDelegate::CreateLambda([SuccessDelegate](const FEmptyResponse&)
		{
			SuccessDelegate.ExecuteIfBound({});
		});
	UpdateCounter(Key, &Count, nullptr, nullptr, UpdateSuccessDelegate, ErrorDelegate);
}

void UAgonesSubsystem::SetCounterCapacity(FString Key, int64 Capacity, FSetCounterCapacityDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	const auto UpdateSuccessDelegate = FUpdateCounterDelegate::CreateLambda([SuccessDelegate](const FEmptyResponse&) 
		{
			SuccessDelegate.ExecuteIfBound({}); 
		});
	UpdateCounter(Key, nullptr, &Capacity, nullptr, UpdateSuccessDelegate, ErrorDelegate);
}

FTimerManager* UAgonesSubsystem::GetTimerManager() const
{
	return TimerManager.Get();
}

void UAgonesSubsystem::GetPlayerCapacity(FGetPlayerCapacityDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/capacity", FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FCountResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::GetPlayerCount(FGetPlayerCountDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/count", FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FCountResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::IsPlayerConnected(
	const FString PlayerId, const FIsPlayerConnectedDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest(
		FString::Format(TEXT("alpha/player/connected/{0}"),
			static_cast<FStringFormatOrderedArguments>(
				TArray<FStringFormatArg, TFixedAllocator<1>>{
					FStringFormatArg(PlayerId)
				}
			)
		),
		FHttpVerb::Get,
		""
	);
	
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FConnectedResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::GetConnectedPlayers(
	const FGetConnectedPlayersDelegate SuccessDelegate, const FAgonesErrorDelegate ErrorDelegate)
{
	FHttpRequestRef Request = BuildAgonesRequest("alpha/player/connected/{0}", FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;

			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}
			
			SuccessDelegate.ExecuteIfBound(FConnectedPlayersResponse(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::GetList(const FString& Key, FListDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	const FString Path = FString::Printf(TEXT("v1beta1/lists/%s"), *Key);
	const FHttpRequestRef Request = BuildAgonesRequest(Path, FHttpVerb::Get, "");
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FList(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::UpdateList(const FString& Key, const FList& List, FListDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(List, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serialize request")});
		return;
	}

	const FString Path = FString::Printf(TEXT("v1beta1/lists/%s"), *Key);
	const FHttpRequestRef Request = BuildAgonesRequest(Path, FHttpVerb::Patch, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FList(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::AddListValue(const FString& Key, const FString& Value, FListDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	const FListValue ListValue = {Value};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(ListValue, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serialize request")});
		return;
	}

	const FString Path = FString::Printf(TEXT("v1beta1/lists/%s:addValue"), *Key);
	const FHttpRequestRef Request = BuildAgonesRequest(Path, FHttpVerb::Post, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FList(JsonObject));
		});
	Request->ProcessRequest();
}

void UAgonesSubsystem::RemoveListValue(const FString& Key, const FString& Value, FListDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate)
{
	const FListValue ListValue = {Value};
	FString Json;
	if (!FJsonObjectConverter::UStructToJsonObjectString(ListValue, Json))
	{
		ErrorDelegate.ExecuteIfBound({TEXT("Failed to serialize request")});
		return;
	}

	const FString Path = FString::Printf(TEXT("v1beta1/lists/%s:removeValue"), *Key);
	const FHttpRequestRef Request = BuildAgonesRequest(Path, FHttpVerb::Post, Json);
	Request->OnProcessRequestComplete().BindWeakLambda(this,
		[SuccessDelegate, ErrorDelegate](FHttpRequestPtr HttpRequest, const FHttpResponsePtr HttpResponse, const bool bSucceeded) {
			TSharedPtr<FJsonObject> JsonObject;
			
			if (!IsValidJsonResponse(JsonObject, bSucceeded, HttpResponse, ErrorDelegate))
			{
				return;
			}

			SuccessDelegate.ExecuteIfBound(FList(JsonObject));
		});
	Request->ProcessRequest();
}

bool UAgonesSubsystem::IsValidResponse(const bool bSucceeded, const FHttpResponsePtr HttpResponse, FAgonesErrorDelegate ErrorDelegate)
{
	if (!bSucceeded)
	{
		ErrorDelegate.ExecuteIfBound({"Unsuccessful Call"});
		return false;
	}

	if (!EHttpResponseCodes::IsOk(HttpResponse->GetResponseCode()))
	{
		ErrorDelegate.ExecuteIfBound(
			{FString::Format(TEXT("Error Code - {0}"),
				static_cast<FStringFormatOrderedArguments>(
					TArray<FStringFormatArg, TFixedAllocator<1>>{
						FStringFormatArg(FString::FromInt(HttpResponse->GetResponseCode()))
					})
				)
			}
		);
		return false;
	}

	return true;
}

bool UAgonesSubsystem::IsValidJsonResponse(TSharedPtr<FJsonObject>& JsonObject, const bool bSucceeded, const FHttpResponsePtr HttpResponse, FAgonesErrorDelegate ErrorDelegate)
{
	if (!IsValidResponse(bSucceeded, HttpResponse, ErrorDelegate))
	{
		return false;
	}

	TSharedPtr<FJsonObject> OutObject;
	const FString Json = HttpResponse->GetContentAsString();
	const TSharedRef<TJsonReader<>> JsonReader = TJsonReaderFactory<>::Create(Json);
	if (!FJsonSerializer::Deserialize(JsonReader, OutObject) || !OutObject.IsValid())
	{
		ErrorDelegate.ExecuteIfBound({FString::Format(TEXT("Failed to parse response - {0}"),
			static_cast<FStringFormatOrderedArguments>(
				TArray<FStringFormatArg, TFixedAllocator<1>>{
					FStringFormatArg(Json)
				})
			)
		});
		return false;
	}

	JsonObject = OutObject.ToSharedRef();
	return true;
}
