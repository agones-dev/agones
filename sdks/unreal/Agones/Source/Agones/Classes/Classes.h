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

#include "CoreMinimal.h"
#include "Dom/JsonObject.h"

#include "Classes.generated.h"

USTRUCT(BlueprintType)
struct FObjectMeta
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString Name;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString Namespace;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString Uid;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString ResourceVersion;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int64 Generation = 0;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int64 CreationTimestamp = 0;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int64 DeletionTimestamp = 0;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	TMap<FString, FString> Annotations;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	TMap<FString, FString> Labels;

	FObjectMeta()
	{
	}

	explicit FObjectMeta(TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetStringField(TEXT("name"), Name);
		JsonObject->TryGetStringField(TEXT("namespace"), Namespace);
		JsonObject->TryGetStringField(TEXT("uid"), Uid);
		JsonObject->TryGetStringField(TEXT("resource_version"), ResourceVersion);
		JsonObject->TryGetNumberField(TEXT("generation"), Generation);
		JsonObject->TryGetNumberField(TEXT("creation_timestamp"), CreationTimestamp);
		JsonObject->TryGetNumberField(TEXT("deletion_timestamp"), DeletionTimestamp);
		const TSharedPtr<FJsonObject>* AnnotationsJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("annotations"), AnnotationsJsonObject))
		{
			for (const auto& Entry : (*AnnotationsJsonObject)->Values)
			{
				if (Entry.Value.IsValid() && !Entry.Value->IsNull())
				{
					FJsonValueString Key = Entry.Key;
					TSharedPtr<FJsonValue> Value = Entry.Value;
					FString AnnotationKey = Key.AsString();
					FString AnnotationValue = Value->AsString();
					Annotations.Add(AnnotationKey, AnnotationValue);
				}
			}
		}
		const TSharedPtr<FJsonObject>* LabelsObject;
		if (JsonObject->TryGetObjectField(TEXT("labels"), LabelsObject))
		{
			for (const auto& Entry : (*LabelsObject)->Values)
			{
				if (Entry.Value.IsValid() && !Entry.Value->IsNull())
				{
					FJsonValueString Key = Entry.Key;
					TSharedPtr<FJsonValue> Value = Entry.Value;
					FString LabelKey = Key.AsString();
					FString LabelValue = Value->AsString();
					Labels.Add(LabelKey, LabelValue);
				}
			}
		}
	}
};

USTRUCT(BlueprintType)
struct FHealth
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	bool bDisabled = false;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int32 PeriodSeconds = 0;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int32 FailureThreshold = 0;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int32 InitialDelaySeconds = 0;

	FHealth()
	{
	}

	explicit FHealth(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetBoolField(TEXT("disabled"), bDisabled);
		JsonObject->TryGetNumberField(TEXT("period_seconds"), PeriodSeconds);
		JsonObject->TryGetNumberField(TEXT("failure_threshold"), FailureThreshold);
		JsonObject->TryGetNumberField(TEXT("initial_delay_seconds"), InitialDelaySeconds);
	}
};

USTRUCT(BlueprintType)
struct FSpec
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FHealth Health;

	FSpec()
	{
	}

	explicit FSpec(const TSharedPtr<FJsonObject> JsonObject)
	{
		const TSharedPtr<FJsonObject>* HealthJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("health"), HealthJsonObject))
		{
			Health = FHealth(*HealthJsonObject);
		}
	}
};

USTRUCT(BlueprintType)
struct FAddress
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString Type;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString Address;

	FAddress()
	{
	}

	explicit FAddress(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetStringField(TEXT("type"), Type);
		JsonObject->TryGetStringField(TEXT("address"), Address);
	}
};

USTRUCT(BlueprintType)
struct FPort
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString Name;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int32 Port = 0;

	FPort()
	{
	}

	explicit FPort(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetStringField(TEXT("name"), Name);
		JsonObject->TryGetNumberField(TEXT("port"), Port);
	}
};

USTRUCT(BlueprintType)
struct FCounter
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int64 Count = 0;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int64 Capacity = 0;

	FCounter()
	{
	}

	explicit FCounter(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetNumberField(TEXT("count"), Count);
		JsonObject->TryGetNumberField(TEXT("capacity"), Capacity);
	}
};

USTRUCT(BlueprintType)
struct FStatus
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString State;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString Address;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	TArray<FAddress> Addresses;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	TArray<FPort> Ports;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	TMap<FString, FCounter> Counters;

	FStatus()
	{
	}

	explicit FStatus(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetStringField(TEXT("state"), State);
		JsonObject->TryGetStringField(TEXT("address"), Address);
		const TArray<TSharedPtr<FJsonValue>>* AddressesArray;
		if (JsonObject->TryGetArrayField(TEXT("addresses"), AddressesArray))
		{
			const int32 ArrLen = AddressesArray->Num();
			for (int32 i = 0; i < ArrLen; ++i)
			{
				const TSharedPtr<FJsonValue>& AddressItem = (*AddressesArray)[i];
				if (AddressItem.IsValid() && !AddressItem->IsNull())
				{
					FAddress NewAddress = FAddress(AddressItem->AsObject());
					Addresses.Add(NewAddress);
				}
			}
		}
		const TArray<TSharedPtr<FJsonValue>>* PortsArray;
		if (JsonObject->TryGetArrayField(TEXT("ports"), PortsArray))
		{
			const int32 ArrLen = PortsArray->Num();
			for (int32 i = 0; i < ArrLen; ++i)
			{
				const TSharedPtr<FJsonValue>& PortItem = (*PortsArray)[i];
				if (PortItem.IsValid() && !PortItem->IsNull())
				{
					FPort Port = FPort(PortItem->AsObject());
					Ports.Add(Port);
				}
			}
		}

		const TSharedPtr<FJsonObject>* CountersObject;
		if (JsonObject->TryGetObjectField(TEXT("counters"), CountersObject))
		{
			for (const auto& CounterPair : CountersObject->Get()->Values)
			{
				const FString& CounterName = CounterPair.Key;
				const TSharedPtr<FJsonValue>& CounterValue = CounterPair.Value;
				if (CounterValue.IsValid() && !CounterValue->IsNull())
				{
					FCounter Counter = FCounter(CounterValue->AsObject());
					Counters.Add(CounterName, Counter);
				}
			}
		}
	}
};

USTRUCT(BlueprintType)
struct FGameServerResponse
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FStatus Status;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FObjectMeta ObjectMeta;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FSpec Spec;

	FGameServerResponse()
	{
	}

	explicit FGameServerResponse(const TSharedPtr<FJsonObject> JsonObject)
	{
		const TSharedPtr<FJsonObject>* ObjectMetaJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("object_meta"), ObjectMetaJsonObject))
		{
			ObjectMeta = FObjectMeta(*ObjectMetaJsonObject);
		}
		const TSharedPtr<FJsonObject>* SpecJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("spec"), SpecJsonObject))
		{
			Spec = FSpec(*SpecJsonObject);
		}
		const TSharedPtr<FJsonObject>* StatusJsonObject;
		if (JsonObject->TryGetObjectField(TEXT("status"), StatusJsonObject))
		{
			Status = FStatus(*StatusJsonObject);
		}
	}
};

USTRUCT(BlueprintType)
struct FKeyValuePair
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString Key;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString Value;
};

USTRUCT(BlueprintType)
struct FDuration
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int64 Seconds = 0;
};

USTRUCT(BlueprintType)
struct FAgonesPlayer
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString PlayerID;
};

USTRUCT(BlueprintType)
struct FPlayerCapacity
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int64 Count = 0;
};

USTRUCT(BlueprintType)
struct FEmptyResponse
{
	GENERATED_BODY()
};

USTRUCT(BlueprintType)
struct FAgonesError
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	FString ErrorMessage;
};

USTRUCT(BlueprintType)
struct FConnectedResponse
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	bool bConnected = false;

	FConnectedResponse()
	{
	}

	explicit FConnectedResponse(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetBoolField(TEXT("bool"), bConnected);
	}
};

USTRUCT(BlueprintType)
struct FDisconnectResponse
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	bool bDisconnected = false;

	FDisconnectResponse()
	{
	}

	explicit FDisconnectResponse(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetBoolField(TEXT("bool"), bDisconnected);
	}
};

USTRUCT(BlueprintType)
struct FCountResponse
{
	GENERATED_BODY()

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	int64 Count = 0;

	FCountResponse()
	{
	}

	explicit FCountResponse(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetNumberField(TEXT("count"), Count);
	}
};

USTRUCT(BlueprintType)
struct FConnectedPlayersResponse
{
	GENERATED_BODY()

	FConnectedPlayersResponse()
	{
	}

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	TArray<FString> ConnectedPlayers;

	explicit FConnectedPlayersResponse(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetStringArrayField(TEXT("list"), ConnectedPlayers);
	}
};

USTRUCT(BlueprintType)
struct FCounterResponse
{
	GENERATED_BODY()

	FCounterResponse()
	{
	}

	UPROPERTY(BlueprintReadOnly, Category = "Agones")
	int64 Count = 0;

	UPROPERTY(BlueprintReadOnly, Category = "Agones")
	int64 Capacity = 0;

	explicit FCounterResponse(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetNumberField(TEXT("count"), Count);
		JsonObject->TryGetNumberField(TEXT("capacity"), Capacity);
	}
};

USTRUCT(BlueprintType)
struct FList
{
	GENERATED_BODY()

	FList()
	{
	}

	UPROPERTY(BlueprintReadOnly, Category = "Agones")
	FString Name;

	UPROPERTY(BlueprintReadOnly, Category = "Agones")
	int64 Capacity = 0;

	UPROPERTY(BlueprintReadOnly, Category="Agones")
	TArray<FString> Values;

	explicit FList(const TSharedPtr<FJsonObject> JsonObject)
	{
		JsonObject->TryGetStringField(TEXT("name"), Name);
		JsonObject->TryGetNumberField(TEXT("capacity"), Capacity);
		JsonObject->TryGetStringArrayField(TEXT("values"), Values);
	}
};

USTRUCT(BlueprintType)
struct FListValue
{
	GENERATED_BODY()
	
	UPROPERTY(BlueprintReadOnly, Category = "Agones")
	FString Value;
};