{{/*
Expands to a complete hook spec that updates a custom resource definition
from the `crds` directory.

Expects dot to be a list with two entries:

1. the original dot providing .Values and so on
2. the name of the crd manifest file (without the .yaml extension)
*/}}
{{- define "steward.hooks.crd-update" }}
{{- $crdName := first ( slice . 1 ) }}
{{- with first . -}}

apiVersion: v1
kind: ServiceAccount
metadata:
  name: helm-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" . }}
  namespace: {{ .Release.Namespace | quote }}
  labels:
    {{- include "steward.labels" . | nindent 4 }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade,pre-rollback
    "helm.sh/hook-weight": "-5"
    "helm.sh/hook-delete-policy": hook-succeeded,hook-failed,before-hook-creation
---

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: helm-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" . }}
  labels:
    {{- include "steward.labels" . | nindent 4 }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade,pre-rollback
    "helm.sh/hook-weight": "-5"
    "helm.sh/hook-delete-policy": hook-succeeded,hook-failed,before-hook-creation
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: helm-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" . }}
  namespace: {{ .Release.Namespace | quote }}
---

apiVersion: batch/v1
kind: Job
metadata:
  name: helm-hook-crd-update-{{ $crdName }}-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" . }}
  namespace: {{ .Release.Namespace | quote }}
  labels:
    {{- include "steward.labels" . | nindent 4 }}
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade,pre-rollback
    "helm.sh/hook-weight": "0"
    "helm.sh/hook-delete-policy": hook-succeeded,before-hook-creation
spec:
  activeDeadlineSeconds: 300 # 5 min
  ttlSecondsAfterFinished: 86400 # 24 hours
  backoffLimit: 1
  template:
    metadata:
      name: dummy
      labels:
        {{- include "steward.labels" . | nindent 8 }}
    spec:
      serviceAccountName: helm-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" . }}
      restartPolicy: Never
      containers:
      - name: kubectl
        {{- with .Values.hooks.images.kubectl }}
        image: {{ printf "%s:%s" .repository .tag | quote }}
        imagePullPolicy: {{ .pullPolicy | quote }}
        {{- end }}
        env:
        - name: CRD_SPEC
          value: {{ .Files.Get ( printf "crds/%s.yaml" $crdName ) | quote }}
        command:
        - "bin/sh"
        - "-c"
        - >
          echo "$CRD_SPEC" | kubectl apply -f -

{{- end -}}
{{- end -}}


{{- define "steward.hooks-helpers.adler32sumOfReleaseId" }}
{{- list .Chart.Name .Release.Name .Release.Namespace | join "\n" | adler32sum -}}
{{- end -}}
