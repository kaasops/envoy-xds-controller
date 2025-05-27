{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "chart.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "chart.fullname" -}}
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
{{- define "chart.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "chart.labels" -}}
helm.sh/chart: {{ include "chart.chart" . }}
{{ include "chart.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "chart.labels-ui" -}}
helm.sh/chart: {{ include "chart.chart" . }}
{{ include "chart.selectorLabels-ui" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "chart.selectorLabels" -}}
app.kubernetes.io/name: {{ include "chart.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "chart.selectorLabels-ui" -}}
app.kubernetes.io/name: {{ include "chart.name" . }}-ui
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "chart.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "chart.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
 Generate volumeMounts
 */}}
 {{- define "chart.volumeMounts" -}}
 {{- $mounts := list -}}
 {{- if .Values.webhook.enabled -}}
 {{- $mounts = append $mounts (dict "name" "cert" "mountPath" "/tmp/k8s-webhook-server/serving-certs" "readOnly" true) -}}
 {{- end -}}
 {{- if .Values.extraVolumeMounts -}}
 {{- $mounts = concat $mounts .Values.extraVolumeMounts -}}
 {{- end -}}
 {{- if .Values.auth.enabled -}}
 {{- $modelVolumeMount := dict "name" "auth" "mountPath" "/etc/exc/access-control" -}}
 {{- $mounts = append $mounts $modelVolumeMount -}}
 {{- end -}}
 {{- if .Values.config -}}
 {{- $configVolumeMount := dict "name" "config" "mountPath" "/etc/exc" -}}
 {{- $mounts = append $mounts $configVolumeMount -}}
 {{- end -}}
 {{- if $mounts -}}
 {{- toYaml $mounts -}}
 {{- end -}}
 {{- end -}}

 {{/*
 Generate volumes
 */}}
 {{- define "chart.volumes" -}}
 {{- $volumes := list -}}
 {{- if .Values.webhook.enabled -}}
 {{- $certVolume := dict "name" "cert"
     "secret" (dict
         "secretName" (.Values.webhook.tls.name | required "webhook.tls.name is required")
         "defaultMode" 420
     ) -}}
 {{- $volumes = append $volumes $certVolume -}}
 {{- end -}}
 {{- if .Values.extraVolumes -}}
 {{- $volumes = concat $volumes .Values.extraVolumes -}}
 {{- end -}}
 {{- if .Values.auth.enabled -}}
 {{- $modelVolume := dict "name" "auth" "configMap" (dict "name" "access-control-model" ) -}}
 {{- $volumes = append $volumes $modelVolume -}}
 {{- end -}}
 {{- if .Values.config -}}
 {{- $configVolume := dict "name" "config" "configMap" (dict "name" "config" ) -}}
 {{- $volumes = append $volumes $configVolume -}}
 {{- end -}}
 {{- if $volumes -}}
 {{- toYaml $volumes -}}
 {{- end -}}
 {{- end -}}