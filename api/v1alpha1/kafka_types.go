// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaSpec defines the desired state of Kafka
type KafkaSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Kafka specific user configuration options
	KafkaUserConfig KafkaUserConfig `json:"kafka_user_config,omitempty"`
}

// KafkaStatus defines the observed state of Kafka
type KafkaStatus struct {
	KafkaSpec `json:",inline"`

	// Service state
	State string `json:"state"`
}

type KafkaUserConfig struct {
	// +kubebuilder:validation:Enum="1.0";"1.1";"2.0";"2.1";"2.2";"2.3";"2.4";"2.5";"2.6";"2.7"
	// Kafka major version
	KafkaVersion string `json:"kafka_version,omitempty"`

	// Enable Schema-Registry service
	SchemaRegistry *bool `json:"schema_registry,omitempty"`

	// Kafka broker configuration values
	Kafka KafkaSubKafkaUserConfig `json:"kafka,omitempty"`

	// Kafka Connect configuration values
	KafkaConnectConfig KafkServiceKafkaConnectUserConfig `json:"kafka_connect_user_config,omitempty"`

	// Allow access to selected service ports from private networks
	PrivateAccess KafkaPrivateAccessUserConfig `json:"private_access,omitempty"`

	// Schema Registry configuration
	SchemaRegistryConfig KafkaSchemaRegistryConfig `json:"schema_registry_config,omitempty"`

	// IP filter Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'
	IPFilter []string `json:"ip_filter,omitempty"`

	// Kafka authentication methods
	KafkaAuthenticationMethods KafkaAuthenticationMethodsUserConfig `json:"kafka_authentication_methods,omitempty"`

	// Enable Kafka Connect service
	KafkaConnect *bool `json:"kafka_connect,omitempty"`

	// Enable Kafka-REST service
	KafkaRest *bool `json:"kafka_rest,omitempty"`

	// Kafka REST configuration
	KafkaRestConfig KafkaRestUserConfig `json:"kafka_rest_config,omitempty"`
}

type KafkaPrivateAccessUserConfig struct {
	// Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Prometheus *bool `json:"prometheus,omitempty"`
}

type KafkaRestUserConfig struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=671088640
	// consumer.request.max.bytes Maximum number of bytes in unencoded message keys and values by a single request
	ConsumerRequestMaxBytes *int64 `json:"consumer_request_max_bytes,omitempty"`

	// +kubebuilder:validation:Minimum=1000
	// +kubebuilder:validation:Maximum=30000
	// +kubebuilder:validation:Enum=1000;15000;30000
	// consumer.request.timeout.ms The maximum total time to wait for messages for a request if the maximum number of messages has not yet been reached
	ConsumerRequestTimeoutMs *int64 `json:"consumer_request_timeout_ms,omitempty"`

	// +kubebuilder:validation:Enum=all;-1;0;1
	// producer.acks The number of acknowledgments the producer requires the leader to have received before considering a request complete. If set to 'all' or '-1', the leader will wait for the full set of in-sync replicas to acknowledge the record.
	ProducerAcks string `json:"producer_acks,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=5000
	// producer.linger.ms Wait for up to the given delay to allow batching records together
	ProducerLingerMs *int64 `json:"producer_linger_ms,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=250
	// simpleconsumer.pool.size.max Maximum number of SimpleConsumers that can be instantiated per broker
	SimpleconsumerPoolSizeMax *int64 `json:"simpleconsumer_pool_size_max,omitempty"`

	// consumer.enable.auto.commit If true the consumer's offset will be periodically committed to Kafka in the background
	ConsumerEnableAutoCommit *bool `json:"consumer_enable_auto_commit,omitempty"`

	// Allow access to selected service ports from the public Internet
	PublicAccess KafkaPublicAccessUserConfig `json:"public_access,omitempty"`

	// +kubebuilder:validation:MaxLength=255
	// Custom domain Serve the web frontend using a custom CNAME pointing to the Aiven DNS name
	CustomDomain string `json:"custom_domain,omitempty"`
}

type KafkaPublicAccessUserConfig struct {
	// Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network
	Prometheus *bool `json:"prometheus,omitempty"`

	// Allow clients to connect to schema_registry from the public internet for service nodes that are in a project VPC or another type of private network
	SchemaRegistry *bool `json:"schema_registry,omitempty"`

	// Allow clients to connect to kafka from the public internet for service nodes that are in a project VPC or another type of private network
	Kafka *bool `json:"kafka,omitempty"`

	// Allow clients to connect to kafka_connect from the public internet for service nodes that are in a project VPC or another type of private network
	KafkaConnect *bool `json:"kafka_connect,omitempty"`

	// Allow clients to connect to kafka_rest from the public internet for service nodes that are in a project VPC or another type of private network
	KafkaRest *bool `json:"kafka_rest,omitempty"`
}

type KafkaAuthenticationMethodsUserConfig struct {
	// Enable certificate/SSL authentication
	Certificate *bool `json:"certificate,omitempty"`

	// Enable SASL authentication
	Sasl *bool `json:"sasl,omitempty"`
}

type KafkaSchemaRegistryConfig struct {
	// leader_eligibility If true, Karapace / Schema Registry on the service nodes can participate in leader election. It might be needed to disable this when the schemas topic is replicated to a secondary cluster and Karapace / Schema Registry there must not participate in leader election. Defaults to 'true'.
	LeaderEligibility *bool `json:"leader_eligibility,omitempty"`

	// +kubebuilder:validation:MaxLength=249
	// topic_name The durable single partition topic that acts as the durable log for the data. This topic must be compacted to avoid losing data due to retention policy. Please note that changing this configuration in an existing Schema Registry / Karapace setup leads to previous schemas being inaccessible, data encoded with them potentially unreadable and schema ID sequence put out of order. It's only possible to do the switch while Schema Registry / Karapace is disabled. Defaults to '_schemas'.
	TopicName string `json:"topic_name,omitempty"`
}

type KafkServiceKafkaConnectUserConfig struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	// The maximum number of records returned by a single poll The maximum number of records returned in a single call to poll() (defaults to 500).
	ConsumerMaxPollRecords *int64 `json:"consumer_max_poll_records,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2147483647
	// Offset flush timeout Maximum number of milliseconds to wait for records to flush and partition offset data to be committed to offset storage before cancelling the process and restoring the offset data to be committed in a future attempt (defaults to 5000).
	OffsetFlushTimeoutMs *int64 `json:"offset_flush_timeout_ms,omitempty"`

	// +kubebuilder:validation:Enum=None;All
	// Client config override policy Defines what client configurations can be overridden by the connector. Default is None
	ConnectorClientConfigOverridePolicy string `json:"connector_client_config_override_policy,omitempty"`

	// +kubebuilder:validation:Minimum=1048576
	// +kubebuilder:validation:Maximum=104857600
	// The maximum amount of data the server should return for a fetch request Records are fetched in batches by the consumer, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that the consumer can make progress. As such, this is not a absolute maximum.
	ConsumerFetchMaxBytes *int64 `json:"consumer_fetch_max_bytes,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2147483647
	// The maximum delay between polls when using consumer group management The maximum delay in milliseconds between invocations of poll() when using consumer group management (defaults to 300000).
	ConsumerMaxPollIntervalMs *int64 `json:"consumer_max_poll_interval_ms,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100000000
	// The interval at which to try committing offsets for tasks The interval at which to try committing offsets for tasks (defaults to 60000).
	OffsetFlushIntervalMs *int64 `json:"offset_flush_interval_ms,omitempty"`

	// +kubebuilder:validation:Minimum=131072
	// +kubebuilder:validation:Maximum=10485760
	// The maximum size of a request in bytes This setting will limit the number of record batches the producer will send in a single request to avoid sending huge requests.
	ProducerMaxRequestSize *int64 `json:"producer_max_request_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2147483647
	// The timeout used to detect failures when using Kafka’s group management facilities The timeout in milliseconds used to detect failures when using Kafka’s group management facilities (defaults to 10000).
	SessionTimeoutMs *int64 `json:"session_timeout_ms,omitempty"`

	// +kubebuilder:validation:Enum=earliest;latest
	// Consumer auto offset reset What to do when there is no initial offset in Kafka or if the current offset does not exist any more on the server. Default is earliest
	ConsumerAutoOffsetReset string `json:"consumer_auto_offset_reset,omitempty"`

	// +kubebuilder:validation:Enum=read_uncommitted;read_committed
	// Consumer isolation level Transaction read isolation level. read_uncommitted is the default, but read_committed can be used if consume-exactly-once behavior is desired.
	ConsumerIsolationLevel string `json:"consumer_isolation_level,omitempty"`

	// +kubebuilder:validation:Minimum=1048576
	// +kubebuilder:validation:Maximum=104857600
	// The maximum amount of data per-partition the server will return. Records are fetched in batches by the consumer.If the first record batch in the first non-empty partition of the fetch is larger than this limit, the batch will still be returned to ensure that the consumer can make progress.
	ConsumerMaxPartitionFetchBytes *int64 `json:"consumer_max_partition_fetch_bytes,omitempty"`
}

type KafkaSubKafkaUserConfig struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100001200
	// message.max.bytes The maximum size of message that the server can receive.
	MessageMaxBytes *int64 `json:"message_max_bytes,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// default.replication.factor Replication factor for autocreated topics
	DefaultReplicationFactor *int64 `json:"default_replication_factor,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1
	// log.cleaner.min.cleanable.ratio Controls log compactor frequency. Larger value means more frequent compactions but also more space wasted for logs. Consider setting log.cleaner.max.compaction.lag.ms to enforce compactions sooner, instead of setting a very high value for this option.
	LogCleanerMinCleanableRatio *int64 `json:"log_cleaner_min_cleanable_ratio,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=104857600
	// log.index.interval.bytes The interval with which Kafka adds an entry to the offset index
	LogIndexIntervalBytes *int64 `json:"log_index_interval_bytes,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=3600000
	// log.segment.delete.delay.ms The amount of time to wait before deleting a file from the filesystem
	LogSegmentDeleteDelayMs *int64 `json:"log_segment_delete_delay_ms,omitempty"`

	// +kubebuilder:validation:Minimum=1000
	// +kubebuilder:validation:Maximum=10000
	// max.incremental.fetch.session.cache.slots The maximum number of incremental fetch sessions that the broker will maintain.
	MaxIncrementalFetchSessionCacheSlots *int64 `json:"max_incremental_fetch_session_cache_slots,omitempty"`

	// +kubebuilder:validation:Minimum=10485760
	// +kubebuilder:validation:Maximum=209715200
	// socket.request.max.bytes The maximum number of bytes in a socket request (defaults to 104857600).
	SocketRequestMaxBytes *int64 `json:"socket_request_max_bytes,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=315569260000
	// log.cleaner.delete.retention.ms How long are delete records retained?
	LogCleanerDeleteRetentionMs *int64 `json:"log_cleaner_delete_retention_ms,omitempty"`

	// +kubebuilder:validation:Minimum=1048576
	// +kubebuilder:validation:Maximum=104857600
	// log.index.size.max.bytes The maximum size in bytes of the offset index
	LogIndexSizeMaxBytes *int64 `json:"log_index_size_max_bytes,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// log.roll.jitter.ms The maximum jitter to subtract from logRollTimeMillis (in milliseconds). If not set, the value in log.roll.jitter.hours is used
	LogRollJitterMs *int64 `json:"log_roll_jitter_ms,omitempty"`

	// +kubebuilder:validation:Minimum=256
	// +kubebuilder:validation:Maximum=2147483647
	// max.connections.per.ip The maximum number of connections allowed from each ip address (defaults to 2147483647).
	MaxConnectionsPerIP *int64 `json:"max_connections_per_ip,omitempty"`

	// +kubebuilder:validation:Minimum=10485760
	// +kubebuilder:validation:Maximum=1048576000
	// replica.fetch.response.max.bytes Maximum bytes expected for the entire fetch response (defaults to 10485760). Records are fetched in batches, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made. As such, this is not an absolute maximum.
	ReplicaFetchResponseMaxBytes *int64 `json:"replica_fetch_response_max_bytes,omitempty"`

	// auto.create.topics.enable Enable auto creation of topics
	AutoCreateTopicsEnable *bool `json:"auto_create_topics_enable,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// log.flush.interval.ms The maximum time in ms that a message in any topic is kept in memory before flushed to disk. If not set, the value in log.flush.scheduler.interval.ms is used
	LogFlushIntervalMs *int64 `json:"log_flush_interval_ms,omitempty"`

	// log.message.downconversion.enable This configuration controls whether down-conversion of message formats is enabled to satisfy consume requests.
	LogMessageDownconversionEnable *bool `json:"log_message_downconversion_enable,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// log.roll.ms The maximum time before a new log segment is rolled out (in milliseconds).
	LogRollMs *int64 `json:"log_roll_ms,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// log.cleaner.min.compaction.lag.ms The minimum time a message will remain uncompacted in the log. Only applicable for logs that are being compacted.
	LogCleanerMinCompactionLagMs *int64 `json:"log_cleaner_min_compaction_lag_ms,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// log.message.timestamp.difference.max.ms The maximum difference allowed between the timestamp when a broker receives a message and the timestamp specified in the message
	LogMessageTimestampDifferenceMaxMs *int64 `json:"log_message_timestamp_difference_max_ms,omitempty"`

	// +kubebuilder:validation:Enum=CreateTime;LogAppendTime
	// log.message.timestamp.type Define whether the timestamp in the message is message create time or log append time.
	LogMessageTimestampType string `json:"log_message_timestamp_type,omitempty"`

	// log.retention.ms The number of milliseconds to keep a log file before deleting it (in milliseconds), If not set, the value in log.retention.minutes is used. If set to -1, no time limit is applied.
	LogRetentionMs *int64 `json:"log_retention_ms,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=60000
	// group.min.session.timeout.ms The minimum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures.
	GroupMinSessionTimeoutMs *int64 `json:"group_min_session_timeout_ms,omitempty"`

	// +kubebuilder:validation:Minimum=10485760
	// +kubebuilder:validation:Maximum=1073741824
	// log.segment.bytes The maximum size of a single log file
	LogSegmentBytes *int64 `json:"log_segment_bytes,omitempty"`

	// +kubebuilder:validation:Enum=gzip;snappy;lz4;zstd;uncompressed;producer
	// compression.type Specify the final compression type for a given topic. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'uncompressed' which is equivalent to no compression; and 'producer' which means retain the original compression codec set by the producer.
	CompressionType string `json:"compression_type,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1800000
	// group.max.session.timeout.ms The maximum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures.
	GroupMaxSessionTimeoutMs *int64 `json:"group_max_session_timeout_ms,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// log.flush.interval.messages The number of messages accumulated on a log partition before messages are flushed to disk
	LogFlushIntervalMessages *int64 `json:"log_flush_interval_messages,omitempty"`

	// log.preallocate Should pre allocate file when create new segment?
	LogPreallocate *bool `json:"log_preallocate,omitempty"`

	// log.retention.bytes The maximum size of the log before deleting messages
	LogRetentionBytes *int64 `json:"log_retention_bytes,omitempty"`

	// +kubebuilder:validation:Minimum=30000
	// log.cleaner.max.compaction.lag.ms The maximum amount of time message will remain uncompacted. Only applicable for logs that are being compacted
	LogCleanerMaxCompactionLagMs *int64 `json:"log_cleaner_max_compaction_lag_ms,omitempty"`

	// +kubebuilder:validation:Maximum=2147483647
	// log.retention.hours The number of hours to keep a log file before deleting it
	LogRetentionHours *int64 `json:"log_retention_hours,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=7
	// min.insync.replicas When a producer sets acks to 'all' (or '-1'), min.insync.replicas specifies the minimum number of replicas that must acknowledge a write for the write to be considered successful.
	MinInsyncReplicas *int64 `json:"min_insync_replicas,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	// num.partitions Number of partitions for autocreated topics
	NumPartitions *int64 `json:"num_partitions,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2147483647
	// offsets.retention.minutes Log retention window in minutes for offsets topic
	OffsetsRetentionMinutes *int64 `json:"offsets_retention_minutes,omitempty"`

	// +kubebuilder:validation:Minimum=1000
	// +kubebuilder:validation:Maximum=3600000
	// connections.max.idle.ms Idle connections timeout: the server socket processor threads close the connections that idle for longer than this.
	ConnectionsMaxIdleMs *int64 `json:"connections_max_idle_ms,omitempty"`

	// +kubebuilder:validation:Enum=compact;delete
	// log.cleanup.policy The default cleanup policy for segments beyond the retention window
	LogCleanupPolicy string `json:"log_cleanup_policy,omitempty"`
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=10000
	// producer.purgatory.purge.interval.requests The purge interval (in number of requests) of the producer request purgatory(defaults to 1000).
	ProducerPurgatoryPurgeIntervalRequests *int64 `json:"producer_purgatory_purge_interval_requests,omitempty"`

	// +kubebuilder:validation:Minimum=1048576
	// +kubebuilder:validation:Maximum=104857600
	// replica.fetch.max.bytes The number of bytes of messages to attempt to fetch for each partition (defaults to 1048576). This is not an absolute maximum, if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made.
	ReplicaFetchMaxBytes *int64 `json:"replica_fetch_max_bytes,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Kafka is the Schema for the kafkas API
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSpec   `json:"spec,omitempty"`
	Status KafkaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KafkaList contains a list of Kafka
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kafka `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kafka{}, &KafkaList{})
}
