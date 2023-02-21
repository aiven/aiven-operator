---
title: "KafkaSchema"
---

## Schema {: #Schema }

KafkaSchema is the Schema for the kafkaschemas API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Must be equal to `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Must be equal to `KafkaSchema`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaSchemaSpec defines the desired state of KafkaSchema. See below for [nested schema](#spec).

## spec {: #spec }

KafkaSchemaSpec defines the desired state of KafkaSchema.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, MaxLength: 63). Project to link the Kafka Schema to.
- [`schema`](#spec.schema-property){: name='spec.schema-property'} (string). Kafka Schema configuration should be a valid Avro Schema JSON format.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, MaxLength: 63). Service to link the Kafka Schema to.
- [`subjectName`](#spec.subjectName-property){: name='spec.subjectName-property'} (string, MaxLength: 63). Kafka Schema Subject name.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`compatibilityLevel`](#spec.compatibilityLevel-property){: name='spec.compatibilityLevel-property'} (string, Enum: `BACKWARD`, `BACKWARD_TRANSITIVE`, `FORWARD`, `FORWARD_TRANSITIVE`, `FULL`, `FULL_TRANSITIVE`, `NONE`). Kafka Schemas compatibility level.

## authSecretRef {: #spec.authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

