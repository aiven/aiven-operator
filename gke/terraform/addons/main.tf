resource "kubernetes_namespace" "cert_manager" {
  metadata {
    name = "cert-manager"
  }
}

# deploy cert-manager
resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  namespace        = kubernetes_namespace.cert_manager.metadata[0].name
  version          = "v1.18.2"
  create_namespace = false

  set = [
    {
      name  = "installCRDs"
      value = "true"
    },
    {
      name  = "global.leaderElection.namespace"
      value = kubernetes_namespace.cert_manager.metadata[0].name
    }
  ]

  depends_on = [
    kubernetes_namespace.cert_manager
  ]
}

# dedicated namespace for the Aiven Operator
resource "kubernetes_namespace" "aiven_operator" {
  metadata {
    name = var.operator_namespace
  }
}

# store the Aiven API token
resource "kubernetes_secret" "aiven_token" {
  metadata {
    name      = "aiven-operator-token"
    namespace = kubernetes_namespace.aiven_operator.metadata[0].name
  }
  data = {
    "token" = var.aiven_token
  }
  type = "Opaque"
}

# deploy the Aiven Operator CRDs
resource "helm_release" "aiven_operator_crds" {
  name      = "aiven-operator-crds"
  chart     = "../../../charts/aiven-operator-crds"
  namespace = kubernetes_namespace.aiven_operator.metadata[0].name

  depends_on = [
    kubernetes_namespace.aiven_operator
  ]
}

# deploy the Aiven Operator
resource "helm_release" "aiven_operator" {
  name      = "aiven-operator"
  chart     = "../../../charts/aiven-operator"
  namespace = kubernetes_namespace.aiven_operator.metadata[0].name

  set = [
    {
      name  = "defaultTokenSecret.name"
      value = kubernetes_secret.aiven_token.metadata[0].name
    },
    {
      name  = "defaultTokenSecret.key"
      value = "token"
    },
    {
      name  = "logging.level"
      value = var.operator_log_level
    },
    {
      name  = "webhooks.enabled"
      value = var.enable_operator_webhooks
    },
    {
      name  = "image.repository"
      value = data.terraform_remote_state.infra.outputs.repository_url
    },
    {
      name  = "image.tag"
      value = var.operator_image_tag
    },
    {
      name  = "image.pullPolicy"
      value = "Always"
    }
  ]

  depends_on = [
    kubernetes_namespace.aiven_operator,
    kubernetes_secret.aiven_token,
    helm_release.aiven_operator_crds,
    helm_release.cert_manager
  ]
}
