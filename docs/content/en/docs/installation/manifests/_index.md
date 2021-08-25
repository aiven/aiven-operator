---
title: "Installing from manifests (recommended)"
linkTitle: "Installing from manifests (recommended)"
weight: 15 
---

The Aiven Kubernetes Operator can be installed by applying the manifests present in the GitHub repository.

1. Install the `cert-manager` operator.
> cert-manager is used to manage the Operator [webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) TLS certificates.
```bash
$ kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.5.3/cert-manager.yaml
```

2. Install the Aiven Kubernetes Operator:
```bash
$ kubectl apply -f https://raw.githubusercontent.com/aiven/aiven-kubernetes-operator/main/config/deployment/v0.1.0.yaml
```

You've now installed the Aiven Kubernetes Operator. [Verify your installation](./verifying).
