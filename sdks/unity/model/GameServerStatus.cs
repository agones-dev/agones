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

using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Agones.Model
{
    /// <summary>
    /// GameServerStatus
    /// </summary>
    public class GameServerStatus : IEquatable<GameServerStatus>
    {
        /// <summary>
        /// Initializes a new instance of the <see cref="GameServerStatus" /> class.
        /// </summary>
        public GameServerStatus(IReadOnlyDictionary<string, object> data)
        {
            if (data == null) return;
            
            this.State = (string) data["state"];
            this.Address = (string) data["address"];

            this.Addresses = new List<StatusAddresses>();
            var addressItems = (IReadOnlyList<object>) data["addresses"];
            foreach (var i in addressItems)
            {
                var address = new StatusAddresses((Dictionary<string, object>) i);
                this.Addresses.Add(address);
            }

            this.Ports = new List<StatusPort>();
            var portItems = (IReadOnlyList<object>) data["ports"];
            foreach (var i in portItems)
            {
                var port = new StatusPort((Dictionary<string, object>) i);
                this.Ports.Add(port);
            }
        }

        public string State { get; }
        public string Address { get; }
        public List<StatusAddresses> Addresses { get; }
        public List<StatusPort> Ports { get; }

        /// <summary>
        /// Returns the string presentation of the object
        /// </summary>
        /// <returns>String presentation of the object</returns>
        public override string ToString()
        {
            var sb = new StringBuilder();
            sb.Append("class GameServerStatus {\n");
            sb.Append("  State: ").Append(State).Append("\n");
            sb.Append("  Address: ").Append(Address).Append("\n");
            sb.Append("  Addresses: ").Append(string.Join(";", Addresses)).Append("\n");
            sb.Append("  Ports: ").Append(string.Join(";", Ports)).Append("\n");
            sb.Append("}\n");
            return sb.ToString();
        }

        /// <summary>
        /// Returns true if objects are equal
        /// </summary>
        /// <param name="input">Object to be compared</param>
        /// <returns>Boolean</returns>
        public override bool Equals(object input)
        {
            return this.Equals(input as GameServerStatus);
        }

        /// <summary>
        /// Returns true if GameServerStatus instances are equal
        /// </summary>
        /// <param name="input">Instance of GameServerStatus to be compared</param>
        /// <returns>Boolean</returns>
        public bool Equals(GameServerStatus input)
        {
            if (input == null)
                return false;

            return
                (
                    this.State == input.State ||
                    (this.State != null &&
                     this.State.Equals(input.State))
                ) &&
                (
                    this.Address == input.Address ||
                    (this.Address != null &&
                     this.Address.Equals(input.Address))
                ) &&
                (
                    this.Addresses == input.Addresses ||
                    this.Addresses != null &&
                    this.Addresses.SequenceEqual(input.Addresses)
                ) &&
                (
                    this.Ports == input.Ports ||
                    this.Ports != null &&
                    this.Ports.SequenceEqual(input.Ports)
                );
        }

        /// <summary>
        /// Gets the hash code
        /// </summary>
        /// <returns>Hash code</returns>
        public override int GetHashCode()
        {
            unchecked // Overflow is fine, just wrap
            {
                int hashCode = 41;
                if (this.State != null)
                    hashCode = hashCode * 59 + this.State.GetHashCode();
                if (this.Address != null)
                    hashCode = hashCode * 59 + this.Address.GetHashCode();
                if (this.Addresses != null)
                    hashCode = hashCode * 59 + this.Addresses.GetHashCode();
                if (this.Ports != null)
                    hashCode = hashCode * 59 + this.Ports.GetHashCode();
                return hashCode;
            }
        }
    }
}