---
title: "Redis"
linkTitle: "Redis"
weight: 50
---

Aiven for Redis®* is a fully managed in-memory NoSQL database that you can deploy in the cloud of your choice to store and access data quickly and efficiently.

!!! note
    Before going through this guide, make sure you have a [Kubernetes cluster](../../installation/prerequisites/) with the [operator installed](../../installation/)
    and a [Kubernetes Secret with an Aiven authentication token](../../authentication/).

## Creating a Redis instance

1\. Create a file named `redis-sample.yaml`, and add the following content:

```yaml
apiVersion: aiven.io/v1alpha1
kind: Redis
metadata:
  name: redis-sample
spec:
  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # outputs the Redis connection on the `redis-secret` Secret
  connInfoSecretTarget:
    name: redis-secret

  # add your Project name here
  project: <your-project-name>

  # cloud provider and plan of your choice
  # you can check all of the possibilities here https://aiven.io/pricing
  cloudName: google-europe-west1
  plan: startup-4

  # general Aiven configuration
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

  # specific Redis configuration
  userConfig:
    redis_maxmemory_policy: "allkeys-random"
```

2\. Create the service by applying the configuration:

```shell
kubectl apply -f redis-sample.yaml
```

3\. Review the resource you created with this command:

```shell
kubectl describe redis.aiven.io redis-sample
```

The output is similar to the following:

```{ .shell .no-copy }
...
Status:
  Conditions:
    Last Transition Time:  2023-01-19T14:48:59Z
    Message:               Instance was created or update on Aiven side
    Reason:                Created
    Status:                True
    Type:                  Initialized
    Last Transition Time:  2023-01-19T14:48:59Z
    Message:               Instance was created or update on Aiven side, status remains unknown
    Reason:                Created
    Status:                Unknown
    Type:                  Running
  State:                   REBUILDING
...
```

The resource will be in the `REBUILDING` state for a few minutes. Once the state changes to `RUNNING`, you can access the resource.

## Using the connection Secret

For your convenience, the operator automatically stores the Redis connection information in a Secret created with the
name specified on the `connInfoSecretTarget` field.

To view the details of the Secret, use the following command:

```shell
kubectl describe secret redis-secret
```

The output is similar to the following:

```{ .shell .no-copy }

Name:         redis-secret
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

You can use the [jq](https://github.com/stedolan/jq) to quickly decode the Secret:

```shell
kubectl get secret redis-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .shell .no-copy }
{
  "HOST": "redis-sample-your-project.aivencloud.com",
  "PASSWORD": "<secret-password>",
  "PORT": "14610",
  "SSL": "required",
  "USER": "default"
}
```
