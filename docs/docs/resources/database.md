---
title: "Database"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

??? example 
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: PostgreSQL
    metadata:
      name: my-pg
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: startup-4
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: Database
    metadata:
      name: my-db
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      serviceName: my-pg
    
      # Database name will default to the value of `metadata.name` if `databaseName` is not specified.
      # Use the `databaseName` field if the desired database name contains underscores.
      databaseName: my_db_name
    
      lcCtype: en_US.UTF-8
      lcCollate: en_US.UTF-8
    ```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `Database`:

```shell
kubectl get databases my-db
```

The output is similar to the following:
```shell
Name     Project               Service Name    
my-db    aiven-project-name    my-pg           
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

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`databaseName`](#spec.databaseName-property){: name='spec.databaseName-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_][a-zA-Z0-9_-]{0,39}$`, MaxLength: 40). DatabaseName is the name of the database to be created.
- [`lcCollate`](#spec.lcCollate-property){: name='spec.lcCollate-property'} (string, Immutable, MaxLength: 128). Default string sort order (LC_COLLATE) of the database. Default value: en_US.UTF-8.
- [`lcCtype`](#spec.lcCtype-property){: name='spec.lcCtype-property'} (string, Immutable, MaxLength: 128). Default character classification (LC_CTYPE) of the database. Default value: en_US.UTF-8.
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). It is a Kubernetes side deletion protections, which prevents the database
from being deleted by Kubernetes. It is recommended to enable this for any production
databases containing critical data.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
