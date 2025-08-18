locals {
  # 1. If my_public_ip is explicitly "0.0.0.0/0", allow all IPs.
  # 2. If my_public_ip is set to any other value, use that value.
  # 3. If my_public_ip is an empty string, use the auto-detected IP from the http data source.
  master_networks = var.my_public_ip == "0.0.0.0/0" ? [
    {
      cidr_block   = "0.0.0.0/0"
      display_name = "All IPs (development mode)"
    }
    ] : (
    var.my_public_ip != "" ? [
      {
        cidr_block   = substr(var.my_public_ip, -3, 3) == "/32" ? var.my_public_ip : "${var.my_public_ip}/32"
        display_name = "Developer IP (from .env)"
      }
      ] : [
      {
        cidr_block   = "${chomp(data.http.my_ip.response_body)}/32"
        display_name = "Developer IP (auto-detected)"
      }
    ]
  )
}

resource "google_compute_network" "vpc" {
  name                    = "${var.cluster_name}-vpc"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "subnet" {
  name          = "${var.cluster_name}-subnet"
  ip_cidr_range = "10.10.0.0/24"
  network       = google_compute_network.vpc.id

  secondary_ip_range {
    range_name    = "pods"
    ip_cidr_range = "10.48.0.0/14"
  }

  secondary_ip_range {
    range_name    = "services"
    ip_cidr_range = "10.52.0.0/20"
  }
}

resource "google_compute_router" "router" {
  name    = "${var.cluster_name}-router"
  network = google_compute_network.vpc.id
}

resource "google_compute_router_nat" "nat" {
  name                               = "${var.cluster_name}-nat"
  router                             = google_compute_router.router.name
  region                             = google_compute_router.router.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

resource "google_container_cluster" "aiven_operator_cluster" {
  name     = var.cluster_name
  location = var.region

  network    = google_compute_network.vpc.name
  subnetwork = google_compute_subnetwork.subnet.name

  # see: https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-overview
  enable_autopilot    = true
  min_master_version  = var.kubernetes_version
  deletion_protection = false

  ip_allocation_policy {
    cluster_secondary_range_name  = google_compute_subnetwork.subnet.secondary_ip_range[0].range_name
    services_secondary_range_name = google_compute_subnetwork.subnet.secondary_ip_range[1].range_name
  }

  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = false
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }

  master_authorized_networks_config {
    dynamic "cidr_blocks" {
      for_each = local.master_networks
      content {
        cidr_block   = cidr_blocks.value.cidr_block
        display_name = cidr_blocks.value.display_name
      }
    }
  }


  master_auth {
    client_certificate_config {
      issue_client_certificate = false
    }
  }
}

resource "google_compute_firewall" "allow_master_to_nodes" {
  name      = "${var.cluster_name}-allow-master"
  network   = google_compute_network.vpc.name
  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["10250", "443"] # Kubelet and webhook ports
  }

  source_ranges = [google_container_cluster.aiven_operator_cluster.private_cluster_config[0].master_ipv4_cidr_block]
}

# Enable Artifact Registry API
resource "google_project_service" "artifactregistry" {
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

# Create the repository for operator images
resource "google_artifact_registry_repository" "image_repo" {
  location      = var.region
  repository_id = var.image_repo_name
  description   = "Docker repository for the Aiven Operator"
  format        = "DOCKER"

  depends_on = [google_project_service.artifactregistry]
}

# Enable Container Registry API
resource "google_project_service" "containerregistry" {
  service            = "containerregistry.googleapis.com"
  disable_on_destroy = false
}
