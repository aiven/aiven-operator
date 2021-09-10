---
title: "Uninstalling"
linkTitle: "Uninstalling"
weight: 90
---

## 🚨 Warning 🚨

Uninstalling the Aiven Operator for Kubernetes can remove the resources created in Aiven, possibly resulting in data loss.

Depending on your installation, please follow one of:

* [Helm]({{< relref "/docs/installation/helm" >}}#uninstalling)
* [kubectl]({{< relref "/docs/installation/kubectl" >}}#uninstalling)

## Dealing with expired tokens

Aiven resources need to have an accompanying secret that contains the token that is used to authorize the manipulation of that resource.
If that token expired then you will not be able to delete the custom resource and deletion will also hang until the situation is resolved.
The recommended approach to deal with that situation is to patch a valid token into the secret again so that proper cleanup of aiven resources can take place.

## Hanging deletions

To protect the secrets that the operator is using from deletion, it adds the [finalizer](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/) `finalizers.aiven.io/needed-to-delete-services` to the secret.
This solves a race condition that happens when deleting a namespace, where there is a possibility of the secret getting deleted before the resource that uses it.
When the controller is deleted it may not cleanup the finalizers from all secrets.
If there is a secret with this finalizer blocking deletion of a namespace, for now please do

```bash
kubectl patch secret <offending-secret> -p '{"metadata":{"finalizers":null}}' --type=merge
```

to remove the finalizer.
