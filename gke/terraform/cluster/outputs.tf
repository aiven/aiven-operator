output "cluster_name" {
  description = "The name of the GKE cluster."
  value       = google_container_cluster.aiven_operator_cluster.name
}

output "cluster_location" {
  description = "The location (region) of the GKE cluster."
  value       = google_container_cluster.aiven_operator_cluster.location
}

output "project_id" {
  description = "The GCP project ID where the cluster is deployed."
  value       = var.project_id
}

output "network_name" {
  description = "The name of the VPC network created for the cluster."
  value       = google_compute_network.vpc.name
}
