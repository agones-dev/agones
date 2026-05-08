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
  /// StatusAddresses represents an address with a specific type.
  /// </summary>
  public class StatusAddresses : IEquatable<StatusAddresses>
  {
    /// <summary>
    /// Initializes a new instance of the <see cref="StatusAddresses" /> class.
    /// </summary>
    /// <param name="data">The data dictionary containing the address and type.</param>
    public StatusAddresses(IReadOnlyDictionary<string, object> data)
    {
      this.Address = (string)data["address"];
      this.Type = (string)data["type"];
    }

    public string Address { get; }
    public string Type { get; }

    /// <summary>
    /// Returns the string presentation of the object
    /// </summary>
    /// <returns>String presentation of the object</returns>
    public override string ToString()
    {
      var sb = new StringBuilder();
      sb.Append("class StatusAddresses {\n");
      sb.Append("  Address: ").Append(Address).Append("\n");
      sb.Append("  Type: ").Append(Type).Append("\n");
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
      return this.Equals(input as StatusAddresses);
    }

    /// <summary>
    /// Returns true if StatusAddresses instances are equal
    /// </summary>
    /// <param name="input">Instance of StatusAddresses to be compared</param>
    /// <returns>Boolean</returns>
    public bool Equals(StatusAddresses input)
    {
      if (input == null)
        return false;

      return
          (
              this.Address == input.Address ||
              (this.Address != null &&
               this.Address.Equals(input.Address))
          ) &&
          (
              this.Type == input.Type ||
              (this.Type != null &&
               this.Type.Equals(input.Type))
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
        if (this.Address != null)
          hashCode = hashCode * 59 + this.Address.GetHashCode();
        if (this.Type != null)
          hashCode = hashCode * 59 + this.Type.GetHashCode();
        return hashCode;
      }
    }
  }
}
