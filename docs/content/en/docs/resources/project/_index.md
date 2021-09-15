---
title: "Aiven Project"
linkTitle: "Aiven Project"
weight: 5
---

> Before going through this guide, make sure you have a [Kubernetes cluster](../installation/prerequisites/) with the [operator installed](../installation/) and a [Kubernetes Secret with an Aiven authentication token](../authentication/).

The `Project` CRD allows you to create Aiven Projects, where your resources can be located.

To create a fully working Aiven Project with the Aiven Operator you need a source Aiven Project already created with a working billing configuration, like a credit card.

Create a file named `project-sample.yaml` with the following content:
```yaml
apiVersion: aiven.io/v1alpha1
kind: Project
metadata:
  name: project-sample
spec:
  # the source Project to copy the billing information from
  copyFromProject: <your-source-project>

  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: project-sample
```

Apply the resource with:
```bash
$ kubectl apply -f project-sample.yaml
```

Verify the newly created Project:
```bash
$ kubectl get projects.aiven.io project-sample

NAME             AGE
project-sample   22s
```