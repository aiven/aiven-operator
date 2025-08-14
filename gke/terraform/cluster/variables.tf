variable "project_id" {
  type        = string
  description = "The GCP project ID."
}

variable "region" {
  type        = string
  description = "The GCP region to deploy resources."
  default     = "europe-west1"
}

variable "cluster_name" {
  type        = string
  description = "The name of the GKE cluster and a prefix for other resources."
}

variable "kubernetes_version" {
  type        = string
  description = "The Kubernetes version for the GKE cluster."
  default     = "1.32"
}

variable "my_public_ip" {
  type        = string
  description = "Optional public IP override for master authorized networks. If empty, it will be auto-detected."
  default     = ""
}
