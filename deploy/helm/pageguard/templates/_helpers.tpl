{{- define "pageguard.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "pageguard.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s" (include "pageguard.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "pageguard.labels" -}}
app.kubernetes.io/name: {{ include "pageguard.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "pageguard.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pageguard.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "pageguard.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "pageguard.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "pageguard.databaseUrl" -}}
{{- if .Values.postgres.enabled -}}
{{- printf "postgres://%s:%s@%s:%v/%s?sslmode=%s" .Values.postgres.user .Values.postgres.password .Values.postgres.host .Values.postgres.port .Values.postgres.database .Values.postgres.sslMode -}}
{{- end -}}
{{- end -}}
