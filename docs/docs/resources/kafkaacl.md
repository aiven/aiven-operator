---
title: "KafkaACL"
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
| [ServiceKafkaAclAdd](https://api.aiven.io/doc/#operation/ServiceKafkaAclAdd) | `service:data:write` |
| [ServiceKafkaAclDelete](https://api.aiven.io/doc/#operation/ServiceKafkaAclDelete) | `service:data:write` |
| [ServiceKafkaAclList](https://api.aiven.io/doc/#operation/ServiceKafkaAclList) | `service:data:write` |

## Usage example

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: KafkaACL
metadata:
  name: my-kafka-acl
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  serviceName: my-kafka
  topic: my-topic
  username: my-user
  permission: admin
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `KafkaACL`:

```shell
kubectl get kafkaacls my-kafka-acl
```

The output is similar to the following:
```shell
Name            Service Name    Project             Username    Permission    Topic       
my-kafka-acl    my-kafka        my-aiven-project    my-user     admin         my-topic    
```

---

## KafkaACL {: #KafkaACL }

KafkaACL is the Schema for the kafkaacls API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `KafkaACL`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaACLSpec defines the desired state of KafkaACL. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`KafkaACL`](#KafkaACL)._

KafkaACLSpec defines the desired state of KafkaACL.

**Required**

- [`permission`](#spec.permission-property){: name='spec.permission-property'} (string, Enum: `admin`, `read`, `readwrite`, `write`). Kafka permission to grant (admin, read, readwrite, write).
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.
- [`topic`](#spec.topic-property){: name='spec.topic-property'} (string). Topic name pattern for the ACL entry.
- [`username`](#spec.username-property){: name='spec.username-property'} (string). Username pattern for the ACL entry.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
