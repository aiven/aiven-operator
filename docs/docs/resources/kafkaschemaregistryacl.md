---
title: "KafkaSchemaRegistryACL"
---

## Prerequisites
	
* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

### Required permissions

To create and manage this resource, you must have the appropriate [roles or permissions](https://aiven.io/docs/platform/concepts/permissions).
See the [Aiven documentation](https://aiven.io/docs/platform/howto/manage-permissions) for details on managing permissions.

This resource uses the following API operations, and for each operation, _any_ of the listed permissions is sufficient:

| Operation | Permissions  |
| ----------- | ----------- |
| [ServiceGet](https://api.aiven.io/doc/#operation/ServiceGet) | `project:services:read` |
| [ServiceSchemaRegistryAclAdd](https://api.aiven.io/doc/#operation/ServiceSchemaRegistryAclAdd) | `service:data:write` |
| [ServiceSchemaRegistryAclDelete](https://api.aiven.io/doc/#operation/ServiceSchemaRegistryAclDelete) | `service:data:write` |
| [ServiceSchemaRegistryAclList](https://api.aiven.io/doc/#operation/ServiceSchemaRegistryAclList) | `service:data:write` |

## Usage example

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: KafkaSchemaRegistryACL
metadata:
  name: my-kafka-schema-registry-acl
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: aiven-project-name
  serviceName: my-kafka
  resource: Subject:my-topic
  username: my-user
  permission: schema_registry_read
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `KafkaSchemaRegistryACL`:

```shell
kubectl get kafkaschemaregistryacls my-kafka-schema-registry-acl
```

The output is similar to the following:
```shell
Name                            Project               Service Name    Resource            Username    State      
my-kafka-schema-registry-acl    aiven-project-name    my-kafka        Subject:my-topic    my-user     RUNNING    
```

---

## KafkaSchemaRegistryACL {: #KafkaSchemaRegistryACL }

KafkaSchemaRegistryACL is the Schema for the kafkaschemaregistryacls API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `KafkaSchemaRegistryACL`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaSchemaRegistryACLSpec defines the desired state of KafkaSchemaRegistryACL. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`KafkaSchemaRegistryACL`](#KafkaSchemaRegistryACL)._

KafkaSchemaRegistryACLSpec defines the desired state of KafkaSchemaRegistryACL.

**Required**

- [`permission`](#spec.permission-property){: name='spec.permission-property'} (string, Enum: `schema_registry_read`, `schema_registry_write`, Immutable).
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`resource`](#spec.resource-property){: name='spec.resource-property'} (string, Immutable, MaxLength: 249). Resource name pattern for the Schema Registry ACL entry.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.
- [`username`](#spec.username-property){: name='spec.username-property'} (string, Immutable, MaxLength: 64). Username pattern for the ACL entry.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
