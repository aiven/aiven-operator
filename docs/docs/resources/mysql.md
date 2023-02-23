---
title: "MySQL"
linkTitle: "MySQL"
weight: 46
---

Aiven for MySQL is a fully managed relational database service, deployable in the cloud of your choice. 

> Before going through this guide, make sure you have a [Kubernetes cluster](../../installation/prerequisites/) with the [operator installed](../../installation/) and a [Kubernetes Secret with an Aiven authentication token](../../authentication/).

## Creating a MySQL instance

1\. Create a file named `mysql-sample.yaml`, and add the following content: 

```yaml
apiVersion: aiven.io/v1alpha1
kind: MySQL
metadata:
  name: mysql-sample
spec:
  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # outputs the MySQL connection on the `mysql-secret` Secret
  connInfoSecretTarget:
    name: mysql-secret

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
kubectl apply -f mysql-sample.yaml 
```

3\. Review the resource you created with this command:

```shell
kubectl describe mysql.aiven.io mysql-sample
```

The output is similar to the following:

```{ .shell .no-copy }
...
Status:
  Conditions:
    Last Transition Time:  2023-02-22T15:43:44Z
    Message:               Instance was created or update on Aiven side
    Reason:                Created
    Status:                True
    Type:                  Initialized
    Last Transition Time:  2023-02-22T15:43:44Z
    Message:               Instance was created or update on Aiven side, status remains unknown
    Reason:                Created
    Status:                Unknown
    Type:                  Running
  State:                   REBUILDING
...
```

The resource will be in the `REBUILDING` state for a few minutes. Once the state changes to `RUNNING`, you can access the resource.


## Using the connection Secret

For your convenience, the operator automatically stores the MySQL connection information in a Secret created with the
name specified on the `connInfoSecretTarget` field.

To view the details of the Secret, use the following command:

```shell
kubectl describe secret mysql-secret 
```

The output is similar to the following:

```{ .shell .no-copy }
Name:         mysql-secret
Namespace:    default
Labels:       <none>
Annotations:  <none>

Type:  Opaque

Data
====
MYSQL_PORT:      5 bytes
MYSQL_SSL_MODE:  8 bytes
MYSQL_URI:       115 bytes
MYSQL_USER:      8 bytes
MYSQL_DATABASE:  9 bytes
MYSQL_HOST:      39 bytes
MYSQL_PASSWORD:  24 bytes
```

You can use [jq](https://github.com/stedolan/jq) to quickly decode the Secret:

```shell
kubectl get secret mysql-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy}
{
  "MYSQL_DATABASE": "defaultdb",
  "MYSQL_HOST": "<secret>",
  "MYSQL_PASSWORD": "<secret>",
  "MYSQL_PORT": "12691",
  "MYSQL_SSL_MODE": "REQUIRED",
  "MYSQL_URI": "<secret>",
  "MYSQL_USER": "avnadmin"
}
```

## Creating a MySQL user

You can create service users for your instance of Aiven for MySQL. Service users are unique to this instance and are not shared with any other services.

1\. Create a file named mysql-service-user.yaml:

```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: mysql-service-user
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: mysql-service-user-secret

  project: <your-project-name>
  serviceName: mysql-sample
```

2\. Create the user by applying the configuration:

```shell
kubectl apply -f mysql-service-user.yaml
```

The `ServiceUser` resource generates a Secret with connection information. 

3\. View the details of the Secret using [jq](https://github.com/stedolan/jq):

```shell
kubectl get secret mysql-service-user-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
  "ACCESS_CERT": "<secret>",
  "ACCESS_KEY": "<secret>",
  "CA_CERT": "<secret>",
  "HOST": "<secret>",
  "PASSWORD": "<secret>",
  "PORT": "14609",
  "USERNAME": "mysql-service-user"
}
```

You can connect to the MySQL instance using these credentials and the host information from the `mysql-secret` Secret.