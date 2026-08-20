output "namespace" {
  value = kubernetes_namespace.observability.metadata[0].name
}
