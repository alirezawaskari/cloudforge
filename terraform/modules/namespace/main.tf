resource "kubernetes_namespace" "this" {
  metadata {
    name   = var.name
    labels = var.labels
  }
}

resource "kubernetes_resource_quota" "this" {
  count = var.resource_quota != null ? 1 : 0

  metadata {
    name      = "${var.name}-quota"
    namespace = kubernetes_namespace.this.metadata[0].name
  }

  spec {
    hard = var.resource_quota
  }
}

resource "kubernetes_limit_range" "this" {
  count = var.default_container_limits != null ? 1 : 0

  metadata {
    name      = "${var.name}-limits"
    namespace = kubernetes_namespace.this.metadata[0].name
  }

  spec {
    limit {
      type            = "Container"
      default         = var.default_container_limits.default
      default_request = var.default_container_limits.default_request
    }
  }
}
