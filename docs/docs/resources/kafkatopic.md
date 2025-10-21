---
title: "KafkaTopic"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

```yaml linenums="1"
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

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `KafkaTopic`:

```shell
kubectl get kafkatopics kafka-topic
```

The output is similar to the following:
```shell
Name           Service Name    Project             Partitions    Replication    
kafka-topic    my-kafka        my-aiven-project    1             2              
```

---

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
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`replication`](#spec.replication-property){: name='spec.replication-property'} (integer, Minimum: 2). Replication factor for the topic.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`config`](#spec.config-property){: name='spec.config-property'} (object). Kafka topic configuration. See below for [nested schema](#spec.config).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (array of objects). Kafka topic tags. See below for [nested schema](#spec.tags).
- [`termination_protection`](#spec.termination_protection-property){: name='spec.termination_protection-property'} (boolean). It is a Kubernetes side deletion protections, which prevents the kafka topic
    from being deleted by Kubernetes. It is recommended to enable this for any production
    databases containing critical data.
- [`topicName`](#spec.topicName-property){: name='spec.topicName-property'} (string, Immutable, MinLength: 1, MaxLength: 249). Topic name. If provided, is used instead of metadata.name.
    This field supports additional characters, has a longer length,
    and will replace metadata.name in future releases.

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

- [`cleanup_policy`](#spec.config.cleanup_policy-property){: name='spec.config.cleanup_policy-property'} (string). The retention policy to use on old segments. Possible values include `delete`, `compact`, or a comma-separated list of them. The default policy (`delete`) will discard old segments when their retention time or size limit has been reached. The `compact` setting will enable log compaction on the topic.
- [`compression_type`](#spec.config.compression_type-property){: name='spec.config.compression_type-property'} (string). Specify the final compression type for a given topic. This configuration accepts the standard compression codecs (`gzip`, `snappy`, `lz4`, `zstd`). It additionally accepts `uncompressed` which is equivalent to no compression; and `producer` which means retain the original compression codec set by the producer.
- [`delete_retention_ms`](#spec.config.delete_retention_ms-property){: name='spec.config.delete_retention_ms-property'} (integer). The amount of time to retain delete tombstone markers for log compacted topics. This setting also gives a bound on the time in which a consumer must complete a read if they begin from offset 0 to ensure that they get a valid snapshot of the final stage (otherwise delete tombstones may be collected before they complete their scan).
- [`diskless_enable`](#spec.config.diskless_enable-property){: name='spec.config.diskless_enable-property'} (boolean). Indicates whether diskless should be enabled.
- [`file_delete_delay_ms`](#spec.config.file_delete_delay_ms-property){: name='spec.config.file_delete_delay_ms-property'} (integer). The time to wait before deleting a file from the filesystem.
- [`flush_messages`](#spec.config.flush_messages-property){: name='spec.config.flush_messages-property'} (integer). This setting allows specifying an interval at which we will force an fsync of data written to the log. For example if this was set to 1 we would fsync after every message; if it were 5 we would fsync after every five messages. In general we recommend you not set this and use replication for durability and allow the operating system's background flush capabilities as it is more efficient.
- [`flush_ms`](#spec.config.flush_ms-property){: name='spec.config.flush_ms-property'} (integer). This setting allows specifying a time interval at which we will force an fsync of data written to the log. For example if this was set to 1000 we would fsync after 1000 ms had passed. In general we recommend you not set this and use replication for durability and allow the operating system's background flush capabilities as it is more efficient.
- [`index_interval_bytes`](#spec.config.index_interval_bytes-property){: name='spec.config.index_interval_bytes-property'} (integer). This setting controls how frequently Kafka adds an index entry to its offset index. The default setting ensures that we index a message roughly every 4096 bytes. More indexing allows reads to jump closer to the exact position in the log but makes the index larger. You probably don't need to change this.
- [`local_retention_bytes`](#spec.config.local_retention_bytes-property){: name='spec.config.local_retention_bytes-property'} (integer). This configuration controls the maximum bytes tiered storage will retain segment files locally before it will discard old log segments to free up space. If set to -2, the limit is equal to overall retention time. If set to -1, no limit is applied but it's possible only if overall retention is also -1.
- [`local_retention_ms`](#spec.config.local_retention_ms-property){: name='spec.config.local_retention_ms-property'} (integer). This configuration controls the maximum time tiered storage will retain segment files locally before it will discard old log segments to free up space. If set to -2, the time limit is equal to overall retention time. If set to -1, no time limit is applied but it's possible only if overall retention is also -1.
- [`max_compaction_lag_ms`](#spec.config.max_compaction_lag_ms-property){: name='spec.config.max_compaction_lag_ms-property'} (integer). The maximum time a message will remain ineligible for compaction in the log. Only applicable for logs that are being compacted.
- [`max_message_bytes`](#spec.config.max_message_bytes-property){: name='spec.config.max_message_bytes-property'} (integer). The largest record batch size allowed by Kafka (after compression if compression is enabled). If this is increased and there are consumers older than 0.10.2, the consumers' fetch size must also be increased so that the they can fetch record batches this large. In the latest message format version, records are always grouped into batches for efficiency. In previous message format versions, uncompressed records are not grouped into batches and this limit only applies to a single record in that case.
- [`message_downconversion_enable`](#spec.config.message_downconversion_enable-property){: name='spec.config.message_downconversion_enable-property'} (boolean). This configuration controls whether down-conversion of message formats is enabled to satisfy consume requests. When set to false, broker will not perform down-conversion for consumers expecting an older message format. The broker responds with UNSUPPORTED_VERSION error for consume requests from such older clients. This configuration does not apply to any message format conversion that might be required for replication to followers.
- [`message_format_version`](#spec.config.message_format_version-property){: name='spec.config.message_format_version-property'} (string). Specify the message format version the broker will use to append messages to the logs. The value should be a valid ApiVersion. Some examples are: 0.8.2, 0.9.0.0, 0.10.0, check ApiVersion for more details. By setting a particular message format version, the user is certifying that all the existing messages on disk are smaller or equal than the specified version. Setting this value incorrectly will cause consumers with older versions to break as they will receive messages with a format that they don't understand.
- [`message_timestamp_difference_max_ms`](#spec.config.message_timestamp_difference_max_ms-property){: name='spec.config.message_timestamp_difference_max_ms-property'} (integer). The maximum difference allowed between the timestamp when a broker receives a message and the timestamp specified in the message. If message.timestamp.type=CreateTime, a message will be rejected if the difference in timestamp exceeds this threshold. This configuration is ignored if message.timestamp.type=LogAppendTime.
- [`message_timestamp_type`](#spec.config.message_timestamp_type-property){: name='spec.config.message_timestamp_type-property'} (string). Define whether the timestamp in the message is message create time or log append time.
- [`min_cleanable_dirty_ratio`](#spec.config.min_cleanable_dirty_ratio-property){: name='spec.config.min_cleanable_dirty_ratio-property'} (number). This configuration controls how frequently the log compactor will attempt to clean the log (assuming log compaction is enabled). By default we will avoid cleaning a log where more than 50% of the log has been compacted. This ratio bounds the maximum space wasted in the log by duplicates (at 50% at most 50% of the log could be duplicates). A higher ratio will mean fewer, more efficient cleanings but will mean more wasted space in the log. If the max.compaction.lag.ms or the min.compaction.lag.ms configurations are also specified, then the log compactor considers the log to be eligible for compaction as soon as either: (i) the dirty ratio threshold has been met and the log has had dirty (uncompacted) records for at least the min.compaction.lag.ms duration, or (ii) if the log has had dirty (uncompacted) records for at most the max.compaction.lag.ms period.
- [`min_compaction_lag_ms`](#spec.config.min_compaction_lag_ms-property){: name='spec.config.min_compaction_lag_ms-property'} (integer). The minimum time a message will remain uncompacted in the log. Only applicable for logs that are being compacted.
- [`min_insync_replicas`](#spec.config.min_insync_replicas-property){: name='spec.config.min_insync_replicas-property'} (integer). When a producer sets acks to `all` (or `-1`), this configuration specifies the minimum number of replicas that must acknowledge a write for the write to be considered successful. If this minimum cannot be met, then the producer will raise an exception (either NotEnoughReplicas or NotEnoughReplicasAfterAppend). When used together, min.insync.replicas and acks allow you to enforce greater durability guarantees. A typical scenario would be to create a topic with a replication factor of 3, set min.insync.replicas to 2, and produce with acks of `all`. This will ensure that the producer raises an exception if a majority of replicas do not receive a write.
- [`preallocate`](#spec.config.preallocate-property){: name='spec.config.preallocate-property'} (boolean). True if we should preallocate the file on disk when creating a new log segment.
- [`remote_storage_enable`](#spec.config.remote_storage_enable-property){: name='spec.config.remote_storage_enable-property'} (boolean). Indicates whether tiered storage should be enabled.
- [`retention_bytes`](#spec.config.retention_bytes-property){: name='spec.config.retention_bytes-property'} (integer). This configuration controls the maximum size a partition (which consists of log segments) can grow to before we will discard old log segments to free up space if we are using the `delete` retention policy. By default there is no size limit only a time limit. Since this limit is enforced at the partition level, multiply it by the number of partitions to compute the topic retention in bytes.
- [`retention_ms`](#spec.config.retention_ms-property){: name='spec.config.retention_ms-property'} (integer). This configuration controls the maximum time we will retain a log before we will discard old log segments to free up space if we are using the `delete` retention policy. This represents an SLA on how soon consumers must read their data. If set to -1, no time limit is applied.
- [`segment_bytes`](#spec.config.segment_bytes-property){: name='spec.config.segment_bytes-property'} (integer). This configuration controls the segment file size for the log. Retention and cleaning is always done a file at a time so a larger segment size means fewer files but less granular control over retention. Setting this to a very low value has consequences, and the Aiven management plane ignores values less than 10 megabytes.
- [`segment_index_bytes`](#spec.config.segment_index_bytes-property){: name='spec.config.segment_index_bytes-property'} (integer). This configuration controls the size of the index that maps offsets to file positions. We preallocate this index file and shrink it only after log rolls. You generally should not need to change this setting.
- [`segment_jitter_ms`](#spec.config.segment_jitter_ms-property){: name='spec.config.segment_jitter_ms-property'} (integer). The maximum random jitter subtracted from the scheduled segment roll time to avoid thundering herds of segment rolling.
- [`segment_ms`](#spec.config.segment_ms-property){: name='spec.config.segment_ms-property'} (integer). This configuration controls the period of time after which Kafka will force the log to roll even if the segment file isn't full to ensure that retention can delete or compact old data. Setting this to a very low value has consequences, and the Aiven management plane ignores values less than 10 seconds.
- [`unclean_leader_election_enable`](#spec.config.unclean_leader_election_enable-property){: name='spec.config.unclean_leader_election_enable-property'} (boolean). Indicates whether to enable replicas not in the ISR set to be elected as leader as a last resort, even though doing so may result in data loss.

## tags {: #spec.tags }

_Appears on [`spec`](#spec)._

Kafka topic tags.

**Required**

- [`key`](#spec.tags.key-property){: name='spec.tags.key-property'} (string, Pattern: `^[a-zA-Z0-9_-]+$`, MinLength: 1, MaxLength: 64).

**Optional**

- [`value`](#spec.tags.value-property){: name='spec.tags.value-property'} (string, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 256).

