{{/*
Expand the name of the chart.
*/}}
{{- define "csi-ebs.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "csi-ebs.fullname" -}}
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
{{- define "csi-ebs.controller.fullname" -}}
{{- printf "%s-%s" (include "csi-ebs.fullname" .) .Values.controller.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified node name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "csi-ebs.node.fullname" -}}
{{- printf "%s-%s" (include "csi-ebs.fullname" .) .Values.node.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "csi-ebs.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Base labels
*/}}
{{- define "csi-ebs.labels" -}}
helm.sh/chart: {{ include "csi-ebs.chart" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "csi-ebs.common.labels" -}}
{{ include "csi-ebs.common.selectorLabels" . }}
{{ include "csi-ebs.labels" . }}
{{- end }}

{{/*
Controller labels
*/}}
{{- define "csi-ebs.controller.labels" -}}
{{ include "csi-ebs.controller.selectorLabels" . }}
{{ include "csi-ebs.labels" . }}
{{- end }}

{{/*
Node labels
*/}}
{{- define "csi-ebs.node.labels" -}}
{{ include "csi-ebs.node.selectorLabels" . }}
{{ include "csi-ebs.labels" . }}
{{- end }}

{{/*
Common selector labels
*/}}
{{- define "csi-ebs.common.selectorLabels" -}}
app.kubernetes.io/name: {{ include "csi-ebs.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Controller selector labels
*/}}
{{- define "csi-ebs.controller.selectorLabels" -}}
app.kubernetes.io/name: {{ include "csi-ebs.controller.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Node selector labels
*/}}
{{- define "csi-ebs.node.selectorLabels" -}}
app.kubernetes.io/name: {{ include "csi-ebs.node.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "csi-ebs.serviceAccountName" -}}
{{- if .Values.rbac.serviceAccount.create }}
{{- default (include "csi-ebs.name" .) .Values.rbac.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.rbac.serviceAccount.name }}
{{- end }}
{{- end }}
