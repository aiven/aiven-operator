variable "aiven_token" {
  type        = string
  description = "The Aiven API token for the operator to use."
  sensitive   = true
}

variable "operator_namespace" {
  type        = string
  description = "The Kubernetes namespace to install the Aiven Operator into."
  default     = "aiven-operator-namespace"
}

variable "enable_operator_webhooks" {
  type        = bool
  description = "If true, enables the operator's admission webhooks."
  default     = true
}

variable "operator_log_level" {
  type        = string
  description = "Log level for the Aiven Operator (e.g., info, debug)."
  default     = "info"
}

variable "operator_image_tag" {
  type        = string
  description = "Docker image tag for the Aiven Operator (e.g., git commit hash)."
  default     = "latest"
}
