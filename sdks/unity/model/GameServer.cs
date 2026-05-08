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
    /// A GameServer Custom Resource Definition object We will only export those resources that make the most sense. Can always expand to more as needed.
    /// </summary>
    public class GameServer : IEquatable<GameServer>
    {
        /// <summary>
        /// Initializes a new instance of the <see cref="GameServer" /> class.
        /// </summary>
        public GameServer(IReadOnlyDictionary<string, object> data)
        {
            if (data == null) return;

            this.ObjectMeta = new GameServerObjectMeta((Dictionary<string, object>) data["object_meta"]);
            this.Spec = new GameServerSpec((Dictionary<string, object>) data["spec"]);
            // Seems possible that the status field could be null, especially for local SDK tooling,
            // so don't know an exception here if the conversion fails.
            this.Status = new GameServerStatus(data["status"] as Dictionary<string, object>);
        }

        /// <summary>
        /// Gets or Sets ObjectMeta
        /// </summary>
        public GameServerObjectMeta ObjectMeta { get; set; }

        /// <summary>
        /// Gets or Sets Spec
        /// </summary>
        public GameServerSpec Spec { get; set; }

        /// <summary>
        /// Gets or Sets Status
        /// </summary>
        public GameServerStatus Status { get; set; }

        /// <summary>
        /// Returns the string presentation of the object
        /// </summary>
        /// <returns>String presentation of the object</returns>
        public override string ToString()
        {
            var sb = new StringBuilder();
            sb.Append("class GameServer {\n");
            sb.Append("  ObjectMeta: ").Append(ObjectMeta).Append("\n");
            sb.Append("  Spec: ").Append(Spec).Append("\n");
            sb.Append("  Status: ").Append(Status).Append("\n");
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
            return this.Equals(input as GameServer);
        }

        /// <summary>
        /// Returns true if GameServer instances are equal
        /// </summary>
        /// <param name="input">Instance of GameServer to be compared</param>
        /// <returns>Boolean</returns>
        public bool Equals(GameServer input)
        {
            if (input == null)
                return false;

            return
                (
                    this.ObjectMeta == input.ObjectMeta ||
                    (this.ObjectMeta != null &&
                     this.ObjectMeta.Equals(input.ObjectMeta))
                ) &&
                (
                    this.Spec == input.Spec ||
                    (this.Spec != null &&
                     this.Spec.Equals(input.Spec))
                ) &&
                (
                    this.Status == input.Status ||
                    (this.Status != null &&
                     this.Status.Equals(input.Status))
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
                if (this.ObjectMeta != null)
                    hashCode = hashCode * 59 + this.ObjectMeta.GetHashCode();
                if (this.Spec != null)
                    hashCode = hashCode * 59 + this.Spec.GetHashCode();
                if (this.Status != null)
                    hashCode = hashCode * 59 + this.Status.GetHashCode();
                return hashCode;
            }
        }
    }
}