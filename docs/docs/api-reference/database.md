---
title: "Database"
---

## Usage example

```yaml
apiVersion: aiven.io/v1alpha1
kind: Database
metadata:
  name: my-db
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: aiven-project-name
  serviceName: google-europe-west1

  lcCtype: en_US.UTF-8
  lcCollate: en_US.UTF-8
```

## Database {: #Database }

Database is the Schema for the databases API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `Database`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). DatabaseSpec defines the desired state of Database. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`Database`](#Database)._

DatabaseSpec defines the desired state of Database.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, MaxLength: 63, Format: `^[a-zA-Z0-9_-]*$`). Project to link the database to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, MaxLength: 63). PostgreSQL service to link the database to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`lcCollate`](#spec.lcCollate-property){: name='spec.lcCollate-property'} (string, MaxLength: 128). Default string sort order (LC_COLLATE) of the database. Default value: en_US.UTF-8.
- [`lcCtype`](#spec.lcCtype-property){: name='spec.lcCtype-property'} (string, MaxLength: 128). Default character classification (LC_CTYPE) of the database. Default value: en_US.UTF-8.
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). It is a Kubernetes side deletion protections, which prevents the database from being deleted by Kubernetes. It is recommended to enable this for any production databases containing critical data.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

