variable "namespace" {
  description = "Namespace to install ingress-nginx into."
  type        = string
  default     = "ingress-nginx"
}

variable "chart_version" {
  description = "ingress-nginx chart version."
  type        = string
  default     = "4.11.3"
}

variable "replica_count" {
  description = "Number of controller replicas."
  type        = number
  default     = 2
}

variable "service_type" {
  description = "Kubernetes Service type for the controller (LoadBalancer on real clouds, NodePort for kind)."
  type        = string
  default     = "LoadBalancer"
}
