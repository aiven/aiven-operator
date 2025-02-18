---
title: "ClickhouseRole"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: ClickhouseRole
metadata:
  name: my-role
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  serviceName: my-clickhouse
  role: my-role
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `ClickhouseRole`:

```shell
kubectl get clickhouseroles my-role
```

The output is similar to the following:
```shell
Name       Project             Service Name     Role       
my-role    my-aiven-project    my-clickhouse    my-role    
```

---

## ClickhouseRole {: #ClickhouseRole }

ClickhouseRole is the Schema for the clickhouseroles API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ClickhouseRole`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ClickhouseRoleSpec defines the desired state of ClickhouseRole. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ClickhouseRole`](#ClickhouseRole)._

ClickhouseRoleSpec defines the desired state of ClickhouseRole.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`role`](#spec.role-property){: name='spec.role-property'} (string, Immutable, MaxLength: 255). The role that is to be created.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
