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
  name: helm-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" ( list . $crdName ) }}
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
  name: helm-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" ( list . $crdName ) }}
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
  name: helm-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" ( list . $crdName ) }}
  namespace: {{ .Release.Namespace | quote }}
---

apiVersion: batch/v1
kind: Job
metadata:
  name: helm-hook-crd-update-{{ $crdName }}-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" ( list . $crdName ) }}
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
      serviceAccountName: helm-{{ include "steward.hooks-helpers.adler32sumOfReleaseId" ( list . $crdName ) }}
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      restartPolicy: Never
      securityContext:
        {{- with .Values.hooks.crdUpdate.podSecurityContext }}
        {{- toYaml . | nindent 8 }}
        {{- else }}
        # chart default
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
        runAsNonRoot: true
        {{- end }}
      containers:
      - name: kubectl
        securityContext:
          {{- with .Values.hooks.crdUpdate.securityContext }}
          {{- toYaml . | nindent 10 }}
          {{- else }}
          # chart default
          privileged: false
          seccompProfile:
            type: RuntimeDefault
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
          {{- end }}
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
          if echo "$CRD_SPEC" | kubectl get -f - ; then
            echo "$CRD_SPEC" | kubectl replace -f -
          else
            echo "$CRD_SPEC" | kubectl create -f -
          fi
        resources:
          {{- with .Values.hooks.crdUpdate.resources }}
          {{- toYaml . | nindent 10 }}
          {{- else }}
          # chart default
          {{- end }}
      nodeSelector:
        {{- with .Values.hooks.crdUpdate.nodeSelector }}
        {{- toYaml . | nindent 8 }}
        {{- else }}
        # chart default
        {{- end }}
      affinity:
        {{- with .Values.hooks.crdUpdate.affinity }}
        {{- toYaml . | nindent 8 }}
        {{- else }}
        # chart default
        {{- end }}
      tolerations:
        {{- with .Values.hooks.crdUpdate.tolerations }}
        {{- toYaml . | nindent 8 }}
        {{- else }}
        # chart default
        {{- end }}
{{- end -}}
{{- end -}}


{{- define "steward.hooks-helpers.adler32sumOfReleaseId" }}
{{- $crdName := first ( slice . 1 ) }}
{{- with first . -}}
{{- list .Chart.Name .Release.Name .Release.Namespace $crdName | join "\n" | adler32sum -}}
{{- end -}}
{{- end -}}
