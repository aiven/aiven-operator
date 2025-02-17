---
title: "ClickHouse"
linkTitle: "ClickHouse"
weight: 55
---

Aiven for ClickHouse® is a fully managed distributed columnar database based on open source ClickHouse.

## Prerequisites

* A Kubernetes cluster with Aiven Kubernetes Operator installed using [helm](../installation/helm.md) or [kubectl](../installation/kubectl.md).
* A [Kubernetes Secret with an Aiven authentication token](../authentication.md).

## Create a ClickHouse instance

1\. Create a file named `clickhouse-sample.yaml`, and add the following:

```yaml
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: example-clickhouse
spec:
  authSecretRef:
    name: aiven-token
    key: token
  # Outputs the ClickHouse connection to the `clickhouse-secret` Secret.
  connInfoSecretTarget:
    name: clickhouse-secret
  # Your Aiven project.
  project: PROJECT_NAME
  # Choose a cloud provider and plan.
  # View the options on the pricing page at https://aiven.io/pricing.
  cloudName: google-europe-west1
  plan: startup-16
  # Configure the maintenance window.
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
```
Where `PROJECT_NAME` is the name of your [Aiven project](https://aiven.io/docs/platform/concepts/orgs-units-projects#projects).

2\. To create the service, run:

```shell
kubectl apply -f clickhouse-sample.yaml
```

The output is similar to the following:

```shell
clickhouse.aiven.io/example-clickhouse created
```

3\. To review the resource you created, run:

```shell
kubectl describe clickhouse.aiven.io example-clickhouse
```

The output is similar to the following:

```shell
...
Status:
  Conditions:
    Last Transition Time:  2024-06-25T07:58:37Z
    Message:               Instance was created or update on Aiven side
    Reason:                Created
    Status:                True
    Type:                  Initialized
    Last Transition Time:  2024-06-25T08:01:47Z
    Message:               Instance is running on Aiven side
    Reason:                CheckRunning
    Status:                True
    Type:                  Running
  State:                   RUNNING
Events:
  Type    Reason                   Age                    From                   Message
  ----    ------                   ----                   ----                   -------
  Normal  CreatedOrUpdatedAtAiven  3m24s (x5 over 3m45s)  clickhouse-reconciler  waiting for the instance to be running
  Normal  ReconcilationStarted     3m14s (x6 over 3m47s)  clickhouse-reconciler  starting reconciliation
  Normal  InstanceFinalizerAdded   3m14s (x6 over 3m47s)  clickhouse-reconciler  waiting for preconditions of the instance
...
```
The resource can be in the `REBUILDING` state for a few minutes. Once the state changes to `RUNNING`, you can access the service.

## Use the connection Secret

Aiven Operator automatically stores the ClickHouse connection information in a Secret created with the name in the `connInfoSecretTarget` field.

To view the details of the Secret, run:

```shell
kubectl describe secret clickhouse-secret
```
The output is similar to the following:

```shell
Name:         clickhouse-secret
Namespace:    default
Labels:       <none>
Annotations:  <none>

Type:  Opaque

Data
====
CLICKHOUSE_USER:      8 bytes
HOST:                 46 bytes
PASSWORD:             24 bytes
PORT:                 5 bytes
USER:                 8 bytes
CLICKHOUSE_HOST:      46 bytes
CLICKHOUSE_PASSWORD:  24 bytes
CLICKHOUSE_PORT:      5 bytes
```

You can use the  JSON processor [jq](https://github.com/jqlang/jq) to decode the Secret:

```shell
kubectl get secret clickhouse-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```json
{
  "CLICKHOUSE_HOST": "HOST",
  "CLICKHOUSE_PASSWORD": "SERVICE_PASSWORD",
  "CLICKHOUSE_PORT": "12691",
  "CLICKHOUSE_USER": "avnadmin",
  "HOST": "HOST",
  "PASSWORD": "ADMIN_USER_PASSWORD",
  "PORT": "12691",
  "USER": "avnadmin"
}
```

## Create a ClickHouse database

!!! note
    Tables cannot be created using Aiven Operator. To create a table,
    use the [Aiven Console or CLI](https://aiven.io/docs/products/clickhouse/howto/manage-databases-tables#create-a-table).

1\. Create a file named `clickhouse-db.yaml` and add the following:

```yaml
apiVersion: aiven.io/v1alpha1
kind: ClickhouseDatabase
metadata:
  name: example-database
spec:
  authSecretRef:
    name: aiven-token
    key: token
  serviceName: example-clickhouse
  project: PROJECT_NAME
```

Where `PROJECT_NAME` is the name of your Aiven project.

2\. To create the database, run:

```shell
kubectl apply -f clickhouse-db.yaml
```

3\. To view details about the resource, run:

```shell
kubectl describe ClickhouseDatabase example-database
```

The output is similar to the following:

```shell
...
Spec:
  Auth Secret Ref:
    Key:         token
    Name:        aiven-token
  Project:       example-project
  Service Name:  example-clickhouse
Status:
  Conditions:
    Last Transition Time:  2024-06-25T13:58:35Z
    Message:               Checking preconditions
    Reason:                Preconditions
    Status:                True
    Type:                  Initialized
    Last Transition Time:  2024-06-25T14:32:05Z
    Message:               Instance is running on Aiven side
    Reason:                CheckRunning
    Status:                True
    Type:                  Running
...
```

## Create a ClickHouse user and role

You can create service users and roles for an instance of ClickHouse, and grant privileges to them. Users and roles are not shared with any other services.

1\. Create a file named `clickhouse-service-users.yaml` and add the following configuration for a service user:

```yaml
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: example-user
spec:
  authSecretRef:
    name: aiven-token
    key: token
  connInfoSecretTarget:
    name: clickhouse-service-user-secret
  serviceName: example-clickhouse
  project: PROJECT_NAME
```
Where `PROJECT_NAME` is the name of your Aiven project.

This resource generates a Secret with connection information and stores it in `clickhouse-service-user-secret`.

2\. To create a role add the following to the same file:

```yaml

---

apiVersion: aiven.io/v1alpha1
kind: ClickhouseRole
metadata:
  name: example-role
spec:
  authSecretRef:
    name: aiven-token
    key: token
  serviceName: example-clickhouse
  project: PROJECT_NAME
  role: read-only
```

3\. Grant [privileges](https://clickhouse.com/docs/en/sql-reference/statements/grant) to the role,
and assign the role to the user by adding the following to the same file:

```yaml

---

apiVersion: aiven.io/v1alpha1
kind: ClickhouseGrant
metadata:
  name: example-grant
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: PROJECT_NAME
  serviceName: example-clickhouse

  privilegeGrants:
    - grantees:
        - role: read-only
      privileges:
        - SELECT
      database: example-database

  roleGrants:
    - roles:
        - read-only
      grantees:
        - user: example-user
```

4\. To create the user, role, and grant, run:

```shell
kubectl apply -f clickhouse-service-users.yaml
```

5\. To get credentials and host information for connecting to this instance, view the connection
    information stored in the Secret by running:

```shell
kubectl get secret clickhouse-service-user-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```json
{
  "CLICKHOUSEUSER_HOST": "HOST",
  "CLICKHOUSEUSER_PASSWORD": "PASSWORD",
  "CLICKHOUSEUSER_PORT": "12691",
  "CLICKHOUSEUSER_USERNAME": "example-user",
  "HOST": "HOST",
  "PASSWORD": "PASSWORD",
  "PORT": "12691",
  "USERNAME": "example-user"
}
```