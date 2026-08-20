# Production environment root module.
#
# Unlike dev/staging, this does NOT provision PostgreSQL/Redis in-cluster —
# production data stores should be managed services (RDS/Cloud SQL,
# ElastiCache/MemoryStore, etc.) provisioned by your cloud-specific
# Terraform (out of scope here — see terraform/README.md) or by the
# platform team. This module only wires the application release to
# externally-supplied connection details and installs cluster-shared
# infrastructure: ingress controller and the observability stack.
#
# No cloud provider blocks or credentials appear anywhere in this repo —
# only the kubernetes/helm providers, authenticated via kubeconfig, which
# works identically against EKS, GKE, AKS, or any CNCF-conformant cluster.

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
  namespace = "cloudforge-production"
}

resource "kubernetes_namespace" "app" {
  metadata {
    name = local.namespace
    labels = {
      "environment" = "production"
    }
  }
}

module "ingress_nginx" {
  source = "../../modules/ingress-nginx"

  replica_count = 3
  service_type  = var.ingress_service_type
}

module "monitoring" {
  source = "../../modules/monitoring"

  namespace = "observability"
}

# Application secrets (DATABASE_URL, REDIS_ADDR, etc.) are expected to
# already exist as a Kubernetes Secret named var.secret_name, populated by
# an External Secrets Operator / Vault / cloud secret manager — see
# docs/SECRETS.md. Terraform intentionally does not create or read secret
# values here.
module "app" {
  source = "../../modules/app"

  namespace   = local.namespace
  environment = "production"
  image_tag   = var.image_tag

  values_overrides = {
    "postgresql.enabled"      = "false"
    "redis.enabled"           = "false"
    "secrets.existingSecret"  = var.secret_name
    "ingress.hosts[0].host"   = var.hostname
    "ingress.tls[0].hosts[0]" = var.hostname
  }

  depends_on = [kubernetes_namespace.app, module.ingress_nginx]
}
