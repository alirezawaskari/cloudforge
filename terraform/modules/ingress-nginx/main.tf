resource "helm_release" "ingress_nginx" {
  name             = "ingress-nginx"
  repository       = "https://kubernetes.github.io/ingress-nginx"
  chart            = "ingress-nginx"
  version          = var.chart_version
  namespace        = var.namespace
  create_namespace = true
  timeout          = 300

  values = [
    yamlencode({
      controller = {
        replicaCount = var.replica_count
        resources = {
          requests = { cpu = "100m", memory = "128Mi" }
          limits   = { cpu = "500m", memory = "256Mi" }
        }
        service = {
          type = var.service_type
        }
        metrics = {
          enabled = true
        }
        podDisruptionBudget = {
          enabled      = var.replica_count > 1
          minAvailable = 1
        }
      }
    })
  ]
}
