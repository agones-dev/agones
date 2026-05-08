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
using System.Text;

namespace Agones.Model
{
    /// <summary>
    /// StatusPort
    /// </summary>
    public class StatusPort : IEquatable<StatusPort>
    {
        /// <summary>
        /// Initializes a new instance of the <see cref="StatusPort" /> class.
        /// </summary>
        public StatusPort(IReadOnlyDictionary<string, object> data)
        {
            this.Name = (string) data["name"];
            this.Port = (Int64) data["port"];
        }

        public string Name { get; }
        public Int64 Port { get; }

        /// <summary>
        /// Returns the string presentation of the object
        /// </summary>
        /// <returns>String presentation of the object</returns>
        public override string ToString()
        {
            var sb = new StringBuilder();
            sb.Append("class StatusPort {\n");
            sb.Append("  Name: ").Append(Name).Append("\n");
            sb.Append("  Port: ").Append(Port).Append("\n");
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
            return this.Equals(input as StatusPort);
        }

        /// <summary>
        /// Returns true if StatusPort instances are equal
        /// </summary>
        /// <param name="input">Instance of StatusPort to be compared</param>
        /// <returns>Boolean</returns>
        public bool Equals(StatusPort input)
        {
            if (input == null)
                return false;

            return
                (
                    this.Name == input.Name ||
                    (this.Name != null &&
                     this.Name.Equals(input.Name))
                ) &&
                (
                    this.Port == input.Port ||
                    (this.Port.Equals(input.Port))
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
                if (this.Name != null)
                    hashCode = hashCode * 59 + this.Name.GetHashCode();
                hashCode = hashCode * 59 + this.Port.GetHashCode();
                return hashCode;
            }
        }
    }
}