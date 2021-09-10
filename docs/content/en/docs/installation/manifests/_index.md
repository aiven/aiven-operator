---
title: "Installing from manifests"
linkTitle: "Installing from manifests"
weight: 15
---

The Aiven Operator for Kubernetes can be installed by applying the manifests present in the GitHub repository.

1. Install the `cert-manager` operator.

> [cert-manager](https://cert-manager.io/) is used to manage the Operator [webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) TLS certificates.

```bash
$ kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

2. Install the operator:

```bash
$ kubectl apply -f https://github.com/aiven/aiven-operator/releases/latest/download/deployment.yaml
```

You've now installed the Aiven Operator for Kubernetes. [Verify your installation]( {{< relref "/verifying" >}} ).
