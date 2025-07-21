---
title: "Troubleshooting"
linkTitle: "Troubleshooting"
weight: 80
---

## Verify operator status

Use the following checks to help you troubleshoot the Aiven Operator for Kubernetes.

### Check the Pods

Verify that all the operator Pods are `READY`, and the `STATUS` is `Running`.

```shell
kubectl get pod -n aiven-operator-system
```

The output is similar to the following:

```{ .shell .no-copy }

NAME                                                            READY   STATUS    RESTARTS   AGE
aiven-operator-controller-manager-576d944499-ggttj   1/1     Running   0          12m
```

Verify that the `cert-manager` Pods are also running.

```shell
kubectl get pod -n cert-manager
```

The output has the status:

```{ .shell .no-copy }
NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-7dd5854bb4-85cpv              1/1     Running   0          76s
cert-manager-cainjector-64c949654c-n2z8l   1/1     Running   0          77s
cert-manager-webhook-6bdffc7c9d-47w6z      1/1     Running   0          76s
```

### Visualize the operator logs

Use the following command to visualize all the logs from the operator.

```shell
kubectl logs -n aiven-operator-system -l control-plane=controller-manager
```

### Verify the operator version

```shell
kubectl get pod -n aiven-operator-system -l control-plane=controller-manager -o jsonpath="{.items[0].spec.containers[0].image}"
```

## Known issues and limitations

We're always working to resolve problems that pop up in Aiven products. If your problem is listed below, we know about
it and are working to fix it. If your problem isn't listed below, report it as an issue.

### cert-manager

#### Issue

The following event appears on the operator Pod:

```{ .shell .no-copy }
MountVolume.SetUp failed for volume "cert" : secret "webhook-server-cert" not found
```

#### Impact

You cannot run the operator.

#### Solution

Make sure that cert-manager is up and running.

```shell
kubectl get pod -n cert-manager
```

The output shows the status of each cert-manager:

```{ .shell .no-copy }
NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-7dd5854bb4-85cpv              1/1     Running   0          76s
cert-manager-cainjector-64c949654c-n2z8l   1/1     Running   0          77s
cert-manager-webhook-6bdffc7c9d-47w6z      1/1     Running   0          76s
```

## Resource Reconciliation Behavior

The Aiven Operator is designed to stop reconciling resources once they reach a ready state. This behavior is different from some other Kubernetes operators and can cause confusion.

### When Reconciliation Stops

The operator stops reconciling a resource when **both** of the following conditions are met:

1. **Latest generation processed** - The operator has processed the latest changes from the Kubernetes resource specification
2. **Resource is running** - The resource has been successfully created in Aiven and is in a running state

You can check if a resource is in this "ready" state by looking for these annotations:

```shell
kubectl describe <resource-type> <resource-name> -n <namespace>
```

Look for these annotations:
- `controllers.aiven.io/instance-is-running`: Indicates the resource is running in Aiven
- `controllers.aiven.io/generation-was-processed`: Indicates the latest spec changes have been processed

### Common Misconception

**Expected behavior**: "If I manually delete a resource from the Aiven console, the operator should recreate it automatically."

**Actual behavior**: Once a resource/secret is ready, the operator stops continuous reconciliation. Manual changes in the Aiven console will **not** trigger automatic recreation.

### How to Force Reconciliation

If you need to force the operator to reconcile a resource (for example, after manually deleting it from Aiven), you have several options:

#### Option 1: Delete and Recreate the Kubernetes Resource

**Note:** The resource will be deleted on the Aiven side and data may be lost when using this approach.

```shell
kubectl delete <resource-type> <resource-name> -n <namespace>
kubectl apply -f your-resource.yaml
```

#### Option 2: Remove the Generation Annotation
Remove the `controllers.aiven.io/generation-was-processed` annotation to force reconciliation without affecting the resource's functionality:
```shell
kubectl annotate <resource-type> <resource-name> -n <namespace> controllers.aiven.io/generation-was-processed-
```

Example:
```shell
kubectl annotate kafkatopic orders-topic -n aiven controllers.aiven.io/generation-was-processed-
```

This is the cleanest approach as it doesn't modify the resource specification or create any side effects.