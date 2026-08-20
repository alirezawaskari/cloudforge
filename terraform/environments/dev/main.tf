# Dev environment root module. Targets whatever cluster your current
# kubeconfig context points at (kind, minikube, or a shared dev cluster) —
# no cloud provider or credentials required.

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
  namespace = "cloudforge-dev"
}

resource "kubernetes_namespace" "app" {
  metadata {
    name = local.namespace
  }
}

# Dev uses the cloudforge chart's own bundled PostgreSQL/Redis StatefulSets
# (deploy/helm/cloudforge/templates/{postgresql,redis}.yaml) rather than
# separately-provisioned Terraform-managed datastores — for a throwaway
# dev namespace, coupling their lifecycle to the app release is the
# simpler, cheaper choice. Staging and production provision datastores
# independently; see terraform/environments/staging and .../production.
module "app" {
  source = "../../modules/app"

  namespace   = local.namespace
  environment = "dev"
  image_tag   = var.image_tag

  depends_on = [kubernetes_namespace.app]
}
