variable "namespace" {
  type = string
}

variable "environment" {
  description = "One of dev, staging, production — selects the matching values-<env>.yaml overlay."
  type        = string

  validation {
    condition     = contains(["dev", "staging", "production"], var.environment)
    error_message = "environment must be one of: dev, staging, production."
  }
}

variable "image_tag" {
  type        = string
  description = "Image tag to deploy, e.g. a git SHA or semver produced by CI."
}

variable "values_overrides" {
  description = "Additional Helm --set overrides, e.g. { \"secrets.existingSecret\" = \"cloudforge-secrets\" }."
  type        = map(string)
  default     = {}
}
