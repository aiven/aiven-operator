---
title: "PostgreSQL"
linkTitle: "PostgreSQL"
weight: 20 
---

PostgreSQL is an open source, relational database. It's ideal for organisations that need a well organised tabular datastore. On top of the strict table and columns formats, PostgreSQL also offers solutions for nested datasets with the native `jsonb` format and advanced set of extensions including [PostGIS](https://postgis.net/), a spatial database extender for location queries. Aiven for PostgreSQL is the perfect fit for your relational data.

With Aiven Kubernetes Operator, you can manage Aiven for PostgreSQL through the well defined Kubernetes API.

> Before going through this guide, make sure to have a [Kubernetes Cluster](../installation/prerequisites/) with the [Operator installed](../installation/) and a [Kubernetes Secret with an Aiven authentication token](../authentication/).

## Create a PostgreSQL instance
Create a file named `aiven-pg.yaml` with the following content:
```yaml
apiVersion: aiven.io/v1alpha1
kind: PG
metadata:
  name: aiven-pg
spec:

  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # outputs the PostgreSQL connection on the `pg-connection` Secret
  connInfoSecretTarget:
    name: pg-connection

  # add your Project name here
  project: <your-project-name-here> 

  # cloud provider and plan of your choice
  # you can check all of the possibilities here https://aiven.io/pricing
  cloudName: google-europe-west1
  plan: hobbyist

  # general Aiven configuration
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

  # specific PostgreSQL configuration
  pgUserConfig:
    pg_version: '11'
```

Let's create the service by applying the configuration:
```bash
$ kubectl apply -f aiven-pg.yaml
```

Take a look at the resource created with the following:
```bash
$ kubectl get pgs.aiven.io aiven-pg

NAME        PROJECT         REGION                PLAN       STATE
aiven-pg    dev-advocates   google-europe-west1   hobbyist   RUNNING
```

The resource might stay in the `BUILDING` state for a couple of minutes, enough to grab a quick coffee! Once the state becomes `RUNNING`, we are ready to access it.

## Connection Information Secret 
For your convenience, we automatically store the PostgreSQL connection information on a Secret created with the name specified on the `connInfoSecretTarget` field.

```bash
$ kubectl describe secret pg-connection 
Name:         pg-connection
Namespace:    default
Labels:       app=aiven-pg
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
  "DATABASE_URI": "postgres://avnadmin:<secret-password>@aiven-pg-your-project.aivencloud.com:13039/defaultdb?sslmode=require",
  "PGDATABASE": "defaultdb",
  "PGHOST": "aiven-pg-your-project.aivencloud.com",
  "PGPASSWORD": "<secret-password>",
  "PGPORT": "13039",
  "PGSSLMODE": "require",
  "PGUSER": "avnadmin"
}
```

To test the PostgreSQL connection from a Kubernetes workload, you can deploy a Pod with a `psql` command. Create a file named `pod-psql.yaml`
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
      envFrom:
      - secretRef:
          name: pg-connection
```