---
title: "Aiven for PostgreSQL"
linkTitle: "Aiven for PostgreSQL"
weight: 20 
---

PostgreSQL is an open source, relational database. It's ideal for organisations that need a well organised tabular datastore. On top of the strict table and columns formats, PostgreSQL also offers solutions for nested datasets with the native `jsonb` format and advanced set of extensions including [PostGIS](https://postgis.net/), a spatial database extender for location queries. Aiven for PostgreSQL is the perfect fit for your relational data.

With Aiven Kubernetes Operator, you can manage Aiven for PostgreSQL through the well defined Kubernetes API.

> Before going through this guide, make sure to have a [Kubernetes Cluster](../installation/prerequisites/) with the [Operator installed](../installation/) and a [Kubernetes Secret with an Aiven authentication token](../authentication/).

## Create a PostgreSQL instance
Create a file named `pg-sample.yaml` with the following content:
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

Let's create the service by applying the configuration:
```bash
$ kubectl apply -f pg-sample.yaml
```

Take a look at the resource created with the following:
```bash
$ kubectl get pgs.aiven.io pg-sample

NAME         PROJECT         REGION                PLAN       STATE
pg-sample    dev-advocates   google-europe-west1   hobbyist   RUNNING
```

The resource might stay in the `BUILDING` state for a couple of minutes, enough to grab a quick coffee! Once the state becomes `RUNNING`, we are ready to access it.

## Connection Information Secret 
For your convenience, we automatically store the PostgreSQL connection information on a Secret created with the name specified on the `connInfoSecretTarget` field.

```bash
$ kubectl describe secret pg-connection 
Name:         pg-connection
Namespace:    default
Labels:       app=pg-sample
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

You can use [jq](https://github.com/stedolan/jq) to quickly decode the Secret:
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

Let's test the PostgreSQL connection from a Kubernetes workload by deploying a Pod running a `psql` command. Create a file named `pod-psql.yaml`
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

It will run once and stop, due the `restartPolicy: Never` flag. Let's inspect its log:
```bash
$ kubectl logs psql-test-connection
                                           version                                           
---------------------------------------------------------------------------------------------
 PostgreSQL 11.12 on x86_64-pc-linux-gnu, compiled by gcc, a 68c5366192 p 6b9244f01a, 64-bit
(1 row)
```

We were able to connect to the PostgreSQL and execute the `SELECT version();` query. Cool, right!?

## Create a PostgreSQL Database
The next Kubernetes resource we will explore is the `Database`. It allows you to create a logical database within the PostgreSQL instance.

Create the `pg-database-sample.yaml` file with the following content:
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

Now you can connect to the `pg-database-sample` using the same credentials stored in the `pg-connection` Secret.

## Create a PostgreSQL User
Aiven has a concept called Service User, allowing us to create users for different services. Let's create one for the PostgreSQL instance.

Create a file named `pg-service-user.yaml`
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

  project: dev-advocates
  serviceName: pg-sample
```

Apply the configuration with:
```bash
$ kubectl apply -f pg-service-user.yaml
```

The `ServiceUser` resource also generates a Secret with connection information, in this case named `pg-service-user-connection`:
```bash
$ kubectl get secret pg-service-user-connection -o json | jq '.data | map_values(@base64d)'

{
  "PASSWORD": "<secret-password>",
  "USERNAME": "pg-service-user"
}
```

You can go ahead and connect to the PostgreSQL instance using the credentials above and the host information from the `pg-connection` Secret.

## Create a PostgreSQL Connection Pool
Connection pooling allows ou to maintain very large numbers of connections to a database while minimizing the consumption of server resources. See more information [here](https://help.aiven.io/en/articles/964730-postgresql-connection-pooling).

Under the hood, PostgreSQL for Aiven uses PGBouncer. Let's create one with the `ConnectionPool` resource combining the previously created `Database` and `ServiceUser`.

Create a new file named `pg-connection-pool.yaml` with the following content:
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

As the previous examples, the `ConnectionPool` will generate a Secret with the connection info using the name from the `connInfoSecretTarget.Name` field:
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
