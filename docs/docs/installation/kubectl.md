---
title: "Install with kubectl"
linkTitle: "Install with kubectl"
weight: 15
---

## Installing

Before you start, make sure you have the [prerequisites](prerequisites.md).

All Aiven Operator for Kubernetes components can be installed from one YAML file that is uploaded for every release:

```shell
kubectl apply -f https://github.com/aiven/aiven-operator/releases/latest/download/deployment.yaml
```

By default the Deployment is installed into the `aiven-operator-system` namespace.

## Uninstalling

Assuming you installed version `vX.Y.Z` of the operator it can be uninstalled via

```shell
kubectl delete -f https://github.com/aiven/aiven-operator/releases/download/vX.Y.Z/deployment.yaml
```
