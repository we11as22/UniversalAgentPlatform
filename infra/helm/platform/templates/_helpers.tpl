{{- define "uap.commonLabels" -}}
app.kubernetes.io/part-of: universal-agent-platform
app.kubernetes.io/managed-by: Helm
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
{{- end -}}

{{- define "uap.workloadLabels" -}}
app.kubernetes.io/name: {{ .name }}
app.kubernetes.io/component: {{ .component | default .name }}
{{ include "uap.commonLabels" .root }}
{{- if .mesh }}
uap.mesh/enabled: "true"
{{- end }}
{{- end -}}

{{- define "uap.serviceAccountName" -}}
{{ .name }}
{{- end -}}

