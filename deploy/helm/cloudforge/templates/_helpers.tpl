{{/*
Expand the name of the chart.
*/}}
{{- define "cloudforge.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Full release name.
*/}}
{{- define "cloudforge.fullname" -}}
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

{{- define "cloudforge.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels, chart/version/managed-by metadata only — no component, so
it's safe to share across the api/postgresql/redis manifests.
*/}}
{{- define "cloudforge.labels" -}}
helm.sh/chart: {{ include "cloudforge.chart" . }}
{{ include "cloudforge.baseSelectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Base selector labels shared by every component (name + instance only).
Never use this alone for a Service/PDB/NetworkPolicy/HPA selector — it
matches the API, PostgreSQL, and Redis pods all at once. Use
cloudforge.selectorLabels (API) or add an explicit
app.kubernetes.io/component label (datastores) instead.
*/}}
{{- define "cloudforge.baseSelectorLabels" -}}
app.kubernetes.io/name: {{ include "cloudforge.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Selector labels for the API deployment specifically.
*/}}
{{- define "cloudforge.selectorLabels" -}}
{{ include "cloudforge.baseSelectorLabels" . }}
app.kubernetes.io/component: api
{{- end }}

{{/*
Service account name.
*/}}
{{- define "cloudforge.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "cloudforge.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Name of the secret holding application configuration.
*/}}
{{- define "cloudforge.secretName" -}}
{{- if .Values.secrets.existingSecret }}
{{- .Values.secrets.existingSecret }}
{{- else }}
{{- include "cloudforge.fullname" . }}
{{- end }}
{{- end }}

{{/*
Image reference, defaulting the tag to the chart's appVersion.
*/}}
{{- define "cloudforge.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}
