---
title: "OpenSearch"
linkTitle: "OpenSearch"
weight: 45
---

OpenSearch® is an open source search and analytics suite including search engine, NoSQL document database, and visualization interface. OpenSearch offers a distributed, full-text search engine based on Apache Lucene® with a RESTful API interface and support for JSON documents.

!!! note
Before going through this guide, make sure you have a [Kubernetes cluster](../installation/prerequisites.md) with the operator installed (see instructions for [helm](../installation/helm.md) or [kubectl](../installation/kubectl.md))
and a [Kubernetes Secret with an Aiven authentication token](../authentication.md).

## Creating an OpenSearch instance

1\. Create a file named `os-sample.yaml`, and add the following content:

```yaml
apiVersion: aiven.io/v1alpha1
kind: OpenSearch
metadata:
  name: os-sample
spec:
  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # outputs the OpenSearch connection on the `os-secret` Secret
  connInfoSecretTarget:
    name: os-secret

  # add your Project name here
  project: <your-project-name>

  # cloud provider and plan of your choice
  # you can check all of the possibilities here https://aiven.io/pricing
  cloudName: google-europe-west1
  plan: startup-4

  # general Aiven configuration
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
```

2\. Create the service by applying the configuration:

```shell
kubectl apply -f os-sample.yaml
```

3\. Review the resource you created with this command:

```shell
kubectl describe opensearch.aiven.io os-sample
```

The output is similar to the following:

```{ .shell .no-copy }
...
Status:
  Conditions:
    Last Transition Time:  2023-01-19T14:41:43Z
    Message:               Instance was created or update on Aiven side
    Reason:                Created
    Status:                True
    Type:                  Initialized
    Last Transition Time:  2023-01-19T14:41:43Z
    Message:               Instance was created or update on Aiven side, status remains unknown
    Reason:                Created
    Status:                Unknown
    Type:                  Running
  State:                   REBUILDING
...
```

The resource will be in the `REBUILDING` state for a few minutes. Once the state changes to `RUNNING`, you can access the resource.

## Using the connection Secret

For your convenience, the operator automatically stores the OpenSearch connection information in a Secret created with the
name specified on the `connInfoSecretTarget` field.

To view the details of the Secret, use the following command:

```shell
kubectl describe secret os-secret
```

The output is similar to the following:

```{ .shell .no-copy }
Name:         os-secret
Namespace:    default
Labels:       <none>
Annotations:  <none>

Type:  Opaque

Data
====
HOST:      61 bytes
PASSWORD:  24 bytes
PORT:      5 bytes
USER:      8 bytes
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the Secret:

```shell
kubectl get secret os-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
  "HOST": "os-sample-your-project.aivencloud.com",
  "PASSWORD": "<secret>",
  "PORT": "13041",
  "USER": "avnadmin"
}
```

## Creating an OpenSearch user

You can create service users for your instance of Aiven for OpenSearch. Service users are unique to this instance and are not shared with any other services.

1\. Create a file named os-service-user.yaml:

```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: os-service-user
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: os-service-user-secret

  project: <your-project-name>
  serviceName: os-sample
```

2\. Create the user by applying the configuration:

```shell
kubectl apply -f os-service-user.yaml
```

The `ServiceUser` resource generates a Secret with connection information.

3\. View the details of the Secret using the following command:

```shell
kubectl get secret os-service-user-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
  "ACCESS_CERT": "<secret>",
  "ACCESS_KEY": "<secret>",
  "CA_CERT": "<secret>",
  "HOST": "os-sample-your-project.aivencloud.com",
  "PASSWORD": "<secret>",
  "PORT": "14609",
  "USERNAME": "os-service-user"
}
```

You can connect to the OpenSearch instance using these credentials and the host information from the `os-secret` Secret.
