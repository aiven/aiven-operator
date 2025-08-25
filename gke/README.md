# GKE Development Environment

Quick setup for developing the Aiven Operator on Google Kubernetes Engine with automated deployments.

## Prerequisites

### Required Tools
- [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) (gcloud)
- [Terraform](https://developer.hashicorp.com/terraform/downloads) ≥1.0
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Helm](https://helm.sh/docs/intro/install/) ≥3.0
- [Docker](https://docs.docker.com/get-docker/)
- [Task](https://taskfile.dev/installation/) (task runner)

### Required GCP Permissions

Your GCP user account needs these IAM roles:
```
roles/compute.admin                    # VPC, subnets, firewall rules
roles/container.admin                  # GKE clusters, node pools
roles/container.clusterAdmin           # Kubernetes RBAC permissions
roles/artifactregistry.admin           # Container image registry
roles/servicemanagement.admin          # Enable required APIs
roles/serviceusage.serviceUsageAdmin   # Manage API services
roles/iam.serviceAccountUser           # Use service accounts
```

### Authentication Setup
```bash
gcloud auth login
gcloud config set project YOUR_PROJECT_ID
gcloud auth configure-docker
```

## Quick Start

1. **Configure environment**:
   ```bash
   cd gke/
   task config:init
   # Edit .env with your PROJECT_ID, AIVEN_PROJECT, and AIVEN_TOKEN
   ```

2. **Deploy infrastructure**:
   ```bash
   task cluster:deploy    # Create GKE cluster + container registry
   task addons:deploy     # Build and deploy operator
   ```

3. **Deploy operator**:
   ```bash
   task addons:deploy     # Auto-rebuilds if code changed, pushes to GCR
   task logs              # View operator logs
   task status            # Check pod status
   ```

4. **Deploy test resources**: See [Test Resources Setup](test-resources/overlays/templates/README.md) for Aiven service templates.

## Environment Variables

Required in `.env`:
- `PROJECT_ID` - Your GCP project ID
- `AIVEN_PROJECT` - Your Aiven project name  
- `AIVEN_TOKEN` - Your Aiven API token

Optional configuration:
- `REGION` - GCP region (default: europe-west1)
- `KUBERNETES_VERSION` - GKE version (default: 1.32)
- `DEV_PREFIX` - Resource name prefix (auto-detected from username)
- `MY_PUBLIC_IP` - IP for cluster access (auto-detected)