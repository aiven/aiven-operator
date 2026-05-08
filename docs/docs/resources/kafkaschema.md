---
title: "KafkaSchema"
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
| [ServiceSchemaRegistrySubjectConfigPut](https://api.aiven.io/doc/#operation/ServiceSchemaRegistrySubjectConfigPut) | `service:data:write` |
| [ServiceSchemaRegistrySubjectDelete](https://api.aiven.io/doc/#operation/ServiceSchemaRegistrySubjectDelete) | `service:data:write` |
| [ServiceSchemaRegistrySubjectVersionGet](https://api.aiven.io/doc/#operation/ServiceSchemaRegistrySubjectVersionGet) | `service:data:write` |
| [ServiceSchemaRegistrySubjectVersionPost](https://api.aiven.io/doc/#operation/ServiceSchemaRegistrySubjectVersionPost) | `service:data:write` |
| [ServiceSchemaRegistrySubjectVersionsGet](https://api.aiven.io/doc/#operation/ServiceSchemaRegistrySubjectVersionsGet) | `service:data:write` |

## Usage examples

	
=== "with_explicit_refs"

    ```yaml linenums="1"
    apiVersion: aiven.io/v1alpha1
    kind: KafkaSchema
    metadata:
      name: order-event-pinned
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: my-aiven-project
      serviceName: my-kafka
      subjectName: com.example.OrderEvent
      schemaType: PROTOBUF
      schema: |
        syntax = "proto3";
        import "common/money.proto";
        message OrderEvent {
          common.Money total = 1;
        }
      references:
        # Pin the referenced subject and version explicitly. Use this when the
        # referenced schema is managed outside this operator, or when you want
        # the dependent to keep pointing at a specific version regardless of
        # what the referent advances to.
        #
        # If the referent IS another KafkaSchema in the same namespace, prefer
        # the kafkaSchemaRef variant — see the with_refs example. That form
        # auto-propagates new versions instead of requiring manual updates here.
        - name: common/money.proto
          subject: common.Money
          version: 1
    ```

	
=== "with_refs"

    ```yaml linenums="1"
    apiVersion: aiven.io/v1alpha1
    kind: KafkaSchema
    metadata:
      name: money-type
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: my-aiven-project
      serviceName: my-kafka
      subjectName: common.Money
      schemaType: PROTOBUF
      schema: |
        syntax = "proto3";
        package common;
        message Money {
          string currency = 1;
          int64 amount_cents = 2;
        }
    ---
    apiVersion: aiven.io/v1alpha1
    kind: KafkaSchema
    metadata:
      name: order-event
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: my-aiven-project
      serviceName: my-kafka
      subjectName: com.example.OrderEvent
      schemaType: PROTOBUF
      schema: |
        syntax = "proto3";
        import "common/money.proto";
        message OrderEvent {
          common.Money total = 1;
        }
      references:
        # Resolve subject and version from another KafkaSchema in the same namespace.
        # No need to hard-code the version - dependents pick up new versions automatically.
        - name: common/money.proto
          kafkaSchemaRef:
            name: money-type
    ```

	
=== "example"

    ```yaml linenums="1"
    apiVersion: aiven.io/v1alpha1
    kind: KafkaSchema
    metadata:
      name: my-schema
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: my-aiven-project
      serviceName: my-kafka
      subjectName: mny-subject
      compatibilityLevel: BACKWARD
      schema: |
        {
            "doc": "example_doc",
            "fields": [{
                "default": 5,
                "doc": "field_doc",
                "name": "field_name",
                "namespace": "field_namespace",
                "type": "int"
            }],
            "name": "example_name",
            "namespace": "example_namespace",
            "type": "record"
        }
    ```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `KafkaSchema`:

```shell
kubectl get kafkaschemas order-event-pinned
```

The output is similar to the following:
```shell
Name                  Service Name    Project             Subject                   Version      
order-event-pinned    my-kafka        my-aiven-project    com.example.OrderEvent    <version>    
```

---

## KafkaSchema {: #KafkaSchema }

KafkaSchema is the Schema for the kafkaschemas API.

Self-references (A -> A) are blocked at admission; transitive cycles
(A -> B -> A) are not detected at admission time.

Deletion: the operator performs a soft delete followed by a hard delete on
the subject. The subject disappears from the registry's listing, re-applying a KafkaSchema with the same subjectName
after deletion starts a brand-new subject at version 1.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `KafkaSchema`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaSchemaSpec defines the desired state of KafkaSchema. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`KafkaSchema`](#KafkaSchema)._

KafkaSchemaSpec defines the desired state of KafkaSchema.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`schema`](#spec.schema-property){: name='spec.schema-property'} (string). Kafka Schema definition. Format depends on schemaType (AVRO/JSON/PROTOBUF).
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.
- [`subjectName`](#spec.subjectName-property){: name='spec.subjectName-property'} (string, Immutable). Kafka Schema Subject name.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`compatibilityLevel`](#spec.compatibilityLevel-property){: name='spec.compatibilityLevel-property'} (string, Enum: `BACKWARD`, `BACKWARD_TRANSITIVE`, `FORWARD`, `FORWARD_TRANSITIVE`, `FULL`, `FULL_TRANSITIVE`, `NONE`). Kafka Schemas compatibility level.
- [`references`](#spec.references-property){: name='spec.references-property'} (array of objects, MaxItems: 100). Schema references for Protobuf or JSON schemas that import other schemas.
    References must form a directed acyclic graph (DAG); cycles are not allowed. See below for [nested schema](#spec.references).
- [`schemaType`](#spec.schemaType-property){: name='spec.schemaType-property'} (string, Enum: `AVRO`, `JSON`, `PROTOBUF`, Immutable). Schema type.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## references {: #spec.references }

_Appears on [`spec`](#spec)._

SchemaReference is a reference to another schema in the registry.
Exactly one of {subject+version} or kafkaSchemaRef must be set.

**Required**

- [`name`](#spec.references.name-property){: name='spec.references.name-property'} (string, MinLength: 1, MaxLength: 512). Name used to reference the schema (e.g., the import path in Protobuf).

**Optional**

- [`kafkaSchemaRef`](#spec.references.kafkaSchemaRef-property){: name='spec.references.kafkaSchemaRef-property'} (object). Reference to another KafkaSchema resource in the same namespace.
    Mutually exclusive with subject/version.

    Cleanup order matters: delete the dependent before the referent. See below for [nested schema](#spec.references.kafkaSchemaRef).
- [`subject`](#spec.references.subject-property){: name='spec.references.subject-property'} (string, MinLength: 1, MaxLength: 512). Subject name of the referenced schema in the registry. Mutually exclusive with kafkaSchemaRef.
- [`version`](#spec.references.version-property){: name='spec.references.version-property'} (integer, Minimum: 1). Version of the referenced schema. Mutually exclusive with kafkaSchemaRef.

### kafkaSchemaRef {: #spec.references.kafkaSchemaRef }

_Appears on [`spec.references`](#spec.references)._

Reference to another KafkaSchema resource in the same namespace.
Mutually exclusive with subject/version.

Cleanup order matters: delete the dependent before the referent.

**Required**

- [`name`](#spec.references.kafkaSchemaRef.name-property){: name='spec.references.kafkaSchemaRef.name-property'} (string, MinLength: 1, MaxLength: 253). Name of the KafkaSchema resource in the same namespace.

