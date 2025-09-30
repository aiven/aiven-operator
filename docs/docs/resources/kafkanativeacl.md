---
title: "KafkaNativeACL"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: my-kafka
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: kafka-secret

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: startup-4

  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

---

apiVersion: aiven.io/v1alpha1
kind: KafkaNativeACL
metadata:
  name: my-kafka-native-acl
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  serviceName: my-kafka
  host: my-host
  operation: Create
  patternType: LITERAL
  permissionType: ALLOW
  principal: User:alice
  resourceName: my-kafka-topic
  resourceType: Topic
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `KafkaNativeACL`:

```shell
kubectl get kafkanativeacls my-kafka-native-acl
```

The output is similar to the following:
```shell
Name                   Service Name    Project             Host       Operation    PatternType    PermissionType    
my-kafka-native-acl    my-kafka        my-aiven-project    my-host    Create       LITERAL        ALLOW             
```

---

## KafkaNativeACL {: #KafkaNativeACL }

KafkaNativeACL
Creates and manages Kafka-native access control lists (ACLs) for an Aiven for Apache Kafka® service.
ACLs control access to Kafka topics, consumer groups, clusters, and Schema Registry.
Kafka-native ACLs provide advanced resource-level access control with fine-grained permissions, including ALLOW and DENY rules.
For simplified topic-level control, you can use [KafkaACL](./kafkaacl.md).

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `KafkaNativeACL`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object, Immutable). KafkaNativeACLSpec defines the desired state of KafkaNativeACL. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`KafkaNativeACL`](#KafkaNativeACL)._

KafkaNativeACLSpec defines the desired state of KafkaNativeACL.

**Required**

- [`operation`](#spec.operation-property){: name='spec.operation-property'} (string, Enum: `All`, `Alter`, `AlterConfigs`, `ClusterAction`, `Create`, `CreateTokens`, `Delete`, `Describe`, `DescribeConfigs`, `DescribeTokens`, `IdempotentWrite`, `Read`, `Write`). Kafka ACL operation represents an operation which an ACL grants or denies permission to perform.
- [`patternType`](#spec.patternType-property){: name='spec.patternType-property'} (string, Enum: `LITERAL`, `PREFIXED`). Kafka ACL pattern type of resource name.
- [`permissionType`](#spec.permissionType-property){: name='spec.permissionType-property'} (string, Enum: `ALLOW`, `DENY`). Kafka ACL permission type.
- [`principal`](#spec.principal-property){: name='spec.principal-property'} (string, MaxLength: 256). Principal is in `PrincipalType:name` format.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`resourceName`](#spec.resourceName-property){: name='spec.resourceName-property'} (string, MaxLength: 256). Resource pattern used to match specified resources.
- [`resourceType`](#spec.resourceType-property){: name='spec.resourceType-property'} (string, Enum: `Cluster`, `DelegationToken`, `Group`, `Topic`, `TransactionalId`, `User`). Kafka ACL resource type represents a type of resource which an ACL can be applied to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`host`](#spec.host-property){: name='spec.host-property'} (string, MaxLength: 256, Default value: `*`). The host or `*` for all hosts.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
