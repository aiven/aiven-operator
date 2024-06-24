---
title: "ClickhouseDatabase"
---

## Usage example

??? example 
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: ClickhouseDatabase
    metadata:
      name: my-db
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: my-aiven-project
      serviceName: my-clickhouse
      databaseName: my-db
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

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`databaseName`](#spec.databaseName-property){: name='spec.databaseName-property'} (string, Immutable, MaxLength: 63). Specifies the Clickhouse database name. Defaults to `metadata.name` if omitted.
Note: `metadata.name` is ASCII-only. For UTF-8 names, use `spec.databaseName`, but ASCII is advised for compatibility.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
