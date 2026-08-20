variable "name" {
  description = "Namespace name."
  type        = string
}

variable "labels" {
  description = "Labels applied to the namespace."
  type        = map(string)
  default     = {}
}

variable "resource_quota" {
  description = "Optional ResourceQuota hard limits, e.g. { \"requests.cpu\" = \"4\", \"requests.memory\" = \"8Gi\" }."
  type        = map(string)
  default     = null
}

variable "default_container_limits" {
  description = "Optional default container resource limits/requests for the namespace's LimitRange."
  type = object({
    default         = map(string)
    default_request = map(string)
  })
  default = null
}
