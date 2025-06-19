---
title: "KafkaConnector"
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
  plan: business-4

  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

  userConfig:
    kafka_connect: true
    kafka:
      group_max_session_timeout_ms: 70000
      log_retention_bytes: 1000000000

---

apiVersion: aiven.io/v1alpha1
kind: KafkaTopic
metadata:
  name: kafka-topic
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  serviceName: my-kafka
  partitions: 3
  replication: 2

---

apiVersion: aiven.io/v1alpha1
kind: OpenSearch
metadata:
  name: my-os
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: os-secret

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: startup-4

  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

---

apiVersion: aiven.io/v1alpha1
kind: KafkaConnector
metadata:
  name: my-kafka-connect
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  serviceName: my-kafka
  connectorClass: io.aiven.kafka.connect.opensearch.OpensearchSinkConnector

  userConfig:
    topics:           my-kafka-topic
    type.name:        es-connector
    connection.url:   '{{ fromSecret "os-secret" "OPENSEARCH_URI" }}'
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `KafkaConnector`:

```shell
kubectl get kafkaconnectors my-kafka-connect
```

The output is similar to the following:
```shell
Name                Service Name    Project             Connector Class                                              State      Tasks Total            Tasks Running            
my-kafka-connect    my-kafka        my-aiven-project    io.aiven.kafka.connect.opensearch.OpensearchSinkConnector    RUNNING    <tasksStatus.total>    <tasksStatus.running>    
```

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
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object, AdditionalProperties: string). The connector-specific configuration
    To build config values from secret the template function `{{ fromSecret "name" "key" }}`
    is provided when interpreting the keys.
    Where "name" is the name of the secret and "key" is the key in the secret
    in the same namespace as the KafkaConnector.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
