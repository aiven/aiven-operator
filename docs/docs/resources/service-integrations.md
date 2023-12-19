---
title: "Service Integrations"
linkTitle: "Service Integrations"
weight: 60
---

Service Integrations provide additional functionality and features by connecting different Aiven services together.

See
our [Getting Started with Service Integrations guide](https://help.aiven.io/en/articles/1456441-getting-started-with-service-integrations)
for more information.

!!! note
    Before going through this guide, make sure you have a [Kubernetes cluster](../../installation/prerequisites/) with the [operator installed](../../installation/),
    and a [Kubernetes Secret with an Aiven authentication token](../../authentication/).

## Send Kafka logs to a Kafka Topic

This integration allows you to send Kafka service logs to a specific Kafka Topic.

First, let's create a Kafka service and a topic.

1\. Create a new file named `kafka-sample-topic.yaml` with the following content:

```yaml
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: kafka-sample
spec:
  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # outputs the Kafka connection on the `kafka-connection` Secret
  connInfoSecretTarget:
    name: kafka-auth

  # add your Project name here
  project: <your-project-name>

  # cloud provider and plan of your choice
  # you can check all of the possibilities here https://aiven.io/pricing
  cloudName: google-europe-west1
  plan: startup-2

  # general Aiven configuration
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

  # specific Kafka configuration
  userConfig:
    kafka_version: '2.7'

---

apiVersion: aiven.io/v1alpha1
kind: KafkaTopic
metadata:
  name: logs
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: <your-project-name>
  serviceName: kafka-sample

  # here we can specify how many partitions the topic should have
  partitions: 3
  # and the topic replication factor
  replication: 2

  # we also support various topic-specific configurations
  config:
    flush_ms: 100
```

2\. Create the resource on Kubernetes:

```shell
kubectl apply -f kafka-sample-topic.yaml
```

3\. Now, create a `ServiceIntegration` resource to send the Kafka logs to the created topic. In the same file, add the
   following YAML:

```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: service-integration-kafka-logs
spec:

  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  project: <your-project-name>

  # indicates the type of the integration
  integrationType: kafka_logs

  # we will send the logs to the same kafka-sample instance
  # the source and destination are the same
  sourceServiceName: kafka-sample
  destinationServiceName: kafka-sample

  # the topic name we will send to
  kafkaLogs:
    kafka_topic: logs
```

4\. Reapply the resource on Kubernetes:

```shell
kubectl apply -f kafka-sample-topic.yaml
```

5\. Let's check the created service integration:

```shell
kubectl get serviceintegrations.aiven.io service-integration-kafka-logs
```

The output is similar to the following:

```{ .shell .no-copy }
NAME                             PROJECT        TYPE         SOURCE SERVICE NAME   DESTINATION SERVICE NAME   SOURCE ENDPOINT ID   DESTINATION ENDPOINT ID
service-integration-kafka-logs   your-project   kafka_logs   kafka-sample          kafka-sample
```

Your Kafka service logs are now being streamed to the `logs` Kafka topic.
