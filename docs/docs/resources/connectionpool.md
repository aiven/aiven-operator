---
title: "ConnectionPool"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: ConnectionPool
metadata:
  name: my-connection-pool
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: aiven-project-name
  serviceName: my-pg
  databaseName: my-database
  username: my-service-user
  poolMode: transaction
  poolSize: 25

---

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
  name: my-database
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: aiven-project-name
  serviceName: my-pg

---

apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: my-service-user
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: aiven-project-name
  serviceName: my-pg
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `ConnectionPool`:

```shell
kubectl get connectionpools my-connection-pool
```

The output is similar to the following:
```shell
Name                  Service Name    Project               Database       Username           Pool Size    Pool Mode      
my-connection-pool    my-pg           aiven-project-name    my-database    my-service-user    25           transaction    
```

---

## ConnectionPool {: #ConnectionPool }

ConnectionPool is the Schema for the connectionpools API.

!!! Info "Exposes secret keys"

    `CONNECTIONPOOL_NAME`, `CONNECTIONPOOL_HOST`, `CONNECTIONPOOL_PORT`, `CONNECTIONPOOL_DATABASE`, `CONNECTIONPOOL_USER`, `CONNECTIONPOOL_PASSWORD`, `CONNECTIONPOOL_SSLMODE`, `CONNECTIONPOOL_DATABASE_URI`, `CONNECTIONPOOL_CA_CERT`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ConnectionPool`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ConnectionPoolSpec defines the desired state of ConnectionPool. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ConnectionPool`](#ConnectionPool)._

ConnectionPoolSpec defines the desired state of ConnectionPool.

**Required**

- [`databaseName`](#spec.databaseName-property){: name='spec.databaseName-property'} (string, MaxLength: 40). Name of the database the pool connects to.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.
- [`username`](#spec.username-property){: name='spec.username-property'} (string, MaxLength: 64). Name of the service user used to connect to the database.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Secret configuration. See below for [nested schema](#spec.connInfoSecretTarget).
- [`connInfoSecretTargetDisabled`](#spec.connInfoSecretTargetDisabled-property){: name='spec.connInfoSecretTargetDisabled-property'} (boolean, Immutable). When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
- [`poolMode`](#spec.poolMode-property){: name='spec.poolMode-property'} (string, Enum: `session`, `transaction`, `statement`). Mode the pool operates in (session, transaction, statement).
- [`poolSize`](#spec.poolSize-property){: name='spec.poolSize-property'} (integer). Number of connections the pool may create towards the backend server.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

_Appears on [`spec`](#spec)._

Secret configuration.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string, Immutable). Name of the secret resource to be created. By default, it is equal to the resource name.

**Optional**

- [`annotations`](#spec.connInfoSecretTarget.annotations-property){: name='spec.connInfoSecretTarget.annotations-property'} (object, AdditionalProperties: string). Annotations added to the secret.
- [`labels`](#spec.connInfoSecretTarget.labels-property){: name='spec.connInfoSecretTarget.labels-property'} (object, AdditionalProperties: string). Labels added to the secret.
- [`prefix`](#spec.connInfoSecretTarget.prefix-property){: name='spec.connInfoSecretTarget.prefix-property'} (string). Prefix for the secret's keys.
Added "as is" without any transformations.
By default, is equal to the kind name in uppercase + underscore, e.g. `KAFKA_`, `REDIS_`, etc.
