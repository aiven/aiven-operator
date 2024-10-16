---
title: "Install with kubectl"
linkTitle: "Install with kubectl"
weight: 15
---

# Install with kubectl

The Aiven Operator for Kubernetes can be installed with kubectl. Before you start, make sure you have the [prerequisites](prerequisites.md).

All Aiven Operator for Kubernetes components can be installed from one YAML file that is uploaded for every release.

To install the latest version, run:

```shell
kubectl apply -f https://github.com/aiven/aiven-operator/releases/latest/download/deployment.yaml
```

By default the deployment is installed into the `aiven-operator-system` namespace.