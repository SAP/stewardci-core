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
{{- define "steward.runController.componentLabel" -}}
app.kubernetes.io/component: run-controller
{{- end -}}


{{/*
The additional labels for the service monitors.
*/}}
{{- define "steward.serviceMonitors.extraLabels" -}}
{{- if .Values.metrics.serviceMonitors.extraLabels -}}
{{- toYaml .Values.metrics.serviceMonitors.extraLabels -}}
{{- end -}}
{{- end -}}

{{/*
The name of the pod security policy for the run controller.
*/}}
{{- define "steward.runController.podSecurityPolicyName" -}}
{{- if .Values.runController.podSecurityPolicyName -}}
{{- .Values.runController.podSecurityPolicyName -}}
{{- else -}}
{{- include "steward.controllers.podSecurityPolicyName.builtin" . -}}
{{- end -}}
{{- end -}}

{{/*
The name of the pod security policy for Steward controllers that is
created by this chart if the user doesn't provide an own PSP.
*/}}
{{- define "steward.controllers.podSecurityPolicyName.builtin" -}}
00-steward-controllers
{{- end -}}

{{/*
The name of the pod security policy for pipeline runs.
*/}}
{{- define "steward.pipelineRuns.podSecurityPolicyName" -}}
{{- if .Values.pipelineRuns.podSecurityPolicyName -}}
{{- .Values.pipelineRuns.podSecurityPolicyName -}}
{{- else -}}
{{- include "steward.pipelineRuns.podSecurityPolicyName.builtin" . -}}
{{- end -}}
{{- end -}}

{{/*
The name of the pod security policy for pipeline runs that is
created by this chart if the user doesn't provide an own PSP.
*/}}
{{- define "steward.pipelineRuns.podSecurityPolicyName.builtin" -}}
00-steward-run
{{- end -}}

{{/*
Resolves to a non-empty string if the Chart should generate a
pod security policy for Steward controllers, otherwise resolves
to the empty string.
*/}}
{{- define "steward.controllers.generatePodSecurityPolicy" -}}
{{- if not .Values.runController.podSecurityPolicyName -}}
true
{{- end -}}
{{- end -}}

{{/*
Resolves to a non-empty string if the Chart should generate a
pod security policy for pipeline runs, otherwise resolves
to the empty string.
*/}}
{{- define "steward.pipelineRuns.generatePodSecurityPolicy" -}}
{{- if not .Values.pipelineRuns.podSecurityPolicyName -}}
true
{{- end -}}
{{- end -}}
