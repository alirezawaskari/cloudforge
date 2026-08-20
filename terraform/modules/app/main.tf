# Installs the cloudforge Helm chart itself. Kept as a thin wrapper so
# environments/{dev,staging,production} stay declarative and don't need to
# know chart internals. Assumes var.namespace already exists (created once
# by the calling environment root module, shared with the datastore modules).

resource "helm_release" "cloudforge" {
  name      = "cloudforge"
  chart     = "${path.module}/../../../deploy/helm/cloudforge"
  namespace = var.namespace
  timeout   = 300
  wait      = true

  values = [
    file("${path.module}/../../../deploy/helm/cloudforge/values-${var.environment}.yaml")
  ]

  set {
    name  = "image.tag"
    value = var.image_tag
  }

  dynamic "set" {
    for_each = var.values_overrides
    content {
      name  = set.key
      value = set.value
    }
  }
}
