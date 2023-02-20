---
title: "KafkaACL"
---

## Schema {: #Schema }

KafkaACL is the Schema for the kafkaacls API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Must be equal to `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Must be equal to `KafkaACL`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaACLSpec defines the desired state of KafkaACL. See below for [nested schema](#spec).

## spec {: #spec }

KafkaACLSpec defines the desired state of KafkaACL.

**Required**

- [`permission`](#spec.permission-property){: name='spec.permission-property'} (string, Enum: `admin`, `read`, `readwrite`, `write`). Kafka permission to grant (admin, read, readwrite, write).
- [`project`](#spec.project-property){: name='spec.project-property'} (string, MaxLength: 63). Project to link the Kafka ACL to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, MaxLength: 63). Service to link the Kafka ACL to.
- [`topic`](#spec.topic-property){: name='spec.topic-property'} (string). Topic name pattern for the ACL entry.
- [`username`](#spec.username-property){: name='spec.username-property'} (string). Username pattern for the ACL entry.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).

## authSecretRef {: #spec.authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

