resource "kubernetes_namespace" "observability" {
  metadata {
    name = var.namespace
  }
}

resource "helm_release" "kube_prometheus_stack" {
  name       = "prometheus"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  version    = var.prometheus_chart_version
  namespace  = kubernetes_namespace.observability.metadata[0].name
  timeout    = 600

  values = [
    file("${path.module}/../../../observability/prometheus/prometheus-values.yaml")
  ]
}

resource "helm_release" "loki_stack" {
  count = var.enable_logging ? 1 : 0

  name       = "loki"
  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki-stack"
  version    = var.loki_chart_version
  namespace  = kubernetes_namespace.observability.metadata[0].name
  timeout    = 300

  values = [
    file("${path.module}/../../../observability/loki/loki-values.yaml")
  ]

  depends_on = [helm_release.kube_prometheus_stack]
}

resource "helm_release" "otel_collector" {
  count = var.enable_tracing ? 1 : 0

  name       = "otel-collector"
  repository = "https://open-telemetry.github.io/opentelemetry-helm-charts"
  chart      = "opentelemetry-collector"
  version    = var.otel_chart_version
  namespace  = kubernetes_namespace.observability.metadata[0].name
  timeout    = 300

  values = [
    file("${path.module}/../../../observability/otel/otel-collector-values.yaml"),
    yamlencode({
      config = yamldecode(file("${path.module}/../../../observability/otel/otel-collector-config.yaml"))
    })
  ]
}
