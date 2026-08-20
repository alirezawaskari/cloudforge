variable "namespace" {
  description = "Namespace for the observability stack."
  type        = string
  default     = "observability"
}

variable "prometheus_chart_version" {
  type    = string
  default = "62.7.0"
}

variable "loki_chart_version" {
  type    = string
  default = "2.10.2"
}

variable "otel_chart_version" {
  type    = string
  default = "0.108.0"
}

variable "enable_logging" {
  description = "Install Loki + Promtail for log aggregation."
  type        = bool
  default     = true
}

variable "enable_tracing" {
  description = "Install the OpenTelemetry Collector for trace ingestion."
  type        = bool
  default     = true
}
