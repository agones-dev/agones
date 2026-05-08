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

declare class Beta {
	getCounterCount(key: string): Promise<number>;

	incrementCounter(key: string, amount: number): Promise<void>;

	decrementCounter(key: string, amount: number): Promise<void>;

	setCounterCount(key: string, amount: number): Promise<void>;

	getCounterCapacity(key: string): Promise<number>;

	setCounterCapacity(key: string, amount: number): Promise<void>;

	getListCapacity(key: string): Promise<number>;

	setListCapacity(key: string, amount: number): Promise<void>;

	listContains(key: string, value: string): Promise<boolean>;

	getListLength(key: string): Promise<number>;

	getListValues(key: string): Promise<string[]>;

	appendListValue(key: string, value: string): Promise<void>;

	deleteListValue(key: string, value: string): Promise<void>;
}

export default Beta;
