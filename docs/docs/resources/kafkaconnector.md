---
title: "KafkaConnector"
---

## KafkaConnector {: #KafkaConnector }

KafkaConnector is the Schema for the kafkaconnectors API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `KafkaConnector`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaConnectorSpec defines the desired state of KafkaConnector. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`KafkaConnector`](#KafkaConnector)._

KafkaConnectorSpec defines the desired state of KafkaConnector.

**Required**

- [`connectorClass`](#spec.connectorClass-property){: name='spec.connectorClass-property'} (string, MaxLength: 1024). The Java class of the connector.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object, AdditionalProperties: string). The connector specific configuration
To build config values from secret the template function `{{ fromSecret "name" "key" }}`
is provided when interpreting the keys.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
