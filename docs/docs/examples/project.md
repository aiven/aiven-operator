---
title: "Aiven Project"
linkTitle: "Aiven Project"
weight: 5
---

The `Project` CRD allows you to create Aiven Projects, where your resources can be located.

## Prerequisites

* A Kubernetes cluster with Aiven Kubernetes Operator installed using [helm](../installation/helm.md) or [kubectl](../installation/kubectl.md).
* A [Kubernetes Secret with an Aiven authentication token](../authentication.md).

## Create a project

To create a fully working Aiven Project with the Aiven Operator you need a source Aiven Project already created with a working billing configuration, like a credit card.

Create a file named `project-sample.yaml` with the following content:

```yaml
apiVersion: aiven.io/v1alpha1
kind: Project
metadata:
  name: project-sample
spec:
  # the source Project to copy the billing information from
  copyFromProject: SOURCE_PROJECT

  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: project-sample
```

Apply the resource with:

```shell
kubectl apply -f project-sample.yaml
```

Verify the newly created Project:

```shell
kubectl get projects.aiven.io project-sample
```

The output is similar to the following:

```{ .shell .no-copy }
NAME             AGE
project-sample   22s
```
