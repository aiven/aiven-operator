---
title: "Installing with kubectl"
linkTitle: "Installing with kubectl"
weight: 15
---

## Installing

All Aiven Operator for Kubernetes components can be installed from one YAML file that is uploaded for every release:

```bash
$ kubectl apply -f https://github.com/aiven/aiven-operator/releases/latest/download/deployment.yaml
```

By default the Deployment is installed into the `aiven-operator-system` namespace.

## Uninstalling

Assuming you installed version `vX.Y.Z` of the operator it can be uninstalled via

```bash
$ kubectl delete -f https://github.com/aiven/aiven-operator/releases/download/vX.Y.Z/deployment.yaml
```
