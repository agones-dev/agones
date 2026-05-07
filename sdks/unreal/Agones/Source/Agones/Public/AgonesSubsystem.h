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

#pragma once

#include "Classes.h"
#include "Components/ActorComponent.h"
#include "CoreMinimal.h"
#include "Containers/Ticker.h"
#include "Interfaces/IHttpRequest.h"
#include "IWebSocket.h"
#include "Subsystems/GameInstanceSubsystem.h"
#include "TimerManager.h"

#include "AgonesSubsystem.generated.h"

DECLARE_DYNAMIC_DELEGATE_OneParam(FAgonesErrorDelegate, const FAgonesError&, Error);

DECLARE_DYNAMIC_DELEGATE_OneParam(FAllocateDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FGameServerDelegate, const FGameServerResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FGetConnectedPlayersDelegate, const FConnectedPlayersResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FGetPlayerCapacityDelegate, const FCountResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FGetPlayerCountDelegate, const FCountResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FHealthDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FIsPlayerConnectedDelegate, const FConnectedResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FPlayerConnectDelegate, const FConnectedResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FPlayerDisconnectDelegate, const FDisconnectResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FReadyDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FReserveDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FSetAnnotationDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FSetLabelDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FSetPlayerCapacityDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FGetCounterDelegate, const FCounterResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FIncrementCounterDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FDecrementCounterDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FSetCounterCountDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FSetCounterCapacityDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FShutdownDelegate, const FEmptyResponse&, Response);

DECLARE_DYNAMIC_MULTICAST_DELEGATE_OneParam(FConnectedDelegate, const FGameServerResponse&, Response);

DECLARE_DYNAMIC_DELEGATE_OneParam(FListDelegate, const FList&, Response);

class FHttpVerb
{
public:
	enum EVerb
	{
		Get,
		Post,
		Put,
		Patch
	};

	// ReSharper disable once CppNonExplicitConvertingConstructor
	FHttpVerb(const EVerb Verb) : Verb(Verb)
	{
	}

	FString ToString() const
	{
		switch (Verb)
		{
			case Post:
				return TEXT("POST");
			case Put:
				return TEXT("PUT");
			case Patch:
				return TEXT("PATCH");
			case Get:
			default:
				return TEXT("GET");
		}
	}

private:
	const EVerb Verb;
};

/**
 * \brief UAgonesSubsystem object that helps interact with the Agones SDK.
 * See - https://agones.dev/ for more information.
 */
UCLASS(Config = Game, defaultconfig)
class AGONES_API UAgonesSubsystem : public UGameInstanceSubsystem
{
	GENERATED_BODY()

public:
	/**
	 * \brief Retrieve subsystem component from game instance. Subsystem exist only on server builds! See ShouldCreateSubsystem. 
	 * \param WorldContext - context of the world
	 */
	UFUNCTION(BlueprintPure, meta = (WorldContext = "WorldContext"), DisplayName = "Get Agones Subsystem", Category = "Agones | Utility")
	static UAgonesSubsystem* Get(const UObject* WorldContext);

	/**
	 * \brief HttpPort is the default Agones HTTP port to use.
	 */
	UPROPERTY(EditAnywhere, Category = Agones, Config)
	FString HttpPort = "9358";

	/**
	 * \brief HealthRateSeconds is the frequency to send Health calls. Value of 0 will disable auto health calls.
	 */
	UPROPERTY(EditAnywhere, BlueprintReadWrite, Category = Agones, Config)
	float HealthRateSeconds = 10.f;

	/**
	 * \brief bDisableAutoHealthPing will stop call to HealthPing() during initialization
	 */
	UPROPERTY(EditAnywhere, BlueprintReadWrite, Category = Agones, Config)
	bool bDisableAutoHealthPing;

	/**
	 * \brief bDisableAutoConnect will stop auto connecting (calling GamesServer and Ready) during initialization.
	 */
	UPROPERTY(EditAnywhere, BlueprintReadWrite, Category = Agones, Config)
	bool bDisableAutoConnect;

	/**
	 * \brief ConnectedDelegate will be called once the Connect func gets a successful response from GameServer.
	 */
	UPROPERTY(BlueprintAssignable, Category = Agones)
	FConnectedDelegate ConnectedDelegate;

	/**
	 * \brief ShouldCreateSubsystem is a built in subsystem function that is called before Initialize.
	 */
	virtual bool ShouldCreateSubsystem(UObject* Outer) const override;

	/**
	* \brief Initialize is a built in subsystem function for initialization of subsystem.
	* \param Collection can help initialize subsystem dependencies
	*/
	virtual void Initialize(FSubsystemCollectionBase& Collection) override;

	/**
	* \brief Deinitialize is a built in subsystem function called during destruction of GameInstance.
	*/
	virtual void Deinitialize() override;

	/**
	* \brief Tick is a built in ticker function called every frame.
	*/
	bool Tick(float DeltaTime);

	/**
	 * \brief HealthPing loops calling the Health endpoint.
	 * \param RateSeconds rate at which the Health endpoint should be called.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Utility")
	void HealthPing(float RateSeconds);

	/**
	 * \brief Connect will call /gameserver till a successful response then call /ready
	 * a delegate is called with the gameserver response after /ready call is made.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Utility")
	void Connect();

	/**
	 * \brief Allocate self marks this gameserver as Allocated.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Lifecycle")
	void Allocate(FAllocateDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief GameServer retrieve the GameServer details.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Configuration")
	void GameServer(FGameServerDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief WatchGameServer subscribes a delegate to be called whenever game server details change.
	 * \param WatchDelegate - Called every time the game server data changes.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Configuration")
	void WatchGameServer(FGameServerDelegate WatchDelegate);

	/**
	 * \brief Health sends a ping to the health check to indicate that this server is healthy.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Lifecycle")
	void Health(FHealthDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief Ready marks the Game Server as ready to receive connections.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Lifecycle")
	void Ready(FReadyDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief Reserve marks the Game Server as Reserved for a given duration.
	 * \param Seconds - Seconds that the Game Server will be reserved.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Lifecycle")
	void Reserve(int64 Seconds, FReserveDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief SetAnnotation sets a metadata annotation on the `GameServer` with the prefix 'agones.dev/sdk-'
	 * calling SetAnnotation("foo", "bar", {}, {}) will result in the annotation "agones.dev/sdk-foo: bar".
	 * \param Key
	 * \param Value
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Metadata")
	void SetAnnotation(const FString& Key, const FString& Value, FSetAnnotationDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief SetLabel sets a metadata label on the `GameServer` with the prefix 'agones.dev/sdk-'
	 * calling SetLabel("foo", "bar", {}, {}) will result in the label "agones.dev/sdk-foo: bar".
	 * \param Key
	 * \param Value
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Metadata")
	void SetLabel(const FString& Key, const FString& Value, FSetLabelDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief Shutdown marks the Game Server as ready to shutdown
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Lifecycle")
	void Shutdown(FShutdownDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Alpha] GetConnectedPlayers returns the list of the currently connected player ids.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Alpha | Player Tracking")
	void GetConnectedPlayers(FGetConnectedPlayersDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Alpha] GetPlayerCapacity gets the last player capacity that was set through the SDK.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Alpha | Player Tracking")
	void GetPlayerCapacity(FGetPlayerCapacityDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Alpha] GetPlayerCount returns the current player count
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Alpha | Player Tracking")
	void GetPlayerCount(FGetPlayerCountDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Alpha] IsPlayerConnected returns if the playerID is currently connected to the GameServer.
	 * \param PlayerId - PlayerID of player to check.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Alpha | Player Tracking")
	void IsPlayerConnected(FString PlayerId, FIsPlayerConnectedDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Alpha] PlayerConnect increases the SDK’s stored player count by one, and appends this playerID to status.players.id.
	 * \param PlayerId - PlayerID of connecting player.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Alpha | Player Tracking")
	void PlayerConnect(FString PlayerId, FPlayerConnectDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Alpha] PlayerDisconnect Decreases the SDK’s stored player count by one, and removes the playerID from
	 * status.players.id.
	 *
	 * \param PlayerId - PlayerID of disconnecting player.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Alpha | Player Tracking")
	void PlayerDisconnect(FString PlayerId, FPlayerDisconnectDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Alpha] SetPlayerCapacity changes the player capacity to a new value.
	 * \param Count - Capacity of game server.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Alpha | Player Tracking")
	void SetPlayerCapacity(int64 Count, FSetPlayerCapacityDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Beta] GetCounter return counter (count and capacity) associated with a Key.
	 * \param Key - Key to counter value
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Beta | Counters")
	void GetCounter(FString Key, FGetCounterDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Beta] IncrementCounter incremenets counter associated with a Key by 1.
	 * \param Key - Key to counter value
	 * \param Amount - Amount that would be added to count.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Beta | Counters")
	void IncrementCounter(FString Key, int64 Amount, FIncrementCounterDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Beta] DecrementCounter decremenets counter associated with a Key by 1.
	 * \param Key - Key to counter value
	 * \param Amount - Amount that would be decremented from count.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Beta | Counters")
	void DecrementCounter(FString Key, int64 Amount, FDecrementCounterDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Beta] SetCounterCount set counter count associated with a Key.
	 * \param Key - Key to counter value
	 * \param Count - Active sessions count.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Beta | Counters")
	void SetCounterCount(FString Key, int64 Count, FSetCounterCountDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief [Beta] SetCounterCount set counter capacity associated with a Key.
	 * \param Key - Key to counter value
	 * \param Capacity - Capacity of game server.
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Beta | Counters")
	void SetCounterCapacity(FString Key, int64 Capacity, FSetCounterCapacityDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief GetList retrieves the list’s properties with the key, returns the list’s information.
	 * \param Key - Key to list value
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Beta | Lists")
	void GetList(const FString& Key, FListDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief UpdateList updates the list’s properties with the key, such as its capacity and values,
	 * and returns the updated list details. Use AddListValue or RemoveListValue for modifying the List.Values field.
	 * \param Key - Key to list value
	 * \param List - The list to replace the server list with
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Beta | Lists")
	void UpdateList(const FString& Key, const FList& List, FListDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);
	
	/**
	 * \brief AddListValue adds a new value to a list with the key and returns the list with this addition.
	 * \param Key - Key to list value
	 * \param Value - Value to add
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Beta | Lists")
	void AddListValue(const FString& Key, const FString& Value, FListDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	/**
	 * \brief RemoveListValue removes a value from the list with the key players and returns updated list.
	 * \param Key - Key to list value
	 * \param Value - Value to remove
	 * \param SuccessDelegate - Called on Successful call.
	 * \param ErrorDelegate - Called on Unsuccessful call.
	 */
	UFUNCTION(BlueprintCallable, Category = "Agones | Beta | Lists")
	void RemoveListValue(const FString& Key, const FString& Value, FListDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);
	
private:
	FTimerManager* GetTimerManager() const;

	DECLARE_DELEGATE_OneParam(FUpdateCounterDelegate, const FEmptyResponse&);
	void UpdateCounter(const FString& Key, const int64* Count, const int64* Capacity, const int64* CountDiff, FUpdateCounterDelegate SuccessDelegate, FAgonesErrorDelegate ErrorDelegate);

	FHttpRequestRef BuildAgonesRequest(
		FString Path = "", const FHttpVerb Verb = FHttpVerb::Post, const FString Content = "{}");

	void HandleWatchMessage(const void* Data, SIZE_T Size, SIZE_T BytesRemaining);

	void DeserializeAndBroadcastWatch(FString const& JsonString);

	void EnsureWebSocketConnection();

	FTSTicker::FDelegateHandle TickHandle;

	FTimerHandle ConnectDelTimerHandle;

	FTimerHandle HealthTimerHandler;

	FTimerHandle EnsureWebSocketTimerHandler;

	TUniquePtr<FTimerManager> TimerManager;

	TSharedPtr<IWebSocket> WatchWebSocket;

	TArray<UTF8CHAR> WatchMessageBuffer;

	TArray<FGameServerDelegate> WatchGameServerCallbacks;
	
	static bool IsValidResponse(const bool bSucceeded, const FHttpResponsePtr HttpResponse, FAgonesErrorDelegate ErrorDelegate);

	static bool IsValidJsonResponse(TSharedPtr<FJsonObject>& JsonObject, const bool bSucceeded, const FHttpResponsePtr HttpResponse, FAgonesErrorDelegate ErrorDelegate);

	UFUNCTION(BlueprintInternalUseOnly)
	void ConnectSuccess(FGameServerResponse GameServerResponse);
};
