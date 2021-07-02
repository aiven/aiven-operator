---
title: "Aiven for PostgreSQL"
linkTitle: "Aiven for PostgreSQL"
weight: 20 
---

PostgreSQL is an open source, relational database. It's ideal for organisations that need a well organised tabular datastore. On top of the strict table and columns formats, PostgreSQL also offers solutions for nested datasets with the native `jsonb` format and advanced set of extensions including [PostGIS](https://postgis.net/), a spatial database extender for location queries. Aiven for PostgreSQL is the perfect fit for your relational data.

With Aiven Kubernetes Operator, you can manage Aiven for PostgreSQL through the well defined Kubernetes API.

> Before going through this guide, make sure you have a [Kubernetes cluster](../installation/prerequisites/) with the [operator installed](../installation/), and a [Kubernetes Secret with an Aiven authentication token](../authentication/).

## Creating a PostgreSQL instance
1. Create a file named `pg-sample.yaml` with the following content:
```yaml
apiVersion: aiven.io/v1alpha1
kind: PG
metadata:
  name: pg-sample
spec:

  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # outputs the PostgreSQL connection on the `pg-connection` Secret
  connInfoSecretTarget:
    name: pg-connection

  # add your Project name here
  project: <your-project-name> 

  # cloud provider and plan of your choice
  # you can check all of the possibilities here https://aiven.io/pricing
  cloudName: google-europe-west1
  plan: startup-4

  # general Aiven configuration
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

  # specific PostgreSQL configuration
  pgUserConfig:
    pg_version: '11'
```

2. Create the service by applying the configuration:
```bash
$ kubectl apply -f pg-sample.yaml
```

3. Review the resource you created with the following command:
```bash
$ kubectl get pgs.aiven.io pg-sample

NAME         PROJECT         REGION                PLAN       STATE
pg-sample    dev-advocates   google-europe-west1   hobbyist   RUNNING
```

The resource can stay in the `BUILDING` state for a couple of minutes. 
Once the state changes to `RUNNING`, you are ready to access it.

## Using the connection Secret 
For your convenience, the operator automatically stores the PostgreSQL connection information in a Secret created with the name specified on the `connInfoSecretTarget` field.

```bash
$ kubectl describe secret pg-connection 

Name:         pg-connection
Namespace:    default
Annotations:  <none>

Type:  Opaque

Data
====
DATABASE_URI:  107 bytes
PGDATABASE:    9 bytes
PGHOST:        38 bytes
PGPASSWORD:    16 bytes
PGPORT:        5 bytes
PGSSLMODE:     7 bytes
PGUSER:        8 bytes
```

You can use the [jq](https://github.com/stedolan/jq) to quickly decode the Secret:
```bash
$ kubectl get secret pg-connection -o json | jq '.data | map_values(@base64d)'

{
  "DATABASE_URI": "postgres://avnadmin:<secret-password>@pg-sample-your-project.aivencloud.com:13039/defaultdb?sslmode=require",
  "PGDATABASE": "defaultdb",
  "PGHOST": "pg-sample-your-project.aivencloud.com",
  "PGPASSWORD": "<secret-password>",
  "PGPORT": "13039",
  "PGSSLMODE": "require",
  "PGUSER": "avnadmin"
}
```

## Testing the connection
You can verify your PostgreSQL connection from a Kubernetes workload by deploying a Pod that runs the `psql` command. 

1. Create a file named `pod-psql.yaml`
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: psql-test-connection
spec:
  restartPolicy: Never
  containers:
    - image: postgres:11-alpine
      name: postgres
      command: ['psql', '$(DATABASE_URI)', '-c', 'SELECT version();']
      
      # the pg-connection Secret becomes environment variables 
      envFrom:
      - secretRef:
          name: pg-connection
```

It runs once and stops, due to the `restartPolicy: Never` flag.

2. Inspect the log:
```bash
$ kubectl logs psql-test-connection
                                           version                                           
---------------------------------------------------------------------------------------------
 PostgreSQL 11.12 on x86_64-pc-linux-gnu, compiled by gcc, a 68c5366192 p 6b9244f01a, 64-bit
(1 row)
```

You have now connected to the PostgreSQL, and executed the `SELECT version();` query. 

## Creating a PostgreSQL database
The `Database` Kubernetes resource allows you to create a logical database within the PostgreSQL instance.

-> Create the `pg-database-sample.yaml` file with the following content:
```yaml
apiVersion: aiven.io/v1alpha1
kind: Database
metadata:
  name: pg-database-sample
spec:
  authSecretRef:
    name: aiven-token
    key: token

  # the name of the previously created PostgreSQL instance
  serviceName: pg-sample 

  project: <your-project-name>
  lcCollate: en_US.UTF-8
  lcCtype: en_US.UTF-8
```

You can now connect to the `pg-database-sample` using the credentials stored in the `pg-connection` Secret.

## Creating a PostgreSQL user
Aiven uses the concept of *service user* that allows you to create users for different services. You can create one for the PostgreSQL instance.

1. Create a file named `pg-service-user.yaml`.
```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: pg-service-user
spec:
  authSecretRef:
    name: aiven-token
    key: token
  
  connInfoSecretTarget:
    name: pg-service-user-connection

  project: <your-project-name>
  serviceName: pg-sample
```

2. Apply the configuration with the following command.
```bash
$ kubectl apply -f pg-service-user.yaml
```

The `ServiceUser` resource generates a Secret with connection information, in this case named `pg-service-user-connection`:
```bash
$ kubectl get secret pg-service-user-connection -o json | jq '.data | map_values(@base64d)'

{
  "PASSWORD": "<secret-password>",
  "USERNAME": "pg-service-user"
}
```

You can now connect to the PostgreSQL instance using the credentials generated above, and the host information from the `pg-connection` Secret.

## Creating a PostgreSQL connection pool
Connection pooling allows you to maintain very large numbers of connections to a database while minimizing the consumption of server resources. See more information [here](https://help.aiven.io/en/articles/964730-postgresql-connection-pooling). Aiven for PostgreSQL uses PGBouncer for connection pooling.

You can create  a connection pool with the `ConnectionPool` resource using the previously created `Database` and `ServiceUser`.

-> Create a new file named `pg-connection-pool.yaml` with the following content:
```yaml
apiVersion: aiven.io/v1alpha1
kind: ConnectionPool
metadata:
  name: pg-connection-pool
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: pg-connection-pool-connection

  project: <your-project-name>
  serviceName: pg-sample
  databaseName: pg-database-sample
  username: pg-service-user
  poolSize: 10
  poolMode: transaction
```

The `ConnectionPool` generates a Secret with the connection info using the name from the `connInfoSecretTarget.Name` field:
```bash
$ kubectl get secret pg-connection-pool-connection -o json | jq '.data | map_values(@base64d)' 

{
  "DATABASE_URI": "postgres://pg-service-user:<secret-password>@pg-sample-you-project.aivencloud.com:13040/pg-connection-pool?sslmode=require",
  "PGDATABASE": "pg-database-sample",
  "PGHOST": "pg-sample-your-project.aivencloud.com",
  "PGPASSWORD": "<secret-password>",
  "PGPORT": "13040",
  "PGSSLMODE": "require",
  "PGUSER": "pg-service-user"
}
```
