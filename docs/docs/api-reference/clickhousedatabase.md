---
title: "ClickhouseDatabase"
---

## Usage example

```yaml
apiVersion: aiven.io/v1alpha1
kind: ClickhouseDatabase
metadata:
  name: my-db
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: aiven-project-name
  serviceName: my-clickhouse
```

## ClickhouseDatabase {: #ClickhouseDatabase }

ClickhouseDatabase is the Schema for the databases API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ClickhouseDatabase`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ClickhouseDatabaseSpec defines the desired state of ClickhouseDatabase. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ClickhouseDatabase`](#ClickhouseDatabase)._

ClickhouseDatabaseSpec defines the desired state of ClickhouseDatabase.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, MaxLength: 63, Format: `^[a-zA-Z0-9_-]*$`). Project to link the database to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, MaxLength: 63). Clickhouse service to link the database to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). It is a Kubernetes side deletion protections, which prevents the database from being deleted by Kubernetes. It is recommended to enable this for any production databases containing critical data.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
