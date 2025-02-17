---
title: "Valkey"
linkTitle: "Valkey"
weight: 50
---

Aiven for Valkey is a fully managed in-memory NoSQL database that you can deploy in the cloud of your choice to store and access data quickly and efficiently.

## Prerequisites

* A Kubernetes cluster with Aiven Kubernetes Operator installed using [helm](../installation/helm.md) or [kubectl](../installation/kubectl.md).
* A [Kubernetes Secret with an Aiven authentication token](../authentication.md).

## Create a Valkey instance

1\. Create a file named `valkey-sample.yaml`, and add the following content:

```yaml
apiVersion: aiven.io/v1alpha1
kind: Valkey
metadata:
  name: valkey-sample
spec:
  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # outputs the Valkey connection on the `valkey-secret` Secret
  connInfoSecretTarget:
    name: valkey-secret

  # your Aiven project
  project: PROJECT_NAME

  # cloud provider and plan of your choice
  # you can check all of the possibilities here https://aiven.io/pricing
  cloudName: google-europe-west1
  plan: startup-4

  # general Aiven configuration
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

  # specific Valkey configuration
  userConfig:
    valkey_maxmemory_policy: "allkeys-random"
```
Where `PROJECT_NAME` is the name of your [Aiven project](https://aiven.io/docs/platform/concepts/orgs-units-projects#projects).

2\. Create the service by applying the configuration:

```shell
kubectl apply -f valkey-sample.yaml
```

3\. Review the resource you created with this command:

```shell
kubectl describe valkey.aiven.io valkey-sample
```

The output is similar to the following:

```{ .shell .no-copy }
...
Status:
  Conditions:
    Last Transition Time:  2025-01-19T14:48:59Z
    Message:               Successfully created or updated the instance in Aiven
    Reason:                Created
    Status:                True
    Type:                  Initialized
    Last Transition Time:  2025-01-19T14:48:59Z
    Message:               Successfully created or updated the instance in Aiven, status remains unknown
    Reason:                Created
    Status:                Unknown
    Type:                  Running
  State:                   REBUILDING
...
```

The resource will be in the `REBUILDING` state for a few minutes. Once the state changes to `RUNNING`, you can access the resource.

## Use the connection Secret

For your convenience, the operator automatically stores the Valkey connection information in a Secret created with the
name specified on the `connInfoSecretTarget` field.

To view the details of the Secret, use the following command:

```shell
kubectl describe secret valkey-secret
```

The output is similar to the following:

```{ .shell .no-copy }

Name:         valkey-secret
Namespace:    default
Labels:       <none>
Annotations:  <none>

Type:  Opaque

Data
====
SSL:       8 bytes
USER:      7 bytes
HOST:      60 bytes
PASSWORD:  24 bytes
PORT:      5 bytes
```

You can use [jq](https://github.com/jqlang/jq) to quickly decode the Secret:

```shell
kubectl get secret valkey-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .shell .no-copy }
{
  "HOST": "valkey-sample-your-project.aivencloud.com",
  "PASSWORD": "<secret-password>",
  "PORT": "14610",
  "SSL": "required",
  "USER": "default"
}
```
