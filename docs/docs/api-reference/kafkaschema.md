---
title: "KafkaSchema"
---

## Usage example

```yaml
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

## KafkaSchema {: #KafkaSchema }

KafkaSchema is the Schema for the kafkaschemas API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `KafkaSchema`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaSchemaSpec defines the desired state of KafkaSchema. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`KafkaSchema`](#KafkaSchema)._

KafkaSchemaSpec defines the desired state of KafkaSchema.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, MaxLength: 63, Format: `^[a-zA-Z0-9_-]*$`). Project to link the Kafka Schema to.
- [`schema`](#spec.schema-property){: name='spec.schema-property'} (string). Kafka Schema configuration should be a valid Avro Schema JSON format.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, MaxLength: 63). Service to link the Kafka Schema to.
- [`subjectName`](#spec.subjectName-property){: name='spec.subjectName-property'} (string, MaxLength: 63). Kafka Schema Subject name.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`compatibilityLevel`](#spec.compatibilityLevel-property){: name='spec.compatibilityLevel-property'} (string, Enum: `BACKWARD`, `BACKWARD_TRANSITIVE`, `FORWARD`, `FORWARD_TRANSITIVE`, `FULL`, `FULL_TRANSITIVE`, `NONE`). Kafka Schemas compatibility level.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

