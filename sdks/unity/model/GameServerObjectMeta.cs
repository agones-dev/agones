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
    /// GameServerObjectMeta
    /// </summary>
    public class GameServerObjectMeta : IEquatable<GameServerObjectMeta>
    {
        /// <summary>
        /// Initializes a new instance of the <see cref="GameServerObjectMeta" /> class.
        /// </summary>
        public GameServerObjectMeta(IReadOnlyDictionary<string, object> data)
        {
            this.Name = (string) data["name"];
            this.Namespace = (string) data["namespace"];
            this.Uid = (string) data["uid"];
            this.ResourceVersion = (string) data["resource_version"];
            this.Generation = Int32.Parse((string) data["generation"]);
            this.CreationTimestamp =
                DateTimeOffset.FromUnixTimeSeconds(long.Parse((string) data["creation_timestamp"])).DateTime;

            if (data.TryGetValue("deletion_timestamp", out var timestamp))
            {
                this.DeletionTimestamp =
                    DateTimeOffset.FromUnixTimeSeconds(long.Parse((string) timestamp)).DateTime;
            }

            if (data.TryGetValue("annotations", out var annotations))
            {
                this.Annotations = new Dictionary<string, string>();
                var values = (Dictionary<string, object>) annotations;
                foreach (var item in values)
                {
                    this.Annotations.Add(item.Key, item.Value.ToString());
                }
            }

            if (data.TryGetValue("labels", out var labels))
            {
                this.Labels = new Dictionary<string, string>();
                var values = (Dictionary<string, object>) labels;
                foreach (var item in values)
                {
                    this.Labels.Add(item.Key, item.Value.ToString());
                }
            }
        }

        public string Name { get; }
        public string Namespace { get; }
        public string Uid { get; }
        public string ResourceVersion { get; }
        public int Generation { get; }
        public DateTime CreationTimestamp { get; }
        public DateTime? DeletionTimestamp { get; }
        public Dictionary<string, string> Annotations { get; }
        public Dictionary<string, string> Labels { get; }

        /// <summary>
        /// Returns the string presentation of the object
        /// </summary>
        /// <returns>String presentation of the object</returns>
        public override string ToString()
        {
            var sb = new StringBuilder();
            sb.Append("class GameServerObjectMeta {\n");
            sb.Append("  Name: ").Append(Name).Append("\n");
            sb.Append("  Namespace: ").Append(Namespace).Append("\n");
            sb.Append("  Uid: ").Append(Uid).Append("\n");
            sb.Append("  ResourceVersion: ").Append(ResourceVersion).Append("\n");
            sb.Append("  Generation: ").Append(Generation).Append("\n");
            sb.Append("  CreationTimestamp: ").Append(CreationTimestamp).Append("\n");
            sb.Append("  DeletionTimestamp: ").Append(DeletionTimestamp).Append("\n");
            sb.Append("  Annotations: ").Append(string.Join(";", Annotations)).Append("\n");
            sb.Append("  Labels: ").Append(string.Join(";", Labels)).Append("\n");
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
            return this.Equals(input as GameServerObjectMeta);
        }

        /// <summary>
        /// Returns true if GameServerObjectMeta instances are equal
        /// </summary>
        /// <param name="input">Instance of GameServerObjectMeta to be compared</param>
        /// <returns>Boolean</returns>
        public bool Equals(GameServerObjectMeta input)
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
                    this.Namespace == input.Namespace ||
                    (this.Namespace != null &&
                     this.Namespace.Equals(input.Namespace))
                ) &&
                (
                    this.Uid == input.Uid ||
                    (this.Uid != null &&
                     this.Uid.Equals(input.Uid))
                ) &&
                (
                    this.ResourceVersion == input.ResourceVersion ||
                    (this.ResourceVersion != null &&
                     this.ResourceVersion.Equals(input.ResourceVersion))
                ) &&
                (
                    this.Generation == input.Generation ||
                    (this.Generation.Equals(input.Generation))
                ) &&
                (
                    this.CreationTimestamp == input.CreationTimestamp ||
                    (this.CreationTimestamp.Equals(input.CreationTimestamp))
                ) &&
                (
                    this.DeletionTimestamp == input.DeletionTimestamp ||
                    (this.DeletionTimestamp != null &&
                     this.DeletionTimestamp.Equals(input.DeletionTimestamp))
                ) &&
                (
                    this.Annotations == input.Annotations ||
                    this.Annotations != null &&
                    this.Annotations.SequenceEqual(input.Annotations)
                ) &&
                (
                    this.Labels == input.Labels ||
                    this.Labels != null &&
                    this.Labels.SequenceEqual(input.Labels)
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
                if (this.Namespace != null)
                    hashCode = hashCode * 59 + this.Namespace.GetHashCode();
                if (this.Uid != null)
                    hashCode = hashCode * 59 + this.Uid.GetHashCode();
                if (this.ResourceVersion != null)
                    hashCode = hashCode * 59 + this.ResourceVersion.GetHashCode();
                hashCode = hashCode * 59 + this.Generation.GetHashCode();
                hashCode = hashCode * 59 + this.CreationTimestamp.GetHashCode();
                if (this.DeletionTimestamp != null)
                    hashCode = hashCode * 59 + this.DeletionTimestamp.GetHashCode();
                if (this.Annotations != null)
                    hashCode = hashCode * 59 + this.Annotations.GetHashCode();
                if (this.Labels != null)
                    hashCode = hashCode * 59 + this.Labels.GetHashCode();
                return hashCode;
            }
        }
    }
}