---
title: "KafkaTopic"
---

## Usage example

```yaml
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
  topicName: my-kafka-topic

  replication: 2
  partitions: 1

  config:
    min_cleanable_dirty_ratio: 0.2
```

## KafkaTopic {: #KafkaTopic }

KafkaTopic is the Schema for the kafkatopics API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `KafkaTopic`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaTopicSpec defines the desired state of KafkaTopic. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`KafkaTopic`](#KafkaTopic)._

KafkaTopicSpec defines the desired state of KafkaTopic.

**Required**

- [`partitions`](#spec.partitions-property){: name='spec.partitions-property'} (integer, Minimum: 1, Maximum: 1000000). Number of partitions to create in the topic.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, MaxLength: 63, Format: `^[a-zA-Z0-9_-]*$`). Target project.
- [`replication`](#spec.replication-property){: name='spec.replication-property'} (integer, Minimum: 2). Replication factor for the topic.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, MaxLength: 63). Service name.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`config`](#spec.config-property){: name='spec.config-property'} (object). Kafka topic configuration. See below for [nested schema](#spec.config).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (array of objects). Kafka topic tags. See below for [nested schema](#spec.tags).
- [`termination_protection`](#spec.termination_protection-property){: name='spec.termination_protection-property'} (boolean). It is a Kubernetes side deletion protections, which prevents the kafka topic from being deleted by Kubernetes. It is recommended to enable this for any production databases containing critical data.
- [`topicName`](#spec.topicName-property){: name='spec.topicName-property'} (string, Immutable, MinLength: 1, MaxLength: 249). Topic name. If provided, is used instead of metadata.name. This field supports additional characters, has a longer length, and will replace metadata.name in future releases.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## config {: #spec.config }

_Appears on [`spec`](#spec)._

Kafka topic configuration.

**Optional**

- [`cleanup_policy`](#spec.config.cleanup_policy-property){: name='spec.config.cleanup_policy-property'} (string). cleanup.policy value.
- [`compression_type`](#spec.config.compression_type-property){: name='spec.config.compression_type-property'} (string). compression.type value.
- [`delete_retention_ms`](#spec.config.delete_retention_ms-property){: name='spec.config.delete_retention_ms-property'} (integer). delete.retention.ms value.
- [`file_delete_delay_ms`](#spec.config.file_delete_delay_ms-property){: name='spec.config.file_delete_delay_ms-property'} (integer). file.delete.delay.ms value.
- [`flush_messages`](#spec.config.flush_messages-property){: name='spec.config.flush_messages-property'} (integer). flush.messages value.
- [`flush_ms`](#spec.config.flush_ms-property){: name='spec.config.flush_ms-property'} (integer). flush.ms value.
- [`index_interval_bytes`](#spec.config.index_interval_bytes-property){: name='spec.config.index_interval_bytes-property'} (integer). index.interval.bytes value.
- [`max_compaction_lag_ms`](#spec.config.max_compaction_lag_ms-property){: name='spec.config.max_compaction_lag_ms-property'} (integer). max.compaction.lag.ms value.
- [`max_message_bytes`](#spec.config.max_message_bytes-property){: name='spec.config.max_message_bytes-property'} (integer). max.message.bytes value.
- [`message_downconversion_enable`](#spec.config.message_downconversion_enable-property){: name='spec.config.message_downconversion_enable-property'} (boolean). message.downconversion.enable value.
- [`message_format_version`](#spec.config.message_format_version-property){: name='spec.config.message_format_version-property'} (string). message.format.version value.
- [`message_timestamp_difference_max_ms`](#spec.config.message_timestamp_difference_max_ms-property){: name='spec.config.message_timestamp_difference_max_ms-property'} (integer). message.timestamp.difference.max.ms value.
- [`message_timestamp_type`](#spec.config.message_timestamp_type-property){: name='spec.config.message_timestamp_type-property'} (string). message.timestamp.type value.
- [`min_cleanable_dirty_ratio`](#spec.config.min_cleanable_dirty_ratio-property){: name='spec.config.min_cleanable_dirty_ratio-property'} (number). min.cleanable.dirty.ratio value.
- [`min_compaction_lag_ms`](#spec.config.min_compaction_lag_ms-property){: name='spec.config.min_compaction_lag_ms-property'} (integer). min.compaction.lag.ms value.
- [`min_insync_replicas`](#spec.config.min_insync_replicas-property){: name='spec.config.min_insync_replicas-property'} (integer). min.insync.replicas value.
- [`preallocate`](#spec.config.preallocate-property){: name='spec.config.preallocate-property'} (boolean). preallocate value.
- [`retention_bytes`](#spec.config.retention_bytes-property){: name='spec.config.retention_bytes-property'} (integer). retention.bytes value.
- [`retention_ms`](#spec.config.retention_ms-property){: name='spec.config.retention_ms-property'} (integer). retention.ms value.
- [`segment_bytes`](#spec.config.segment_bytes-property){: name='spec.config.segment_bytes-property'} (integer). segment.bytes value.
- [`segment_index_bytes`](#spec.config.segment_index_bytes-property){: name='spec.config.segment_index_bytes-property'} (integer). segment.index.bytes value.
- [`segment_jitter_ms`](#spec.config.segment_jitter_ms-property){: name='spec.config.segment_jitter_ms-property'} (integer). segment.jitter.ms value.
- [`segment_ms`](#spec.config.segment_ms-property){: name='spec.config.segment_ms-property'} (integer). segment.ms value.

## tags {: #spec.tags }

_Appears on [`spec`](#spec)._

Kafka topic tags.

**Required**

- [`key`](#spec.tags.key-property){: name='spec.tags.key-property'} (string, MinLength: 1, MaxLength: 64, Format: `^[a-zA-Z0-9_-]*$`).

**Optional**

- [`value`](#spec.tags.value-property){: name='spec.tags.value-property'} (string, MaxLength: 256, Format: `^[a-zA-Z0-9_-]*$`).
