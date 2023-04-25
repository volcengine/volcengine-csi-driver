{{/*
Expand the name of the chart.
*/}}
{{- define "csi-tos.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "csi-tos.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified node name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "csi-tos.node.fullname" -}}
{{- printf "%s-%s" (include "csi-tos.fullname" .) .Values.node.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified launcher name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "csi-tos.launcher.fullname" -}}
{{- printf "%s-%s" (include "csi-tos.fullname" .) .Values.launcher.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "csi-tos.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "csi-tos.labels" -}}
helm.sh/chart: {{ include "csi-tos.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Node labels
*/}}
{{- define "csi-tos.node.labels" -}}
{{ include "csi-tos.labels" . }}
{{ include "csi-tos.node.selectorLabels" . }}
{{- end }}

{{/*
Node selector labels
*/}}
{{- define "csi-tos.node.selectorLabels" -}}
app.kubernetes.io/name: {{ include "csi-tos.node.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Launcher labels
*/}}
{{- define "csi-tos.launcher.labels" -}}
{{ include "csi-tos.labels" . }}
{{ include "csi-tos.launcher.selectorLabels" . }}
{{- end }}

{{/*
Launcher selector labels
*/}}
{{- define "csi-tos.launcher.selectorLabels" -}}
app.kubernetes.io/name: {{ include "csi-tos.launcher.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}