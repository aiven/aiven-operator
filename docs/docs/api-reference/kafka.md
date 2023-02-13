---
title: "Kafka"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | Kafka |

KafkaSpec defines the desired state of Kafka.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`cloudName`](#cloudName){: name='cloudName'} (string, MaxLength: 256). Cloud the service runs in. 
- [`connInfoSecretTarget`](#connInfoSecretTarget){: name='connInfoSecretTarget'} (object). Information regarding secret creation. See [below for nested schema](#connInfoSecretTarget).
- [`disk_space`](#disk_space){: name='disk_space'} (string). The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing. 
- [`karapace`](#karapace){: name='karapace'} (boolean). Switch the service to use Karapace for schema registry and REST proxy. 
- [`maintenanceWindowDow`](#maintenanceWindowDow){: name='maintenanceWindowDow'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc. 
- [`maintenanceWindowTime`](#maintenanceWindowTime){: name='maintenanceWindowTime'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format. 
- [`plan`](#plan){: name='plan'} (string, MaxLength: 128). Subscription plan. 
- [`project`](#project){: name='project'} (string, Immutable, MaxLength: 63). Target project. 
- [`projectVPCRef`](#projectVPCRef){: name='projectVPCRef'} (object, Immutable). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See [below for nested schema](#projectVPCRef).
- [`projectVpcId`](#projectVpcId){: name='projectVpcId'} (string, Immutable, MaxLength: 36). Identifier of the VPC the service should be in, if any. 
- [`serviceIntegrations`](#serviceIntegrations){: name='serviceIntegrations'} (array, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See [below for nested schema](#serviceIntegrations).
- [`tags`](#tags){: name='tags'} (object). Tags are key-value pairs that allow you to categorize services. 
- [`terminationProtection`](#terminationProtection){: name='terminationProtection'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services. 
- [`userConfig`](#userConfig){: name='userConfig'} (object). Kafka specific user configuration options. See [below for nested schema](#userConfig).

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

## connInfoSecretTarget {: #connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#name){: name='name'} (string). Name of the Secret resource to be created. 

## projectVPCRef {: #projectVPCRef }

ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically.

**Required**

- [`name`](#name){: name='name'} (string, MinLength: 1).  

**Optional**

- [`namespace`](#namespace){: name='namespace'} (string, MinLength: 1).  

## serviceIntegrations {: #serviceIntegrations }

Service integrations to specify when creating a service. Not applied after initial service creation.

**Required**

- [`integrationType`](#integrationType){: name='integrationType'} (string, Enum: `read_replica`).  
- [`sourceServiceName`](#sourceServiceName){: name='sourceServiceName'} (string, MinLength: 1, MaxLength: 64).  

## userConfig {: #userConfig }

Kafka specific user configuration options.

**Optional**

- [`additional_backup_regions`](#additional_backup_regions){: name='additional_backup_regions'} (array, MaxItems: 1). Additional Cloud Regions for Backup Replication. 
- [`custom_domain`](#custom_domain){: name='custom_domain'} (string, MaxLength: 255). Serve the web frontend using a custom CNAME pointing to the Aiven DNS name. 
- [`ip_filter`](#ip_filter){: name='ip_filter'} (array, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See [below for nested schema](#ip_filter).
- [`kafka`](#kafka){: name='kafka'} (object). Kafka broker configuration values. See [below for nested schema](#kafka).
- [`kafka_authentication_methods`](#kafka_authentication_methods){: name='kafka_authentication_methods'} (object). Kafka authentication methods. See [below for nested schema](#kafka_authentication_methods).
- [`kafka_connect`](#kafka_connect){: name='kafka_connect'} (boolean). Enable Kafka Connect service. 
- [`kafka_connect_config`](#kafka_connect_config){: name='kafka_connect_config'} (object). Kafka Connect configuration values. See [below for nested schema](#kafka_connect_config).
- [`kafka_rest`](#kafka_rest){: name='kafka_rest'} (boolean). Enable Kafka-REST service. 
- [`kafka_rest_authorization`](#kafka_rest_authorization){: name='kafka_rest_authorization'} (boolean). Enable authorization in Kafka-REST service. 
- [`kafka_rest_config`](#kafka_rest_config){: name='kafka_rest_config'} (object). Kafka REST configuration. See [below for nested schema](#kafka_rest_config).
- [`kafka_version`](#kafka_version){: name='kafka_version'} (string, Enum: `3.2`, `3.3`). Kafka major version. 
- [`private_access`](#private_access){: name='private_access'} (object). Allow access to selected service ports from private networks. See [below for nested schema](#private_access).
- [`privatelink_access`](#privatelink_access){: name='privatelink_access'} (object). Allow access to selected service components through Privatelink. See [below for nested schema](#privatelink_access).
- [`public_access`](#public_access){: name='public_access'} (object). Allow access to selected service ports from the public Internet. See [below for nested schema](#public_access).
- [`schema_registry`](#schema_registry){: name='schema_registry'} (boolean). Enable Schema-Registry service. 
- [`schema_registry_config`](#schema_registry_config){: name='schema_registry_config'} (object). Schema Registry configuration. See [below for nested schema](#schema_registry_config).
- [`static_ips`](#static_ips){: name='static_ips'} (boolean). Use static public IP addresses. 

### ip_filter {: #ip_filter }

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#network){: name='network'} (string, MaxLength: 43). CIDR address block. 

**Optional**

- [`description`](#description){: name='description'} (string, MaxLength: 1024). Description for IP filter list entry. 

### kafka {: #kafka }

Kafka broker configuration values.

**Optional**

- [`auto_create_topics_enable`](#auto_create_topics_enable){: name='auto_create_topics_enable'} (boolean). Enable auto creation of topics. 
- [`compression_type`](#compression_type){: name='compression_type'} (string, Enum: `gzip`, `snappy`, `lz4`, `zstd`, `uncompressed`, `producer`). Specify the final compression type for a given topic. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'uncompressed' which is equivalent to no compression; and 'producer' which means retain the original compression codec set by the producer. 
- [`connections_max_idle_ms`](#connections_max_idle_ms){: name='connections_max_idle_ms'} (integer, Minimum: 1000, Maximum: 3600000). Idle connections timeout: the server socket processor threads close the connections that idle for longer than this. 
- [`default_replication_factor`](#default_replication_factor){: name='default_replication_factor'} (integer, Minimum: 1, Maximum: 10). Replication factor for autocreated topics. 
- [`group_initial_rebalance_delay_ms`](#group_initial_rebalance_delay_ms){: name='group_initial_rebalance_delay_ms'} (integer, Minimum: 0, Maximum: 300000). The amount of time, in milliseconds, the group coordinator will wait for more consumers to join a new group before performing the first rebalance. A longer delay means potentially fewer rebalances, but increases the time until processing begins. The default value for this is 3 seconds. During development and testing it might be desirable to set this to 0 in order to not delay test execution time. 
- [`group_max_session_timeout_ms`](#group_max_session_timeout_ms){: name='group_max_session_timeout_ms'} (integer, Minimum: 0, Maximum: 1800000). The maximum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures. 
- [`group_min_session_timeout_ms`](#group_min_session_timeout_ms){: name='group_min_session_timeout_ms'} (integer, Minimum: 0, Maximum: 60000). The minimum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures. 
- [`log_cleaner_delete_retention_ms`](#log_cleaner_delete_retention_ms){: name='log_cleaner_delete_retention_ms'} (integer, Minimum: 0, Maximum: 315569260000). How long are delete records retained?. 
- [`log_cleaner_max_compaction_lag_ms`](#log_cleaner_max_compaction_lag_ms){: name='log_cleaner_max_compaction_lag_ms'} (integer, Minimum: 30000). The maximum amount of time message will remain uncompacted. Only applicable for logs that are being compacted. 
- [`log_cleaner_min_cleanable_ratio`](#log_cleaner_min_cleanable_ratio){: name='log_cleaner_min_cleanable_ratio'} (number). Controls log compactor frequency. Larger value means more frequent compactions but also more space wasted for logs. Consider setting log.cleaner.max.compaction.lag.ms to enforce compactions sooner, instead of setting a very high value for this option. 
- [`log_cleaner_min_compaction_lag_ms`](#log_cleaner_min_compaction_lag_ms){: name='log_cleaner_min_compaction_lag_ms'} (integer, Minimum: 0). The minimum time a message will remain uncompacted in the log. Only applicable for logs that are being compacted. 
- [`log_cleanup_policy`](#log_cleanup_policy){: name='log_cleanup_policy'} (string, Enum: `delete`, `compact`, `compact,delete`). The default cleanup policy for segments beyond the retention window. 
- [`log_flush_interval_messages`](#log_flush_interval_messages){: name='log_flush_interval_messages'} (integer, Minimum: 1). The number of messages accumulated on a log partition before messages are flushed to disk. 
- [`log_flush_interval_ms`](#log_flush_interval_ms){: name='log_flush_interval_ms'} (integer, Minimum: 0). The maximum time in ms that a message in any topic is kept in memory before flushed to disk. If not set, the value in log.flush.scheduler.interval.ms is used. 
- [`log_index_interval_bytes`](#log_index_interval_bytes){: name='log_index_interval_bytes'} (integer, Minimum: 0, Maximum: 104857600). The interval with which Kafka adds an entry to the offset index. 
- [`log_index_size_max_bytes`](#log_index_size_max_bytes){: name='log_index_size_max_bytes'} (integer, Minimum: 1048576, Maximum: 104857600). The maximum size in bytes of the offset index. 
- [`log_message_downconversion_enable`](#log_message_downconversion_enable){: name='log_message_downconversion_enable'} (boolean). This configuration controls whether down-conversion of message formats is enabled to satisfy consume requests. 
- [`log_message_timestamp_difference_max_ms`](#log_message_timestamp_difference_max_ms){: name='log_message_timestamp_difference_max_ms'} (integer, Minimum: 0). The maximum difference allowed between the timestamp when a broker receives a message and the timestamp specified in the message. 
- [`log_message_timestamp_type`](#log_message_timestamp_type){: name='log_message_timestamp_type'} (string, Enum: `CreateTime`, `LogAppendTime`). Define whether the timestamp in the message is message create time or log append time. 
- [`log_preallocate`](#log_preallocate){: name='log_preallocate'} (boolean). Should pre allocate file when create new segment?. 
- [`log_retention_bytes`](#log_retention_bytes){: name='log_retention_bytes'} (integer, Minimum: -1). The maximum size of the log before deleting messages. 
- [`log_retention_hours`](#log_retention_hours){: name='log_retention_hours'} (integer, Minimum: -1, Maximum: 2147483647). The number of hours to keep a log file before deleting it. 
- [`log_retention_ms`](#log_retention_ms){: name='log_retention_ms'} (integer, Minimum: -1). The number of milliseconds to keep a log file before deleting it (in milliseconds), If not set, the value in log.retention.minutes is used. If set to -1, no time limit is applied. 
- [`log_roll_jitter_ms`](#log_roll_jitter_ms){: name='log_roll_jitter_ms'} (integer, Minimum: 0). The maximum jitter to subtract from logRollTimeMillis (in milliseconds). If not set, the value in log.roll.jitter.hours is used. 
- [`log_roll_ms`](#log_roll_ms){: name='log_roll_ms'} (integer, Minimum: 1). The maximum time before a new log segment is rolled out (in milliseconds). 
- [`log_segment_bytes`](#log_segment_bytes){: name='log_segment_bytes'} (integer, Minimum: 10485760, Maximum: 1073741824). The maximum size of a single log file. 
- [`log_segment_delete_delay_ms`](#log_segment_delete_delay_ms){: name='log_segment_delete_delay_ms'} (integer, Minimum: 0, Maximum: 3600000). The amount of time to wait before deleting a file from the filesystem. 
- [`max_connections_per_ip`](#max_connections_per_ip){: name='max_connections_per_ip'} (integer, Minimum: 256, Maximum: 2147483647). The maximum number of connections allowed from each ip address (defaults to 2147483647). 
- [`max_incremental_fetch_session_cache_slots`](#max_incremental_fetch_session_cache_slots){: name='max_incremental_fetch_session_cache_slots'} (integer, Minimum: 1000, Maximum: 10000). The maximum number of incremental fetch sessions that the broker will maintain. 
- [`message_max_bytes`](#message_max_bytes){: name='message_max_bytes'} (integer, Minimum: 0, Maximum: 100001200). The maximum size of message that the server can receive. 
- [`min_insync_replicas`](#min_insync_replicas){: name='min_insync_replicas'} (integer, Minimum: 1, Maximum: 7). When a producer sets acks to 'all' (or '-1'), min.insync.replicas specifies the minimum number of replicas that must acknowledge a write for the write to be considered successful. 
- [`num_partitions`](#num_partitions){: name='num_partitions'} (integer, Minimum: 1, Maximum: 1000). Number of partitions for autocreated topics. 
- [`offsets_retention_minutes`](#offsets_retention_minutes){: name='offsets_retention_minutes'} (integer, Minimum: 1, Maximum: 2147483647). Log retention window in minutes for offsets topic. 
- [`producer_purgatory_purge_interval_requests`](#producer_purgatory_purge_interval_requests){: name='producer_purgatory_purge_interval_requests'} (integer, Minimum: 10, Maximum: 10000). The purge interval (in number of requests) of the producer request purgatory(defaults to 1000). 
- [`replica_fetch_max_bytes`](#replica_fetch_max_bytes){: name='replica_fetch_max_bytes'} (integer, Minimum: 1048576, Maximum: 104857600). The number of bytes of messages to attempt to fetch for each partition (defaults to 1048576). This is not an absolute maximum, if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made. 
- [`replica_fetch_response_max_bytes`](#replica_fetch_response_max_bytes){: name='replica_fetch_response_max_bytes'} (integer, Minimum: 10485760, Maximum: 1048576000). Maximum bytes expected for the entire fetch response (defaults to 10485760). Records are fetched in batches, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made. As such, this is not an absolute maximum. 
- [`socket_request_max_bytes`](#socket_request_max_bytes){: name='socket_request_max_bytes'} (integer, Minimum: 10485760, Maximum: 209715200). The maximum number of bytes in a socket request (defaults to 104857600). 
- [`transaction_remove_expired_transaction_cleanup_interval_ms`](#transaction_remove_expired_transaction_cleanup_interval_ms){: name='transaction_remove_expired_transaction_cleanup_interval_ms'} (integer, Minimum: 600000, Maximum: 3600000). The interval at which to remove transactions that have expired due to transactional.id.expiration.ms passing (defaults to 3600000 (1 hour)). 
- [`transaction_state_log_segment_bytes`](#transaction_state_log_segment_bytes){: name='transaction_state_log_segment_bytes'} (integer, Minimum: 1048576, Maximum: 2147483647). The transaction topic segment bytes should be kept relatively small in order to facilitate faster log compaction and cache loads (defaults to 104857600 (100 mebibytes)). 

### kafka_authentication_methods {: #kafka_authentication_methods }

Kafka authentication methods.

**Optional**

- [`certificate`](#certificate){: name='certificate'} (boolean). Enable certificate/SSL authentication. 
- [`sasl`](#sasl){: name='sasl'} (boolean). Enable SASL authentication. 

### kafka_connect_config {: #kafka_connect_config }

Kafka Connect configuration values.

**Optional**

- [`connector_client_config_override_policy`](#connector_client_config_override_policy){: name='connector_client_config_override_policy'} (string, Enum: `None`, `All`). Defines what client configurations can be overridden by the connector. Default is None. 
- [`consumer_auto_offset_reset`](#consumer_auto_offset_reset){: name='consumer_auto_offset_reset'} (string, Enum: `earliest`, `latest`). What to do when there is no initial offset in Kafka or if the current offset does not exist any more on the server. Default is earliest. 
- [`consumer_fetch_max_bytes`](#consumer_fetch_max_bytes){: name='consumer_fetch_max_bytes'} (integer, Minimum: 1048576, Maximum: 104857600). Records are fetched in batches by the consumer, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that the consumer can make progress. As such, this is not a absolute maximum. 
- [`consumer_isolation_level`](#consumer_isolation_level){: name='consumer_isolation_level'} (string, Enum: `read_uncommitted`, `read_committed`). Transaction read isolation level. read_uncommitted is the default, but read_committed can be used if consume-exactly-once behavior is desired. 
- [`consumer_max_partition_fetch_bytes`](#consumer_max_partition_fetch_bytes){: name='consumer_max_partition_fetch_bytes'} (integer, Minimum: 1048576, Maximum: 104857600). Records are fetched in batches by the consumer.If the first record batch in the first non-empty partition of the fetch is larger than this limit, the batch will still be returned to ensure that the consumer can make progress. 
- [`consumer_max_poll_interval_ms`](#consumer_max_poll_interval_ms){: name='consumer_max_poll_interval_ms'} (integer, Minimum: 1, Maximum: 2147483647). The maximum delay in milliseconds between invocations of poll() when using consumer group management (defaults to 300000). 
- [`consumer_max_poll_records`](#consumer_max_poll_records){: name='consumer_max_poll_records'} (integer, Minimum: 1, Maximum: 10000). The maximum number of records returned in a single call to poll() (defaults to 500). 
- [`offset_flush_interval_ms`](#offset_flush_interval_ms){: name='offset_flush_interval_ms'} (integer, Minimum: 1, Maximum: 100000000). The interval at which to try committing offsets for tasks (defaults to 60000). 
- [`offset_flush_timeout_ms`](#offset_flush_timeout_ms){: name='offset_flush_timeout_ms'} (integer, Minimum: 1, Maximum: 2147483647). Maximum number of milliseconds to wait for records to flush and partition offset data to be committed to offset storage before cancelling the process and restoring the offset data to be committed in a future attempt (defaults to 5000). 
- [`producer_batch_size`](#producer_batch_size){: name='producer_batch_size'} (integer, Minimum: 0, Maximum: 5242880). This setting gives the upper bound of the batch size to be sent. If there are fewer than this many bytes accumulated for this partition, the producer will 'linger' for the linger.ms time waiting for more records to show up. A batch size of zero will disable batching entirely (defaults to 16384). 
- [`producer_buffer_memory`](#producer_buffer_memory){: name='producer_buffer_memory'} (integer, Minimum: 5242880, Maximum: 134217728). The total bytes of memory the producer can use to buffer records waiting to be sent to the broker (defaults to 33554432). 
- [`producer_compression_type`](#producer_compression_type){: name='producer_compression_type'} (string, Enum: `gzip`, `snappy`, `lz4`, `zstd`, `none`). Specify the default compression type for producers. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'none' which is the default and equivalent to no compression. 
- [`producer_linger_ms`](#producer_linger_ms){: name='producer_linger_ms'} (integer, Minimum: 0, Maximum: 5000). This setting gives the upper bound on the delay for batching: once there is batch.size worth of records for a partition it will be sent immediately regardless of this setting, however if there are fewer than this many bytes accumulated for this partition the producer will 'linger' for the specified time waiting for more records to show up. Defaults to 0. 
- [`producer_max_request_size`](#producer_max_request_size){: name='producer_max_request_size'} (integer, Minimum: 131072, Maximum: 67108864). This setting will limit the number of record batches the producer will send in a single request to avoid sending huge requests. 
- [`session_timeout_ms`](#session_timeout_ms){: name='session_timeout_ms'} (integer, Minimum: 1, Maximum: 2147483647). The timeout in milliseconds used to detect failures when using Kafka’s group management facilities (defaults to 10000). 

### kafka_rest_config {: #kafka_rest_config }

Kafka REST configuration.

**Optional**

- [`consumer_enable_auto_commit`](#consumer_enable_auto_commit){: name='consumer_enable_auto_commit'} (boolean). If true the consumer's offset will be periodically committed to Kafka in the background. 
- [`consumer_request_max_bytes`](#consumer_request_max_bytes){: name='consumer_request_max_bytes'} (integer, Minimum: 0, Maximum: 671088640). Maximum number of bytes in unencoded message keys and values by a single request. 
- [`consumer_request_timeout_ms`](#consumer_request_timeout_ms){: name='consumer_request_timeout_ms'} (integer, Enum: `1000`, `15000`, `30000`, Minimum: 1000, Maximum: 30000). The maximum total time to wait for messages for a request if the maximum number of messages has not yet been reached. 
- [`producer_acks`](#producer_acks){: name='producer_acks'} (string, Enum: `all`, `-1`, `0`, `1`). The number of acknowledgments the producer requires the leader to have received before considering a request complete. If set to 'all' or '-1', the leader will wait for the full set of in-sync replicas to acknowledge the record. 
- [`producer_compression_type`](#producer_compression_type){: name='producer_compression_type'} (string, Enum: `gzip`, `snappy`, `lz4`, `zstd`, `none`). Specify the default compression type for producers. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'none' which is the default and equivalent to no compression. 
- [`producer_linger_ms`](#producer_linger_ms){: name='producer_linger_ms'} (integer, Minimum: 0, Maximum: 5000). Wait for up to the given delay to allow batching records together. 
- [`simpleconsumer_pool_size_max`](#simpleconsumer_pool_size_max){: name='simpleconsumer_pool_size_max'} (integer, Minimum: 10, Maximum: 250). Maximum number of SimpleConsumers that can be instantiated per broker. 

### private_access {: #private_access }

Allow access to selected service ports from private networks.

**Optional**

- [`kafka`](#kafka){: name='kafka'} (boolean). Allow clients to connect to kafka with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`kafka_connect`](#kafka_connect){: name='kafka_connect'} (boolean). Allow clients to connect to kafka_connect with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`kafka_rest`](#kafka_rest){: name='kafka_rest'} (boolean). Allow clients to connect to kafka_rest with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`schema_registry`](#schema_registry){: name='schema_registry'} (boolean). Allow clients to connect to schema_registry with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 

### privatelink_access {: #privatelink_access }

Allow access to selected service components through Privatelink.

**Optional**

- [`jolokia`](#jolokia){: name='jolokia'} (boolean). Enable jolokia. 
- [`kafka`](#kafka){: name='kafka'} (boolean). Enable kafka. 
- [`kafka_connect`](#kafka_connect){: name='kafka_connect'} (boolean). Enable kafka_connect. 
- [`kafka_rest`](#kafka_rest){: name='kafka_rest'} (boolean). Enable kafka_rest. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Enable prometheus. 
- [`schema_registry`](#schema_registry){: name='schema_registry'} (boolean). Enable schema_registry. 

### public_access {: #public_access }

Allow access to selected service ports from the public Internet.

**Optional**

- [`kafka`](#kafka){: name='kafka'} (boolean). Allow clients to connect to kafka from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`kafka_connect`](#kafka_connect){: name='kafka_connect'} (boolean). Allow clients to connect to kafka_connect from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`kafka_rest`](#kafka_rest){: name='kafka_rest'} (boolean). Allow clients to connect to kafka_rest from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`schema_registry`](#schema_registry){: name='schema_registry'} (boolean). Allow clients to connect to schema_registry from the public internet for service nodes that are in a project VPC or another type of private network. 

### schema_registry_config {: #schema_registry_config }

Schema Registry configuration.

**Optional**

- [`leader_eligibility`](#leader_eligibility){: name='leader_eligibility'} (boolean). If true, Karapace / Schema Registry on the service nodes can participate in leader election. It might be needed to disable this when the schemas topic is replicated to a secondary cluster and Karapace / Schema Registry there must not participate in leader election. Defaults to `true`. 
- [`topic_name`](#topic_name){: name='topic_name'} (string, MinLength: 1, MaxLength: 249). The durable single partition topic that acts as the durable log for the data. This topic must be compacted to avoid losing data due to retention policy. Please note that changing this configuration in an existing Schema Registry / Karapace setup leads to previous schemas being inaccessible, data encoded with them potentially unreadable and schema ID sequence put out of order. It's only possible to do the switch while Schema Registry / Karapace is disabled. Defaults to `_schemas`. 

