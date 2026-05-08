{*
 Copyright Contributors to Agones a Series of LF Projects, LLC.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*}

{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "agones.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "agones.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "agones.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Creates a YAML object representing known feature gates. Can then be used as:
  {{- $featureGates := include "agones.featureGates" . | fromYaml }}
then you can check a feature gate with e.g. $featureGates.Example

Implemented by intentionally duplicating YAML - later keys take precedence.
So we start with defaultfeaturegates.yaml and then splat intentionally set
feature gates. In the process, we validate that the feature gate is known.
*/}}
{{- define "agones.featureGates" -}}
{{- .Files.Get "defaultfeaturegates.yaml" -}}
{{- if $.Values.agones.featureGates }}
{{- $gates := .Files.Get "defaultfeaturegates.yaml" | fromYaml }}
{{- range splitList "&" $.Values.agones.featureGates }}
{{- $f := splitn "=" 2 . -}}
{{- if hasKey $gates $f._0 }}
{{$f._0}}: {{$f._1}}
{{- else -}}
{{- printf "Unknown feature gate %q" $f._0 | fail -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}