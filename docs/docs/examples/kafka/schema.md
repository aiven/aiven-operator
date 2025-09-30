---
title: "Kafka Schema"
linkTitle: "Kafka Schema"
weight: 40
---

## Create a `KafkaSchema`

Aiven develops and maintain [Karapace](https://github.com/aiven/karapace), an open source implementation of Kafka REST
and schema registry. Is available out of the box for our managed Kafka service.

> The schema registry address and authentication is the same as the Kafka broker, the only different is the usage of the port 13044.

First, let's create an Aiven for Apache Kafka service.

1\. Create a file named `kafka-sample-schema.yaml` and add the content below:

```yaml
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: kafka-sample-schema
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: kafka-auth

  project: PROJECT_NAME
  cloudName: google-europe-west1
  plan: startup-4
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

  userConfig:
    kafka_version: "2.7"

    # this flag enables the Schema registry
    schema_registry: true
```

2\. Apply the changes with the following command:

```shell
kubectl apply -f kafka-schema.yaml
```

Now, let's create the schema itself.

1\. Create a new file named `kafka-sample-schema.yaml` and add the YAML content below:

```yaml
apiVersion: aiven.io/v1alpha1
kind: KafkaSchema
metadata:
  name: kafka-schema
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: PROJECT_NAME
  serviceName: kafka-sample-schema

  # the name of the Schema
  subjectName: MySchema

  # the schema itself, in JSON format
  schema: |
    {
      "type": "record",
      "name": "MySchema",
      "fields": [
        {
          "name": "field",
          "type": "string"
        }
      ]
    }

  # sets the schema compatibility level
  compatibilityLevel: BACKWARD
```

2\. Create the schema with the command:

```shell
kubectl apply -f kafka-schema.yaml
```

3\. Review the resource you created with the following command:

```shell
kubectl get kafkaschemas.aiven.io kafka-schema
```

The output is similar to the following:

```{ .shell .no-copy }
NAME           SERVICE NAME   PROJECT          SUBJECT    COMPATIBILITY LEVEL   VERSION
kafka-schema   kafka-sample   PROJECT_NAME   MySchema   BACKWARD              1
```

Now you can follow the instructions to [use a schema registry in Java](https://aiven.io/docs/products/kafka/howto/schema-registry) on how to use the schema created.
