output "aiven_operator_helm_release_status" {
  description = "Status of the Aiven Operator Helm release."
  value       = helm_release.aiven_operator.status
}

output "aiven_operator_namespace" {
  description = "Namespace where the Aiven Operator was installed."
  value       = helm_release.aiven_operator.namespace
}
