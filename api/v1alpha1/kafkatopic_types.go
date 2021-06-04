// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaTopicSpec defines the desired state of KafkaTopic
type KafkaTopicSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Target project.
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service name.
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=63
	// Topic name.
	TopicName string `json:"topic_name"`

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
	Config KafkaTopicConfig `json:"config,omitempty"`

	// It is a Kubernetes side deletion protections, which prevents the kafka topic
	// from being deleted by Kubernetes. It is recommended to enable this for any production
	// databases containing critical data.
	TerminationProtection bool `json:"termination_protection,omitempty"`
}

type KafkaTopicTag struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	Key string `json:"key"`

	// +kubebuilder:validation:MaxLength=256
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	Value string `json:"value,omitempty"`
}

type KafkaTopicConfig struct {
	// cleanup.policy value
	CleanupPolicy string `json:"cleanup_policy,omitempty"`

	// compression.type value
	CompressionType string `json:"compression_type,omitempty"`

	// delete.retention.ms value
	DeleteRetentionMs *int64 `json:"delete_retention_ms,omitempty"`

	// file.delete.delay.ms value
	FileDeleteDelayMs *int64 `json:"file_delete_delay_ms,omitempty"`

	// flush.messages value
	FlushMessages *int64 `json:"flush_messages,omitempty"`

	// flush.ms value
	FlushMs *int64 `json:"flush_ms,omitempty"`

	// index.interval.bytes value
	IndexIntervalBytes *int64 `json:"index_interval_bytes,omitempty"`

	// max.compaction.lag.ms value
	MaxCompactionLagMs *int64 `json:"max_compaction_lag_ms,omitempty"`

	// max.message.bytes value
	MaxMessageBytes *int64 `json:"max_message_bytes,omitempty"`

	// message.downconversion.enable value
	MessageDownconversionEnable *bool `json:"message_downconversion_enable,omitempty"`

	// message.format.version value
	MessageFormatVersion string `json:"message_format_version,omitempty"`

	// message.timestamp.difference.max.ms value
	MessageTimestampDifferenceMaxMs *int64 `json:"message_timestamp_difference_max_ms,omitempty"`

	// message.timestamp.type value
	MessageTimestampType string `json:"message_timestamp_type,omitempty"`

	// TODO: float is currently not supported by K8s provider SDK, but will be in the next versions
	// min.cleanable.dirty.ratio value
	//MinCleanableDirtyRatio *float64 `json:"min_cleanable_dirty_ratio,omitempty"`

	// min.compaction.lag.ms value
	MinCompactionLagMs *int64 `json:"min_compaction_lag_ms,omitempty"`

	// min.insync.replicas value
	MinInsyncReplicas *int64 `json:"min_insync_replicas,omitempty"`

	// preallocate value
	Preallocate *bool `json:"preallocate,omitempty"`

	// retention.bytes value
	RetentionBytes *int64 `json:"retention_bytes,omitempty"`

	// retention.ms value
	RetentionMs *int64 `json:"retention_ms,omitempty"`

	// segment.bytes value
	SegmentBytes *int64 `json:"segment_bytes,omitempty"`

	// segment.index.bytes value
	SegmentIndexBytes *int64 `json:"segment_index_bytes,omitempty"`

	// segment.jitter.ms value
	SegmentJitterMs *int64 `json:"segment_jitter_ms,omitempty"`

	// segment.ms value
	SegmentMs *int64 `json:"segment_ms,omitempty"`

	// unclean.leader.election.enable value
	UncleanLeaderElectionEnable *bool `json:"unclean_leader_election_enable,omitempty"`
}

// KafkaTopicStatus defines the observed state of KafkaTopic
type KafkaTopicStatus struct {
	KafkaTopicSpec `json:",inline"`

	State string `json:"state"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaTopic is the Schema for the kafkatopics API
type KafkaTopic struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaTopicSpec   `json:"spec,omitempty"`
	Status KafkaTopicStatus `json:"status,omitempty"`
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
