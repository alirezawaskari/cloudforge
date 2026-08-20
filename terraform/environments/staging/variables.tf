variable "kubeconfig_path" {
  type    = string
  default = "~/.kube/config"
}

variable "kube_context" {
  type    = string
  default = "staging-cluster"
}

variable "image_tag" {
  description = "cloudforge-api image tag to deploy — typically a git SHA from CI."
  type        = string
}

variable "database_password" {
  type      = string
  sensitive = true
}

variable "hostname" {
  type    = string
  default = "staging.cloudforge.example.com"
}

variable "enable_monitoring" {
  type    = bool
  default = true
}
