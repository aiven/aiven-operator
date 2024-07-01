---
title: "Uninstall"
linkTitle: "Uninstall"
weight: 90
---

!!! danger

    Uninstalling the Aiven Operator for Kubernetes can remove the resources created in Aiven, possibly resulting in data loss.

Depending on your installation, please follow one of:

- [Helm](../installation/helm.md#uninstalling)
- [kubectl](../installation/kubectl.md#uninstalling)

## Expired tokens

Aiven resources need to have an accompanying secret that contains the token that is used to authorize the manipulation of that resource.
If that token expired then you will not be able to delete the custom resource and deletion will also hang until the situation is resolved.
The recommended approach to deal with that situation is to patch a valid token into the secret again so that proper cleanup of aiven resources can take place.

## Hanging deletions

To protect the Secrets that the operator is using from deletion, it adds the [finalizer](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/) `finalizers.aiven.io/needed-to-delete-services` to the Secret.
This solves a race condition that happens when deleting a namespace, where there is a possibility of the Secret getting deleted before the resource that uses it.
When the controller is deleted it may not cleanup the finalizers from all Secrets.
If there is a Secret with this finalizer blocking deletion of a namespace, you can remove the finalizer by running:

```shell
kubectl patch secret <offending-secret> -p '{"metadata":{"finalizers":null}}' --type=merge
```
