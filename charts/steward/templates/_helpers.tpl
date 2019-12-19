{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "steward.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "steward.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "steward.labels" -}}
helm.sh/chart: {{ include "steward.chart" . }}
{{ include "steward.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service | quote }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "steward.selectorLabels" -}}
app.kubernetes.io/name: {{ include "steward.name" . | quote }}
{{- end -}}

{{/*
The component label for the run controller.
*/}}
{{- define "steward.run-controller.componentLabel" -}}
app.kubernetes.io/component: run-controller
{{- end -}}

{{/*
The component label for the tenant controller.
*/}}
{{- define "steward.tenant-controller.componentLabel" -}}
app.kubernetes.io/component: tenant-controller
{{- end -}}
