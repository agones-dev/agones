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
    /// SpecHealth
    /// </summary>
    public class SpecHealth : IEquatable<SpecHealth>
    {
        /// <summary>
        /// Initializes a new instance of the <see cref="SpecHealth" /> class.
        /// </summary>
        public SpecHealth(IReadOnlyDictionary<string, object> data)
        {
            this.Disabled = data.TryGetValue("disabled", out var disabled) && (bool)disabled;
            if (!this.Disabled)
            {
                this.PeriodSeconds = (Int64)data["period_seconds"];
                this.FailureThreshold = (Int64)data["failure_threshold"];
                this.InitialDelaySeconds = (Int64)data["initial_delay_seconds"];
            }
        }

        public bool Disabled { get; }
        public Int64 PeriodSeconds { get; }
        public Int64 FailureThreshold { get; }
        public Int64 InitialDelaySeconds { get; }

        /// <summary>
        /// Returns the string presentation of the object
        /// </summary>
        /// <returns>String presentation of the object</returns>
        public override string ToString()
        {
            var sb = new StringBuilder();
            sb.Append("class SpecHealth {\n");
            sb.Append("  Disabled: ").Append(Disabled).Append("\n");
            sb.Append("  PeriodSeconds: ").Append(PeriodSeconds).Append("\n");
            sb.Append("  FailureThreshold: ").Append(FailureThreshold).Append("\n");
            sb.Append("  InitialDelaySeconds: ").Append(InitialDelaySeconds).Append("\n");
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
            return this.Equals(input as SpecHealth);
        }

        /// <summary>
        /// Returns true if SpecHealth instances are equal
        /// </summary>
        /// <param name="input">Instance of SpecHealth to be compared</param>
        /// <returns>Boolean</returns>
        public bool Equals(SpecHealth input)
        {
            if (input == null)
                return false;

            return
                (
                    this.Disabled == input.Disabled ||
                    (this.Disabled.Equals(input.Disabled))
                ) &&
                (
                    this.PeriodSeconds == input.PeriodSeconds ||
                    (this.PeriodSeconds.Equals(input.PeriodSeconds))
                ) &&
                (
                    this.FailureThreshold == input.FailureThreshold ||
                    (this.FailureThreshold.Equals(input.FailureThreshold))
                ) &&
                (
                    this.InitialDelaySeconds == input.InitialDelaySeconds ||
                    (this.InitialDelaySeconds.Equals(input.InitialDelaySeconds))
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
                hashCode = hashCode * 59 + this.Disabled.GetHashCode();
                hashCode = hashCode * 59 + this.PeriodSeconds.GetHashCode();
                hashCode = hashCode * 59 + this.FailureThreshold.GetHashCode();
                hashCode = hashCode * 59 + this.InitialDelaySeconds.GetHashCode();
                return hashCode;
            }
        }
    }
}