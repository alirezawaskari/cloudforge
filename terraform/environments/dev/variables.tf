variable "kubeconfig_path" {
  description = "Path to the kubeconfig file for the target cluster."
  type        = string
  default     = "~/.kube/config"
}

variable "kube_context" {
  description = "kubeconfig context to use, e.g. kind-cloudforge."
  type        = string
  default     = "kind-cloudforge"
}

variable "image_tag" {
  description = "cloudforge-api image tag to deploy."
  type        = string
  default     = "local"
}
