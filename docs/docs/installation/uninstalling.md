---
title: "Uninstall"
linkTitle: "Uninstall"
weight: 90
---

# Uninstall

Depending on your installation, you can uninstall the Aiven Operator using Helm or kubectl. 

!!! danger

    Uninstalling the Aiven Operator for Kubernetes can remove the resources created in Aiven, which can result in data loss.

## Uninstall with Helm

1. To get the name of your deployment, run:

```shell
helm list
```

The output has the name of each deployment similar to the following:

```{ .shell .no-copy }
NAME                NAMESPACE REVISION UPDATED                                  STATUS   CHART                      APP VERSION
aiven-operator      default   1        2021-09-09 10:56:14.623700249 +0200 CEST deployed aiven-operator-v0.1.0      v0.1.0
aiven-operator-crds default   1        2021-09-09 10:56:05.736411868 +0200 CEST deployed aiven-operator-crds-v0.1.0 v0.1.0
```

2. To remove the CRDs, run:

```shell
helm uninstall aiven-operator-crds
```

The confirmation message is similar to the following:

```{ .shell .no-copy }
release "aiven-operator-crds" uninstalled
```

3. To remove the operator, run:

```shell
helm uninstall aiven-operator
```

The confirmation message is similar to the following:

```{ .shell .no-copy }
release "aiven-operator" uninstalled
```

## Uninstall with kubectl

To uninstall the operator, run:

```shell
kubectl delete -f https://github.com/aiven/aiven-operator/releases/download/vX.Y.Z/deployment.yaml
```

Where `vX.Y.Z` is the version of the operator you installed.

## Expired tokens

Aiven resources need to have an accompanying secret that contains the token that is used to authorize the manipulation of that resource.
If that token expired then you will not be able to delete the custom resource and deletion will also hang until the situation is resolved.
The recommended approach to deal with that situation is to patch a valid token into the secret again so that proper cleanup of aiven resources can take place.

## Hanging deletions

To protect the Secrets that the operator is using from deletion, it adds the [finalizer](https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/) `finalizers.aiven.io/needed-to-delete-services` to the Secrets.
This solves a race condition that happens when deleting a namespace, where there is a possibility of the Secret getting deleted before the resource that uses it.
When the controller is deleted it may not cleanup the finalizers from all Secrets.
If there is a Secret with this finalizer blocking deletion of a namespace, you can remove the finalizer.

To remove a finalizer, run:

```shell
kubectl patch secret <offending-secret> -p '{"metadata":{"finalizers":null}}' --type=merge
```