// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"github.com/aiven/go-client-codegen/handler/kafkatopic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaTopicSpec defines the desired state of KafkaTopic
type KafkaTopicSpec struct {
	ServiceDependant `json:",inline"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=249
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Topic name. If provided, is used instead of metadata.name.
	// This field supports additional characters, has a longer length,
	// and will replace metadata.name in future releases
	TopicName string `json:"topicName,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000000
	// Number of partitions to create in the topic
	Partitions int `json:"partitions"`

	// +kubebuilder:validation:Minimum=2
	// Replication factor for the topic
	Replication int `json:"replication"`

	// Kafka topic tags
	Tags []KafkaTopicTag `json:"tags,omitempty"`

	// Kafka topic configuration
	Config *KafkaTopicConfig `json:"config,omitempty"`

	// It is a Kubernetes side deletion protections, which prevents the kafka topic
	// from being deleted by Kubernetes. It is recommended to enable this for any production
	// databases containing critical data.
	TerminationProtection *bool `json:"termination_protection,omitempty"`
}

// GetTopicName returns topic name with a backward compatibility.
// metadata.Name is deprecated
func (in *KafkaTopic) GetTopicName() string {
	if in.Spec.TopicName != "" {
		return in.Spec.TopicName
	}
	return in.Name
}

type KafkaTopicTag struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9_-]+$"
	Key string `json:"key"`

	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9_-]+$"
	Value string `json:"value,omitempty"`
}

type KafkaTopicConfig struct {
	// The retention policy to use on old segments. Possible values include 'delete', 'compact', or a comma-separated list of them. The default policy ('delete') will discard old segments when their retention time or size limit has been reached. The 'compact' setting will enable log compaction on the topic.
	CleanupPolicy kafkatopic.CleanupPolicyType `json:"cleanup_policy,omitempty"`

	// Specify the final compression type for a given topic. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'uncompressed' which is equivalent to no compression; and 'producer' which means retain the original compression codec set by the producer.
	CompressionType kafkatopic.CompressionType `json:"compression_type,omitempty"`

	// The amount of time to retain delete tombstone markers for log compacted topics. This setting also gives a bound on the time in which a consumer must complete a read if they begin from offset 0 to ensure that they get a valid snapshot of the final stage (otherwise delete tombstones may be collected before they complete their scan).
	DeleteRetentionMs *int `json:"delete_retention_ms,omitempty"`

	// The time to wait before deleting a file from the filesystem.
	FileDeleteDelayMs *int `json:"file_delete_delay_ms,omitempty"`

	// This setting allows specifying an interval at which we will force an fsync of data written to the log. For example if this was set to 1 we would fsync after every message; if it were 5 we would fsync after every five messages. In general we recommend you not set this and use replication for durability and allow the operating system's background flush capabilities as it is more efficient.
	FlushMessages *int `json:"flush_messages,omitempty"`

	// This setting allows specifying a time interval at which we will force an fsync of data written to the log. For example if this was set to 1000 we would fsync after 1000 ms had passed. In general we recommend you not set this and use replication for durability and allow the operating system's background flush capabilities as it is more efficient.
	FlushMs *int `json:"flush_ms,omitempty"`

	// This setting controls how frequently Kafka adds an index entry to its offset index. The default setting ensures that we index a message roughly every 4096 bytes. More indexing allows reads to jump closer to the exact position in the log but makes the index larger. You probably don't need to change this.
	IndexIntervalBytes *int `json:"index_interval_bytes,omitempty"`

	// Indicates whether inkless should be enabled.
	InklessEnable *bool `json:"inkless_enable,omitempty"`

	// This configuration controls the maximum bytes tiered storage will retain segment files locally before it will discard old log segments to free up space. If set to -2, the limit is equal to overall retention time. If set to -1, no limit is applied but it's possible only if overall retention is also -1.
	LocalRetentionBytes *int `json:"local_retention_bytes,omitempty"`

	// This configuration controls the maximum time tiered storage will retain segment files locally before it will discard old log segments to free up space. If set to -2, the time limit is equal to overall retention time. If set to -1, no time limit is applied but it's possible only if overall retention is also -1.
	LocalRetentionMs *int `json:"local_retention_ms,omitempty"`

	// The maximum time a message will remain ineligible for compaction in the log. Only applicable for logs that are being compacted.
	MaxCompactionLagMs *int `json:"max_compaction_lag_ms,omitempty"`

	// The largest record batch size allowed by Kafka (after compression if compression is enabled). If this is increased and there are consumers older than 0.10.2, the consumers' fetch size must also be increased so that the they can fetch record batches this large. In the latest message format version, records are always grouped into batches for efficiency. In previous message format versions, uncompressed records are not grouped into batches and this limit only applies to a single record in that case.
	MaxMessageBytes *int `json:"max_message_bytes,omitempty"`

	// This configuration controls whether down-conversion of message formats is enabled to satisfy consume requests. When set to false, broker will not perform down-conversion for consumers expecting an older message format. The broker responds with UNSUPPORTED_VERSION error for consume requests from such older clients. This configuration does not apply to any message format conversion that might be required for replication to followers.
	MessageDownconversionEnable *bool `json:"message_downconversion_enable,omitempty"`

	// Specify the message format version the broker will use to append messages to the logs. The value should be a valid ApiVersion. Some examples are: 0.8.2, 0.9.0.0, 0.10.0, check ApiVersion for more details. By setting a particular message format version, the user is certifying that all the existing messages on disk are smaller or equal than the specified version. Setting this value incorrectly will cause consumers with older versions to break as they will receive messages with a format that they don't understand.
	MessageFormatVersion kafkatopic.MessageFormatVersionType `json:"message_format_version,omitempty"`

	// The maximum difference allowed between the timestamp when a broker receives a message and the timestamp specified in the message. If message.timestamp.type=CreateTime, a message will be rejected if the difference in timestamp exceeds this threshold. This configuration is ignored if message.timestamp.type=LogAppendTime.
	MessageTimestampDifferenceMaxMs *int `json:"message_timestamp_difference_max_ms,omitempty"`

	// Define whether the timestamp in the message is message create time or log append time.
	MessageTimestampType kafkatopic.MessageTimestampType `json:"message_timestamp_type,omitempty"`

	// This configuration controls how frequently the log compactor will attempt to clean the log (assuming log compaction is enabled). By default we will avoid cleaning a log where more than 50% of the log has been compacted. This ratio bounds the maximum space wasted in the log by duplicates (at 50% at most 50% of the log could be duplicates). A higher ratio will mean fewer, more efficient cleanings but will mean more wasted space in the log. If the max.compaction.lag.ms or the min.compaction.lag.ms configurations are also specified, then the log compactor considers the log to be eligible for compaction as soon as either: (i) the dirty ratio threshold has been met and the log has had dirty (uncompacted) records for at least the min.compaction.lag.ms duration, or (ii) if the log has had dirty (uncompacted) records for at most the max.compaction.lag.ms period.
	MinCleanableDirtyRatio *float64 `json:"min_cleanable_dirty_ratio,omitempty"`

	// The minimum time a message will remain uncompacted in the log. Only applicable for logs that are being compacted.
	MinCompactionLagMs *int `json:"min_compaction_lag_ms,omitempty"`

	// When a producer sets acks to 'all' (or '-1'), this configuration specifies the minimum number of replicas that must acknowledge a write for the write to be considered successful. If this minimum cannot be met, then the producer will raise an exception (either NotEnoughReplicas or NotEnoughReplicasAfterAppend). When used together, min.insync.replicas and acks allow you to enforce greater durability guarantees. A typical scenario would be to create a topic with a replication factor of 3, set min.insync.replicas to 2, and produce with acks of 'all'. This will ensure that the producer raises an exception if a majority of replicas do not receive a write.
	MinInsyncReplicas *int `json:"min_insync_replicas,omitempty"`

	// True if we should preallocate the file on disk when creating a new log segment.
	Preallocate *bool `json:"preallocate,omitempty"`

	// Indicates whether tiered storage should be enabled.
	RemoteStorageEnable *bool `json:"remote_storage_enable,omitempty"`

	// This configuration controls the maximum size a partition (which consists of log segments) can grow to before we will discard old log segments to free up space if we are using the 'delete' retention policy. By default there is no size limit only a time limit. Since this limit is enforced at the partition level, multiply it by the number of partitions to compute the topic retention in bytes.
	RetentionBytes *int `json:"retention_bytes,omitempty"`

	// This configuration controls the maximum time we will retain a log before we will discard old log segments to free up space if we are using the 'delete' retention policy. This represents an SLA on how soon consumers must read their data. If set to -1, no time limit is applied.
	RetentionMs *int `json:"retention_ms,omitempty"`

	// This configuration controls the segment file size for the log. Retention and cleaning is always done a file at a time so a larger segment size means fewer files but less granular control over retention. Setting this to a very low value has consequences, and the Aiven management plane ignores values less than 10 megabytes.
	SegmentBytes *int `json:"segment_bytes,omitempty"`

	// This configuration controls the size of the index that maps offsets to file positions. We preallocate this index file and shrink it only after log rolls. You generally should not need to change this setting.
	SegmentIndexBytes *int `json:"segment_index_bytes,omitempty"`

	// The maximum random jitter subtracted from the scheduled segment roll time to avoid thundering herds of segment rolling
	SegmentJitterMs *int `json:"segment_jitter_ms,omitempty"`

	// This configuration controls the period of time after which Kafka will force the log to roll even if the segment file isn't full to ensure that retention can delete or compact old data. Setting this to a very low value has consequences, and the Aiven management plane ignores values less than 10 seconds.
	SegmentMs *int `json:"segment_ms,omitempty"`

	// Indicates whether to enable replicas not in the ISR set to be elected as leader as a last resort, even though doing so may result in data loss.
	UncleanLeaderElectionEnable *bool `json:"unclean_leader_election_enable,omitempty"`
}

// KafkaTopicStatus defines the observed state of KafkaTopic
type KafkaTopicStatus struct {
	// Conditions represent the latest available observations of an KafkaTopic state
	Conditions []metav1.Condition `json:"conditions"`

	// State represents the state of the kafka topic
	State kafkatopic.TopicStateType `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaTopic is the Schema for the kafkatopics API
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Partitions",type="string",JSONPath=".spec.partitions"
// +kubebuilder:printcolumn:name="Replication",type="string",JSONPath=".spec.replication"
type KafkaTopic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaTopicSpec   `json:"spec,omitempty"`
	Status KafkaTopicStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &KafkaTopic{}

func (in *KafkaTopic) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *KafkaTopic) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (*KafkaTopic) NoSecret() bool {
	return true
}

// +kubebuilder:object:root=true

// KafkaTopicList contains a list of KafkaTopic
type KafkaTopicList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaTopic `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaTopic{}, &KafkaTopicList{})
}
