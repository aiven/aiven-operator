data "terraform_remote_state" "infra" {
  backend = "local"
  config = {
    path = "../cluster/terraform.tfstate"
  }
}

# gets the cluster details from the cluster module's output
data "google_container_cluster" "cluster" {
  project  = data.terraform_remote_state.infra.outputs.project_id
  location = data.terraform_remote_state.infra.outputs.cluster_location
  name     = data.terraform_remote_state.infra.outputs.cluster_name
}

# gets the current gcloud authentication details
data "google_client_config" "default" {}
