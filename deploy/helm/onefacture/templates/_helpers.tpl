{{- define "onefacture.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "onefacture.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "onefacture.labels" -}}
app.kubernetes.io/name: {{ include "onefacture.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end -}}

{{- define "onefacture.selectorLabels" -}}
app.kubernetes.io/name: {{ include "onefacture.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
