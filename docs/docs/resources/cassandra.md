---
title: "Cassandra"
linkTitle: "Cassandra"
weight: 55
---

Aiven for Apache Cassandra® is a distributed database designed to handle large volumes of writes.

!!! note 
    Before going through this guide, make sure you have a [Kubernetes cluster](../../installation/prerequisites/) with the [operator installed](../../installation/) and a [Kubernetes Secret with an Aiven authentication token](../../authentication/).

## Creating a Cassandra instance

1\. Create a file named `cassandra-sample.yaml`, and add the following content: 

```yaml
apiVersion: aiven.io/v1alpha1
kind: Cassandra
metadata:
  name: cassandra-sample
spec:
  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # outputs the Cassandra connection on the `cassandra-secret` Secret
  connInfoSecretTarget:
    name: cassandra-secret

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
$ kubectl apply -f cassandra-sample.yaml 
```

The output is:

```shell
cassandra.aiven.io/cassandra-sample created
```

3\. Review the resource you created with this command:

```shell
$ kubectl describe cassandra.aiven.io cassandra-sample
```

The output is similar to the following:

```shell
...
Status:
  Conditions:
    Last Transition Time:  2023-01-31T10:17:25Z
    Message:               Instance was created or update on Aiven side
    Reason:                Created
    Status:                True
    Type:                  Initialized
    Last Transition Time:  2023-01-31T10:24:00Z
    Message:               Instance is running on Aiven side
    Reason:                CheckRunning
    Status:                True
    Type:                  Running
  State:                   RUNNING
...
```

The resource can be in the `REBUILDING` state for a few minutes. Once the state changes to `RUNNING`, you can access the resource.

## Using the connection Secret

For your convenience, the operator automatically stores the Cassandra connection information in a Secret created with the
name specified on the `connInfoSecretTarget` field.

To view the details of the Secret, use the following command:

```shell
$ kubectl describe secret cassandra-secret 
```

The output is similar to the following:

```shell
Name:         cassandra-secret
Namespace:    default
Labels:       <none>
Annotations:  <none>

Type:  Opaque

Data
====
CASSANDRA_HOSTS:     59 bytes
CASSANDRA_PASSWORD:  24 bytes
CASSANDRA_PORT:      5 bytes
CASSANDRA_URI:       66 bytes
CASSANDRA_USER:      8 bytes
CASSANDRA_HOST:      60 bytes
```

You can use the [jq](https://github.com/stedolan/jq) to quickly decode the Secret:

```shell
$ kubectl get secret cassandra-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```json
{
  "CASSANDRA_HOST": "<secret>",
  "CASSANDRA_HOSTS": "<secret>",
  "CASSANDRA_PASSWORD": "<secret>",
  "CASSANDRA_PORT": "14609",
  "CASSANDRA_URI": "<secret>",
  "CASSANDRA_USER": "avnadmin"
}

```

## Creating a Cassandra user

You can create service users for your instance of Aiven for Apache Cassandra. Service users are unique to this instance and are not shared with any other services.

1\. Create a file named cassandra-service-user.yaml:

```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: cassandra-service-user
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: cassandra-service-user-secret

  project: <your-project-name>
  serviceName: cassandra-sample
```

2\. Create the user by applying the configuration:

```shell
$ kubectl apply -f cassandra-service-user.yaml
```

The `ServiceUser` resource generates a Secret with connection information. 

3\. View the details of the Secret using the following command:

```shell
$ kubectl get secret cassandra-service-user-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```json
{
  "ACCESS_CERT": "<secret>",
  "ACCESS_KEY": "<secret>",
  "CA_CERT": "<secret>",
  "HOST": "<secret>",
  "PASSWORD": "<secret>",
  "PORT": "14609",
  "USERNAME": "cassandra-service-user"
}
```

You can connect to the Cassandra instance using these credentials and the host information from the `cassandra-secret` Secret.
