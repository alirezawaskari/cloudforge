variable "kubeconfig_path" {
  type    = string
  default = "~/.kube/config"
}

variable "kube_context" {
  type    = string
  default = "production-cluster"
}

variable "image_tag" {
  description = "cloudforge-api image tag to deploy — a released semver tag, not a branch or 'latest'."
  type        = string
}

variable "secret_name" {
  description = "Name of the pre-existing Kubernetes Secret holding DATABASE_URL/REDIS_ADDR/etc., populated by an external secret manager."
  type        = string
  default     = "cloudforge-production-secrets"
}

variable "hostname" {
  type    = string
  default = "cloudforge.example.com"
}

variable "ingress_service_type" {
  description = "Service type for the ingress-nginx controller. LoadBalancer on any conformant cloud provisions an external LB with no cloud-specific Terraform required."
  type        = string
  default     = "LoadBalancer"
}
