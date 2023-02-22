---
title: "Kafka"
---

## Usage example

```yaml
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: my-kafka
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: kafka-token

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: startup-2
  disc_space: 15Gib

  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
```

## Schema {: #Schema }

Kafka is the Schema for the kafkas API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Must be equal to `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Must be equal to `Kafka`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaSpec defines the desired state of Kafka. See below for [nested schema](#spec).

## spec {: #spec }

KafkaSpec defines the desired state of Kafka.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, MaxLength: 63). Target project.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, MaxLength: 256). Cloud the service runs in.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Information regarding secret creation. See below for [nested schema](#spec.connInfoSecretTarget).
- [`disk_space`](#spec.disk_space-property){: name='spec.disk_space-property'} (string). The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
- [`karapace`](#spec.karapace-property){: name='spec.karapace-property'} (boolean). Switch the service to use Karapace for schema registry and REST proxy.
- [`maintenanceWindowDow`](#spec.maintenanceWindowDow-property){: name='spec.maintenanceWindowDow-property'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
- [`maintenanceWindowTime`](#spec.maintenanceWindowTime-property){: name='spec.maintenanceWindowTime-property'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
- [`plan`](#spec.plan-property){: name='spec.plan-property'} (string, MaxLength: 128). Subscription plan.
- [`projectVPCRef`](#spec.projectVPCRef-property){: name='spec.projectVPCRef-property'} (object, Immutable). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See below for [nested schema](#spec.projectVPCRef).
- [`projectVpcId`](#spec.projectVpcId-property){: name='spec.projectVpcId-property'} (string, Immutable, MaxLength: 36). Identifier of the VPC the service should be in, if any.
- [`serviceIntegrations`](#spec.serviceIntegrations-property){: name='spec.serviceIntegrations-property'} (array of objects, Immutable, MaxItems: 1).  See below for [nested schema](#spec.serviceIntegrations).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize services.
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services.
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). Kafka specific user configuration options. See below for [nested schema](#spec.userConfig).

## authSecretRef {: #spec.authSecretRef }

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string). Name of the secret resource to be created. By default, is equal to the resource name.

## projectVPCRef {: #spec.projectVPCRef }

ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically.

**Required**

- [`name`](#spec.projectVPCRef.name-property){: name='spec.projectVPCRef.name-property'} (string, MinLength: 1). 

**Optional**

- [`namespace`](#spec.projectVPCRef.namespace-property){: name='spec.projectVPCRef.namespace-property'} (string, MinLength: 1). 

## serviceIntegrations {: #spec.serviceIntegrations }

**Required**

- [`integrationType`](#spec.serviceIntegrations.integrationType-property){: name='spec.serviceIntegrations.integrationType-property'} (string, Enum: `read_replica`). 
- [`sourceServiceName`](#spec.serviceIntegrations.sourceServiceName-property){: name='spec.serviceIntegrations.sourceServiceName-property'} (string, MinLength: 1, MaxLength: 64). 

## userConfig {: #spec.userConfig }

Kafka specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Additional Cloud Regions for Backup Replication.
- [`custom_domain`](#spec.userConfig.custom_domain-property){: name='spec.userConfig.custom_domain-property'} (string, MaxLength: 255). Serve the web frontend using a custom CNAME pointing to the Aiven DNS name.
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See below for [nested schema](#spec.userConfig.ip_filter).
- [`kafka`](#spec.userConfig.kafka-property){: name='spec.userConfig.kafka-property'} (object). Kafka broker configuration values. See below for [nested schema](#spec.userConfig.kafka).
- [`kafka_authentication_methods`](#spec.userConfig.kafka_authentication_methods-property){: name='spec.userConfig.kafka_authentication_methods-property'} (object). Kafka authentication methods. See below for [nested schema](#spec.userConfig.kafka_authentication_methods).
- [`kafka_connect`](#spec.userConfig.kafka_connect-property){: name='spec.userConfig.kafka_connect-property'} (boolean). Enable Kafka Connect service.
- [`kafka_connect_config`](#spec.userConfig.kafka_connect_config-property){: name='spec.userConfig.kafka_connect_config-property'} (object). Kafka Connect configuration values. See below for [nested schema](#spec.userConfig.kafka_connect_config).
- [`kafka_rest`](#spec.userConfig.kafka_rest-property){: name='spec.userConfig.kafka_rest-property'} (boolean). Enable Kafka-REST service.
- [`kafka_rest_authorization`](#spec.userConfig.kafka_rest_authorization-property){: name='spec.userConfig.kafka_rest_authorization-property'} (boolean). Enable authorization in Kafka-REST service.
- [`kafka_rest_config`](#spec.userConfig.kafka_rest_config-property){: name='spec.userConfig.kafka_rest_config-property'} (object). Kafka REST configuration. See below for [nested schema](#spec.userConfig.kafka_rest_config).
- [`kafka_version`](#spec.userConfig.kafka_version-property){: name='spec.userConfig.kafka_version-property'} (string, Enum: `3.2`, `3.3`). Kafka major version.
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`schema_registry`](#spec.userConfig.schema_registry-property){: name='spec.userConfig.schema_registry-property'} (boolean). Enable Schema-Registry service.
- [`schema_registry_config`](#spec.userConfig.schema_registry_config-property){: name='spec.userConfig.schema_registry_config-property'} (object). Schema Registry configuration. See below for [nested schema](#spec.userConfig.schema_registry_config).
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

### ip_filter {: #spec.userConfig.ip_filter }

Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### kafka {: #spec.userConfig.kafka }

Kafka broker configuration values.

**Optional**

- [`auto_create_topics_enable`](#spec.userConfig.kafka.auto_create_topics_enable-property){: name='spec.userConfig.kafka.auto_create_topics_enable-property'} (boolean). Enable auto creation of topics.
- [`compression_type`](#spec.userConfig.kafka.compression_type-property){: name='spec.userConfig.kafka.compression_type-property'} (string, Enum: `gzip`, `snappy`, `lz4`, `zstd`, `uncompressed`, `producer`). Specify the final compression type for a given topic. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'uncompressed' which is equivalent to no compression; and 'producer' which means retain the original compression codec set by the producer.
- [`connections_max_idle_ms`](#spec.userConfig.kafka.connections_max_idle_ms-property){: name='spec.userConfig.kafka.connections_max_idle_ms-property'} (integer, Minimum: 1000, Maximum: 3600000). Idle connections timeout: the server socket processor threads close the connections that idle for longer than this.
- [`default_replication_factor`](#spec.userConfig.kafka.default_replication_factor-property){: name='spec.userConfig.kafka.default_replication_factor-property'} (integer, Minimum: 1, Maximum: 10). Replication factor for autocreated topics.
- [`group_initial_rebalance_delay_ms`](#spec.userConfig.kafka.group_initial_rebalance_delay_ms-property){: name='spec.userConfig.kafka.group_initial_rebalance_delay_ms-property'} (integer, Minimum: 0, Maximum: 300000). The amount of time, in milliseconds, the group coordinator will wait for more consumers to join a new group before performing the first rebalance. A longer delay means potentially fewer rebalances, but increases the time until processing begins. The default value for this is 3 seconds. During development and testing it might be desirable to set this to 0 in order to not delay test execution time.
- [`group_max_session_timeout_ms`](#spec.userConfig.kafka.group_max_session_timeout_ms-property){: name='spec.userConfig.kafka.group_max_session_timeout_ms-property'} (integer, Minimum: 0, Maximum: 1800000). The maximum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures.
- [`group_min_session_timeout_ms`](#spec.userConfig.kafka.group_min_session_timeout_ms-property){: name='spec.userConfig.kafka.group_min_session_timeout_ms-property'} (integer, Minimum: 0, Maximum: 60000). The minimum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures.
- [`log_cleaner_delete_retention_ms`](#spec.userConfig.kafka.log_cleaner_delete_retention_ms-property){: name='spec.userConfig.kafka.log_cleaner_delete_retention_ms-property'} (integer, Minimum: 0, Maximum: 315569260000). How long are delete records retained?.
- [`log_cleaner_max_compaction_lag_ms`](#spec.userConfig.kafka.log_cleaner_max_compaction_lag_ms-property){: name='spec.userConfig.kafka.log_cleaner_max_compaction_lag_ms-property'} (integer, Minimum: 30000). The maximum amount of time message will remain uncompacted. Only applicable for logs that are being compacted.
- [`log_cleaner_min_cleanable_ratio`](#spec.userConfig.kafka.log_cleaner_min_cleanable_ratio-property){: name='spec.userConfig.kafka.log_cleaner_min_cleanable_ratio-property'} (number). Controls log compactor frequency. Larger value means more frequent compactions but also more space wasted for logs. Consider setting log.cleaner.max.compaction.lag.ms to enforce compactions sooner, instead of setting a very high value for this option.
- [`log_cleaner_min_compaction_lag_ms`](#spec.userConfig.kafka.log_cleaner_min_compaction_lag_ms-property){: name='spec.userConfig.kafka.log_cleaner_min_compaction_lag_ms-property'} (integer, Minimum: 0). The minimum time a message will remain uncompacted in the log. Only applicable for logs that are being compacted.
- [`log_cleanup_policy`](#spec.userConfig.kafka.log_cleanup_policy-property){: name='spec.userConfig.kafka.log_cleanup_policy-property'} (string, Enum: `delete`, `compact`, `compact,delete`). The default cleanup policy for segments beyond the retention window.
- [`log_flush_interval_messages`](#spec.userConfig.kafka.log_flush_interval_messages-property){: name='spec.userConfig.kafka.log_flush_interval_messages-property'} (integer, Minimum: 1). The number of messages accumulated on a log partition before messages are flushed to disk.
- [`log_flush_interval_ms`](#spec.userConfig.kafka.log_flush_interval_ms-property){: name='spec.userConfig.kafka.log_flush_interval_ms-property'} (integer, Minimum: 0). The maximum time in ms that a message in any topic is kept in memory before flushed to disk. If not set, the value in log.flush.scheduler.interval.ms is used.
- [`log_index_interval_bytes`](#spec.userConfig.kafka.log_index_interval_bytes-property){: name='spec.userConfig.kafka.log_index_interval_bytes-property'} (integer, Minimum: 0, Maximum: 104857600). The interval with which Kafka adds an entry to the offset index.
- [`log_index_size_max_bytes`](#spec.userConfig.kafka.log_index_size_max_bytes-property){: name='spec.userConfig.kafka.log_index_size_max_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). The maximum size in bytes of the offset index.
- [`log_message_downconversion_enable`](#spec.userConfig.kafka.log_message_downconversion_enable-property){: name='spec.userConfig.kafka.log_message_downconversion_enable-property'} (boolean). This configuration controls whether down-conversion of message formats is enabled to satisfy consume requests.
- [`log_message_timestamp_difference_max_ms`](#spec.userConfig.kafka.log_message_timestamp_difference_max_ms-property){: name='spec.userConfig.kafka.log_message_timestamp_difference_max_ms-property'} (integer, Minimum: 0). The maximum difference allowed between the timestamp when a broker receives a message and the timestamp specified in the message.
- [`log_message_timestamp_type`](#spec.userConfig.kafka.log_message_timestamp_type-property){: name='spec.userConfig.kafka.log_message_timestamp_type-property'} (string, Enum: `CreateTime`, `LogAppendTime`). Define whether the timestamp in the message is message create time or log append time.
- [`log_preallocate`](#spec.userConfig.kafka.log_preallocate-property){: name='spec.userConfig.kafka.log_preallocate-property'} (boolean). Should pre allocate file when create new segment?.
- [`log_retention_bytes`](#spec.userConfig.kafka.log_retention_bytes-property){: name='spec.userConfig.kafka.log_retention_bytes-property'} (integer, Minimum: -1). The maximum size of the log before deleting messages.
- [`log_retention_hours`](#spec.userConfig.kafka.log_retention_hours-property){: name='spec.userConfig.kafka.log_retention_hours-property'} (integer, Minimum: -1, Maximum: 2147483647). The number of hours to keep a log file before deleting it.
- [`log_retention_ms`](#spec.userConfig.kafka.log_retention_ms-property){: name='spec.userConfig.kafka.log_retention_ms-property'} (integer, Minimum: -1). The number of milliseconds to keep a log file before deleting it (in milliseconds), If not set, the value in log.retention.minutes is used. If set to -1, no time limit is applied.
- [`log_roll_jitter_ms`](#spec.userConfig.kafka.log_roll_jitter_ms-property){: name='spec.userConfig.kafka.log_roll_jitter_ms-property'} (integer, Minimum: 0). The maximum jitter to subtract from logRollTimeMillis (in milliseconds). If not set, the value in log.roll.jitter.hours is used.
- [`log_roll_ms`](#spec.userConfig.kafka.log_roll_ms-property){: name='spec.userConfig.kafka.log_roll_ms-property'} (integer, Minimum: 1). The maximum time before a new log segment is rolled out (in milliseconds).
- [`log_segment_bytes`](#spec.userConfig.kafka.log_segment_bytes-property){: name='spec.userConfig.kafka.log_segment_bytes-property'} (integer, Minimum: 10485760, Maximum: 1073741824). The maximum size of a single log file.
- [`log_segment_delete_delay_ms`](#spec.userConfig.kafka.log_segment_delete_delay_ms-property){: name='spec.userConfig.kafka.log_segment_delete_delay_ms-property'} (integer, Minimum: 0, Maximum: 3600000). The amount of time to wait before deleting a file from the filesystem.
- [`max_connections_per_ip`](#spec.userConfig.kafka.max_connections_per_ip-property){: name='spec.userConfig.kafka.max_connections_per_ip-property'} (integer, Minimum: 256, Maximum: 2147483647). The maximum number of connections allowed from each ip address (defaults to 2147483647).
- [`max_incremental_fetch_session_cache_slots`](#spec.userConfig.kafka.max_incremental_fetch_session_cache_slots-property){: name='spec.userConfig.kafka.max_incremental_fetch_session_cache_slots-property'} (integer, Minimum: 1000, Maximum: 10000). The maximum number of incremental fetch sessions that the broker will maintain.
- [`message_max_bytes`](#spec.userConfig.kafka.message_max_bytes-property){: name='spec.userConfig.kafka.message_max_bytes-property'} (integer, Minimum: 0, Maximum: 100001200). The maximum size of message that the server can receive.
- [`min_insync_replicas`](#spec.userConfig.kafka.min_insync_replicas-property){: name='spec.userConfig.kafka.min_insync_replicas-property'} (integer, Minimum: 1, Maximum: 7). When a producer sets acks to 'all' (or '-1'), min.insync.replicas specifies the minimum number of replicas that must acknowledge a write for the write to be considered successful.
- [`num_partitions`](#spec.userConfig.kafka.num_partitions-property){: name='spec.userConfig.kafka.num_partitions-property'} (integer, Minimum: 1, Maximum: 1000). Number of partitions for autocreated topics.
- [`offsets_retention_minutes`](#spec.userConfig.kafka.offsets_retention_minutes-property){: name='spec.userConfig.kafka.offsets_retention_minutes-property'} (integer, Minimum: 1, Maximum: 2147483647). Log retention window in minutes for offsets topic.
- [`producer_purgatory_purge_interval_requests`](#spec.userConfig.kafka.producer_purgatory_purge_interval_requests-property){: name='spec.userConfig.kafka.producer_purgatory_purge_interval_requests-property'} (integer, Minimum: 10, Maximum: 10000). The purge interval (in number of requests) of the producer request purgatory(defaults to 1000).
- [`replica_fetch_max_bytes`](#spec.userConfig.kafka.replica_fetch_max_bytes-property){: name='spec.userConfig.kafka.replica_fetch_max_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). The number of bytes of messages to attempt to fetch for each partition (defaults to 1048576). This is not an absolute maximum, if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made.
- [`replica_fetch_response_max_bytes`](#spec.userConfig.kafka.replica_fetch_response_max_bytes-property){: name='spec.userConfig.kafka.replica_fetch_response_max_bytes-property'} (integer, Minimum: 10485760, Maximum: 1048576000). Maximum bytes expected for the entire fetch response (defaults to 10485760). Records are fetched in batches, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made. As such, this is not an absolute maximum.
- [`socket_request_max_bytes`](#spec.userConfig.kafka.socket_request_max_bytes-property){: name='spec.userConfig.kafka.socket_request_max_bytes-property'} (integer, Minimum: 10485760, Maximum: 209715200). The maximum number of bytes in a socket request (defaults to 104857600).
- [`transaction_remove_expired_transaction_cleanup_interval_ms`](#spec.userConfig.kafka.transaction_remove_expired_transaction_cleanup_interval_ms-property){: name='spec.userConfig.kafka.transaction_remove_expired_transaction_cleanup_interval_ms-property'} (integer, Minimum: 600000, Maximum: 3600000). The interval at which to remove transactions that have expired due to transactional.id.expiration.ms passing (defaults to 3600000 (1 hour)).
- [`transaction_state_log_segment_bytes`](#spec.userConfig.kafka.transaction_state_log_segment_bytes-property){: name='spec.userConfig.kafka.transaction_state_log_segment_bytes-property'} (integer, Minimum: 1048576, Maximum: 2147483647). The transaction topic segment bytes should be kept relatively small in order to facilitate faster log compaction and cache loads (defaults to 104857600 (100 mebibytes)).

### kafka_authentication_methods {: #spec.userConfig.kafka_authentication_methods }

Kafka authentication methods.

**Optional**

- [`certificate`](#spec.userConfig.kafka_authentication_methods.certificate-property){: name='spec.userConfig.kafka_authentication_methods.certificate-property'} (boolean). Enable certificate/SSL authentication.
- [`sasl`](#spec.userConfig.kafka_authentication_methods.sasl-property){: name='spec.userConfig.kafka_authentication_methods.sasl-property'} (boolean). Enable SASL authentication.

### kafka_connect_config {: #spec.userConfig.kafka_connect_config }

Kafka Connect configuration values.

**Optional**

- [`connector_client_config_override_policy`](#spec.userConfig.kafka_connect_config.connector_client_config_override_policy-property){: name='spec.userConfig.kafka_connect_config.connector_client_config_override_policy-property'} (string, Enum: `None`, `All`). Defines what client configurations can be overridden by the connector. Default is None.
- [`consumer_auto_offset_reset`](#spec.userConfig.kafka_connect_config.consumer_auto_offset_reset-property){: name='spec.userConfig.kafka_connect_config.consumer_auto_offset_reset-property'} (string, Enum: `earliest`, `latest`). What to do when there is no initial offset in Kafka or if the current offset does not exist any more on the server. Default is earliest.
- [`consumer_fetch_max_bytes`](#spec.userConfig.kafka_connect_config.consumer_fetch_max_bytes-property){: name='spec.userConfig.kafka_connect_config.consumer_fetch_max_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). Records are fetched in batches by the consumer, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that the consumer can make progress. As such, this is not a absolute maximum.
- [`consumer_isolation_level`](#spec.userConfig.kafka_connect_config.consumer_isolation_level-property){: name='spec.userConfig.kafka_connect_config.consumer_isolation_level-property'} (string, Enum: `read_uncommitted`, `read_committed`). Transaction read isolation level. read_uncommitted is the default, but read_committed can be used if consume-exactly-once behavior is desired.
- [`consumer_max_partition_fetch_bytes`](#spec.userConfig.kafka_connect_config.consumer_max_partition_fetch_bytes-property){: name='spec.userConfig.kafka_connect_config.consumer_max_partition_fetch_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). Records are fetched in batches by the consumer.If the first record batch in the first non-empty partition of the fetch is larger than this limit, the batch will still be returned to ensure that the consumer can make progress.
- [`consumer_max_poll_interval_ms`](#spec.userConfig.kafka_connect_config.consumer_max_poll_interval_ms-property){: name='spec.userConfig.kafka_connect_config.consumer_max_poll_interval_ms-property'} (integer, Minimum: 1, Maximum: 2147483647). The maximum delay in milliseconds between invocations of poll() when using consumer group management (defaults to 300000).
- [`consumer_max_poll_records`](#spec.userConfig.kafka_connect_config.consumer_max_poll_records-property){: name='spec.userConfig.kafka_connect_config.consumer_max_poll_records-property'} (integer, Minimum: 1, Maximum: 10000). The maximum number of records returned in a single call to poll() (defaults to 500).
- [`offset_flush_interval_ms`](#spec.userConfig.kafka_connect_config.offset_flush_interval_ms-property){: name='spec.userConfig.kafka_connect_config.offset_flush_interval_ms-property'} (integer, Minimum: 1, Maximum: 100000000). The interval at which to try committing offsets for tasks (defaults to 60000).
- [`offset_flush_timeout_ms`](#spec.userConfig.kafka_connect_config.offset_flush_timeout_ms-property){: name='spec.userConfig.kafka_connect_config.offset_flush_timeout_ms-property'} (integer, Minimum: 1, Maximum: 2147483647). Maximum number of milliseconds to wait for records to flush and partition offset data to be committed to offset storage before cancelling the process and restoring the offset data to be committed in a future attempt (defaults to 5000).
- [`producer_batch_size`](#spec.userConfig.kafka_connect_config.producer_batch_size-property){: name='spec.userConfig.kafka_connect_config.producer_batch_size-property'} (integer, Minimum: 0, Maximum: 5242880). This setting gives the upper bound of the batch size to be sent. If there are fewer than this many bytes accumulated for this partition, the producer will 'linger' for the linger.ms time waiting for more records to show up. A batch size of zero will disable batching entirely (defaults to 16384).
- [`producer_buffer_memory`](#spec.userConfig.kafka_connect_config.producer_buffer_memory-property){: name='spec.userConfig.kafka_connect_config.producer_buffer_memory-property'} (integer, Minimum: 5242880, Maximum: 134217728). The total bytes of memory the producer can use to buffer records waiting to be sent to the broker (defaults to 33554432).
- [`producer_compression_type`](#spec.userConfig.kafka_connect_config.producer_compression_type-property){: name='spec.userConfig.kafka_connect_config.producer_compression_type-property'} (string, Enum: `gzip`, `snappy`, `lz4`, `zstd`, `none`). Specify the default compression type for producers. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'none' which is the default and equivalent to no compression.
- [`producer_linger_ms`](#spec.userConfig.kafka_connect_config.producer_linger_ms-property){: name='spec.userConfig.kafka_connect_config.producer_linger_ms-property'} (integer, Minimum: 0, Maximum: 5000). This setting gives the upper bound on the delay for batching: once there is batch.size worth of records for a partition it will be sent immediately regardless of this setting, however if there are fewer than this many bytes accumulated for this partition the producer will 'linger' for the specified time waiting for more records to show up. Defaults to 0.
- [`producer_max_request_size`](#spec.userConfig.kafka_connect_config.producer_max_request_size-property){: name='spec.userConfig.kafka_connect_config.producer_max_request_size-property'} (integer, Minimum: 131072, Maximum: 67108864). This setting will limit the number of record batches the producer will send in a single request to avoid sending huge requests.
- [`session_timeout_ms`](#spec.userConfig.kafka_connect_config.session_timeout_ms-property){: name='spec.userConfig.kafka_connect_config.session_timeout_ms-property'} (integer, Minimum: 1, Maximum: 2147483647). The timeout in milliseconds used to detect failures when using Kafka’s group management facilities (defaults to 10000).

### kafka_rest_config {: #spec.userConfig.kafka_rest_config }

Kafka REST configuration.

**Optional**

- [`consumer_enable_auto_commit`](#spec.userConfig.kafka_rest_config.consumer_enable_auto_commit-property){: name='spec.userConfig.kafka_rest_config.consumer_enable_auto_commit-property'} (boolean). If true the consumer's offset will be periodically committed to Kafka in the background.
- [`consumer_request_max_bytes`](#spec.userConfig.kafka_rest_config.consumer_request_max_bytes-property){: name='spec.userConfig.kafka_rest_config.consumer_request_max_bytes-property'} (integer, Minimum: 0, Maximum: 671088640). Maximum number of bytes in unencoded message keys and values by a single request.
- [`consumer_request_timeout_ms`](#spec.userConfig.kafka_rest_config.consumer_request_timeout_ms-property){: name='spec.userConfig.kafka_rest_config.consumer_request_timeout_ms-property'} (integer, Enum: `1000`, `15000`, `30000`, Minimum: 1000, Maximum: 30000). The maximum total time to wait for messages for a request if the maximum number of messages has not yet been reached.
- [`producer_acks`](#spec.userConfig.kafka_rest_config.producer_acks-property){: name='spec.userConfig.kafka_rest_config.producer_acks-property'} (string, Enum: `all`, `-1`, `0`, `1`). The number of acknowledgments the producer requires the leader to have received before considering a request complete. If set to 'all' or '-1', the leader will wait for the full set of in-sync replicas to acknowledge the record.
- [`producer_compression_type`](#spec.userConfig.kafka_rest_config.producer_compression_type-property){: name='spec.userConfig.kafka_rest_config.producer_compression_type-property'} (string, Enum: `gzip`, `snappy`, `lz4`, `zstd`, `none`). Specify the default compression type for producers. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'none' which is the default and equivalent to no compression.
- [`producer_linger_ms`](#spec.userConfig.kafka_rest_config.producer_linger_ms-property){: name='spec.userConfig.kafka_rest_config.producer_linger_ms-property'} (integer, Minimum: 0, Maximum: 5000). Wait for up to the given delay to allow batching records together.
- [`simpleconsumer_pool_size_max`](#spec.userConfig.kafka_rest_config.simpleconsumer_pool_size_max-property){: name='spec.userConfig.kafka_rest_config.simpleconsumer_pool_size_max-property'} (integer, Minimum: 10, Maximum: 250). Maximum number of SimpleConsumers that can be instantiated per broker.

### private_access {: #spec.userConfig.private_access }

Allow access to selected service ports from private networks.

**Optional**

- [`kafka`](#spec.userConfig.private_access.kafka-property){: name='spec.userConfig.private_access.kafka-property'} (boolean). Allow clients to connect to kafka with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`kafka_connect`](#spec.userConfig.private_access.kafka_connect-property){: name='spec.userConfig.private_access.kafka_connect-property'} (boolean). Allow clients to connect to kafka_connect with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`kafka_rest`](#spec.userConfig.private_access.kafka_rest-property){: name='spec.userConfig.private_access.kafka_rest-property'} (boolean). Allow clients to connect to kafka_rest with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`schema_registry`](#spec.userConfig.private_access.schema_registry-property){: name='spec.userConfig.private_access.schema_registry-property'} (boolean). Allow clients to connect to schema_registry with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

Allow access to selected service components through Privatelink.

**Optional**

- [`jolokia`](#spec.userConfig.privatelink_access.jolokia-property){: name='spec.userConfig.privatelink_access.jolokia-property'} (boolean). Enable jolokia.
- [`kafka`](#spec.userConfig.privatelink_access.kafka-property){: name='spec.userConfig.privatelink_access.kafka-property'} (boolean). Enable kafka.
- [`kafka_connect`](#spec.userConfig.privatelink_access.kafka_connect-property){: name='spec.userConfig.privatelink_access.kafka_connect-property'} (boolean). Enable kafka_connect.
- [`kafka_rest`](#spec.userConfig.privatelink_access.kafka_rest-property){: name='spec.userConfig.privatelink_access.kafka_rest-property'} (boolean). Enable kafka_rest.
- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.
- [`schema_registry`](#spec.userConfig.privatelink_access.schema_registry-property){: name='spec.userConfig.privatelink_access.schema_registry-property'} (boolean). Enable schema_registry.

### public_access {: #spec.userConfig.public_access }

Allow access to selected service ports from the public Internet.

**Optional**

- [`kafka`](#spec.userConfig.public_access.kafka-property){: name='spec.userConfig.public_access.kafka-property'} (boolean). Allow clients to connect to kafka from the public internet for service nodes that are in a project VPC or another type of private network.
- [`kafka_connect`](#spec.userConfig.public_access.kafka_connect-property){: name='spec.userConfig.public_access.kafka_connect-property'} (boolean). Allow clients to connect to kafka_connect from the public internet for service nodes that are in a project VPC or another type of private network.
- [`kafka_rest`](#spec.userConfig.public_access.kafka_rest-property){: name='spec.userConfig.public_access.kafka_rest-property'} (boolean). Allow clients to connect to kafka_rest from the public internet for service nodes that are in a project VPC or another type of private network.
- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.
- [`schema_registry`](#spec.userConfig.public_access.schema_registry-property){: name='spec.userConfig.public_access.schema_registry-property'} (boolean). Allow clients to connect to schema_registry from the public internet for service nodes that are in a project VPC or another type of private network.

### schema_registry_config {: #spec.userConfig.schema_registry_config }

Schema Registry configuration.

**Optional**

- [`leader_eligibility`](#spec.userConfig.schema_registry_config.leader_eligibility-property){: name='spec.userConfig.schema_registry_config.leader_eligibility-property'} (boolean). If true, Karapace / Schema Registry on the service nodes can participate in leader election. It might be needed to disable this when the schemas topic is replicated to a secondary cluster and Karapace / Schema Registry there must not participate in leader election. Defaults to `true`.
- [`topic_name`](#spec.userConfig.schema_registry_config.topic_name-property){: name='spec.userConfig.schema_registry_config.topic_name-property'} (string, MinLength: 1, MaxLength: 249). The durable single partition topic that acts as the durable log for the data. This topic must be compacted to avoid losing data due to retention policy. Please note that changing this configuration in an existing Schema Registry / Karapace setup leads to previous schemas being inaccessible, data encoded with them potentially unreadable and schema ID sequence put out of order. It's only possible to do the switch while Schema Registry / Karapace is disabled. Defaults to `_schemas`.

