{{/*
Expand the name of the chart.
*/}}
{{- define "csi-nas.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "csi-nas.fullname" -}}
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
Create a default fully qualified controller name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "csi-nas.controller.fullname" -}}
{{- printf "%s-%s" (include "csi-nas.fullname" .) .Values.controller.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified node name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "csi-nas.node.fullname" -}}
{{- printf "%s-%s" (include "csi-nas.fullname" .) .Values.node.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "csi-nas.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "csi-nas.labels" -}}
helm.sh/chart: {{ include "csi-nas.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Controller labels
*/}}
{{- define "csi-nas.controller.labels" -}}
{{ include "csi-nas.controller.selectorLabels" . }}
{{ include "csi-nas.labels" . }}
{{- end }}

{{/*
Node labels
*/}}
{{- define "csi-nas.node.labels" -}}
{{ include "csi-nas.node.selectorLabels" . }}
{{ include "csi-nas.labels" . }}
{{- end }}

{{/*
Controller selector labels
*/}}
{{- define "csi-nas.controller.selectorLabels" -}}
app.kubernetes.io/name: {{ include "csi-nas.controller.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Node selector labels
*/}}
{{- define "csi-nas.node.selectorLabels" -}}
app.kubernetes.io/name: {{ include "csi-nas.node.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "csi-nas.serviceAccountName" -}}
{{- if .Values.rbac.serviceAccount.create }}
{{- default (include "csi-nas.fullname" .) .Values.rbac.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.rbac.serviceAccount.name }}
{{- end }}
{{- end }}
