{{- define "flowengine.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- define "flowengine.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "flowengine.name" .) | trunc 63 | trimSuffix "-" }}
{{- end }}
