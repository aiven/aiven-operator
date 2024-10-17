---
title: "Install with Helm (recommended)"
linkTitle: "Install with Helm (recommended)"
weight: 10
---

# Install with Helm

The Aiven Operator for Kubernetes® can be installed with [Helm](https://helm.sh/). Before you start, make sure you have the [prerequisites](prerequisites.md).

1\. Add the [Aiven Helm chart repository](https://github.com/aiven/aiven-charts) and update yoru local Helm information by running:

   ```shell
   helm repo add aiven https://aiven.github.io/aiven-charts && helm repo update
   ```

2\. Install Custom Resource Definitions (CRDs) by running:

   ```shell
   helm install aiven-operator-crds aiven/aiven-operator-crds
   ```

3\. To verify the installation, run:

   ```shell
   kubectl api-resources --api-group=aiven.io
   ```

   The output is similar to the following:

   ```{ .shell .no-copy }
   NAME                  SHORTNAMES   APIVERSION          NAMESPACED   KIND
   connectionpools                    aiven.io/v1alpha1   true         ConnectionPool
   databases                          aiven.io/v1alpha1   true         Database
   ...
   ```

4\. To install the Aiven Operator, run:

   ```shell
   helm install aiven-operator aiven/aiven-operator
   ```

!!! note
    Installation will fail if webhooks are enabled and the CRDs for the cert-manager are not installed.
    Alternatively, you can install without webhooks enabled by running:
    ```shell
    helm install aiven-operator aiven/aiven-operator --set webhooks.enabled=false
    ```

5\. To verify the installation, run:

   ```shell
   helm status aiven-operator
   ```

   The output is similar to the following:

   ```{ .shell .no-copy }
   NAME: aiven-operator
   LAST DEPLOYED: Fri Sep 10 15:23:26 2021
   NAMESPACE: default
   STATUS: deployed
   REVISION: 1
   TEST SUITE: None
   ```

6\. To verify the operator pod is running, run:

   ```bash
   kubectl get pod -l app.kubernetes.io/name=aiven-operator
   ```

## Install without full cluster administrator access

If you don't have the necessary permissions to create cluster-wide resources such as `ClusterRole` and `ClusterRoleBinding` when installing the Helm chart,
a cluster administrator can manually install these roles.
This ensures that the operator can function properly.

## Configuration options

Refer to the [values.yaml file](https://github.com/aiven/aiven-charts/blob/main/charts/aiven-operator/values.yaml) of the chart.

### Restrict operator access to specific namespaces

You can configure the operator to monitor resources within specific namespaces.
If the `ClusterRole` is enabled, it's bound to the operator's `ServiceAccount` within each watched namespace using a `RoleBinding`.
This setup grants the operator the permissions specified in the `ClusterRole`, but only within the context of the specific namespaces where the `RoleBinding` is created.
