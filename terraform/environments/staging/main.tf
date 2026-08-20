# Staging environment root module. Same shape as dev, sized up, with the
# observability stack included since staging is meant to mirror production
# behavior for pre-release validation.

terraform {
  required_version = ">= 1.7.0"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.31"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.14"
    }
  }
}

provider "kubernetes" {
  config_path    = var.kubeconfig_path
  config_context = var.kube_context
}

provider "helm" {
  kubernetes {
    config_path    = var.kubeconfig_path
    config_context = var.kube_context
  }
}

locals {
  namespace = "cloudforge-staging"
}

resource "kubernetes_namespace" "app" {
  metadata {
    name = local.namespace
  }
}

# Staging uses the cloudforge chart's own bundled PostgreSQL/Redis
# StatefulSets (same as dev) with production-sized storage set via
# values-staging.yaml, rather than separately-provisioned datastores —
# staging is disposable infrastructure meant to be torn down and rebuilt
# from a release tag, so coupling data-store lifecycle to the app release
# is intentional here. Production does the opposite; see
# terraform/environments/production.
module "app" {
  source = "../../modules/app"

  namespace   = local.namespace
  environment = "staging"
  image_tag   = var.image_tag

  values_overrides = {
    "postgresql.auth.password"  = var.database_password
    "secrets.data.DATABASE_URL" = "postgres://cloudforge:${var.database_password}@cloudforge-postgresql:5432/cloudforge?sslmode=disable"
    "ingress.hosts[0].host"     = var.hostname
    "ingress.tls[0].hosts[0]"   = var.hostname
  }

  depends_on = [kubernetes_namespace.app]
}

module "monitoring" {
  source = "../../modules/monitoring"

  count = var.enable_monitoring ? 1 : 0

  namespace = "observability"
}
