// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KafkaConnectSpec defines the desired state of KafkaConnect
type KafkaConnectSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// x-kubernetes-immutable: true
	// Target project.
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// x-kubernetes-immutable: true
	// Service name.
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=128
	// Subscription plan.
	Plan string `json:"plan,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Cloud the service runs in.
	CloudName string `json:"cloud_name,omitempty"`

	// +kubebuilder:validation:MaxLength=36
	// Identifier of the VPC the service should be in, if any.
	ProjectVPCID string `json:"project_vpc_id,omitempty"`

	// +kubebuilder:validation:Enum=monday;tuesday;wednesday;thursday;friday;saturday;sunday;never
	// Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
	MaintenanceWindowDow string `json:"maintenance_window_dow,omitempty"`

	// +kubebuilder:validation:MaxLength=8
	// Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
	MaintenanceWindowTime string `json:"maintenance_window_time,omitempty"`

	// PostgreSQL specific user configuration options
	KafkaConnectUserConfig KafkaConnectUserConfig `json:"kafka_connect_user_config,omitempty"`
}

// KafkaConnectStatus defines the observed state of KafkaConnect
type KafkaConnectStatus struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Target project.
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service name.
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=128
	// Subscription plan.
	Plan string `json:"plan,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Cloud the service runs in.
	CloudName string `json:"cloud_name,omitempty"`

	// +kubebuilder:validation:MaxLength=36
	// Identifier of the VPC the service should be in, if any.
	ProjectVPCID string `json:"project_vpc_id,omitempty"`

	// +kubebuilder:validation:Enum=monday;tuesday;wednesday;thursday;friday;saturday;sunday;never
	// Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
	MaintenanceWindowDow string `json:"maintenance_window_dow,omitempty"`

	// +kubebuilder:validation:MaxLength=8
	// Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
	MaintenanceWindowTime string `json:"maintenance_window_time,omitempty"`

	// PostgreSQL Service state
	State string `json:"state,omitempty"`

	// URI for connecting to the PostgreSQL service.
	ServiceURI string `json:"service_uri,omitempty"`

	// PostgreSQL hostname
	ServiceHost string `json:"service_host,omitempty"`

	// Username used for connecting to the PostgreSQL service.
	ServiceUsername string `json:"service_username,omitempty"`

	// Password used for connecting to the PostgreSQL service.
	ServicePassword string `json:"service_password,omitempty"`

	// PostgreSQL service port.
	ServicePort int `json:"service_port,omitempty"`

	// Service status
	Status string `json:"status"`

	// PostgreSQL specific user configuration options
	KafkaConnectUserConfig KafkaConnectUserConfig `json:"kafka_connect_user_config,omitempty"`
}

type KafkaConnectUserConfig struct {
	// Defines what client configurations can be overridden by the connector. Default is None
	ConnectorClientConfigOverridePolicy string `json:"additionalProperties,omitempty"`

	// What to do when there is no initial offset in Kafka or if the current offset does not exist any more on the server. Default is earliest
	ConsumerAutoOffsetReset string `json:"consumer_auto_offset_reset,omitempty"`

	// Records are fetched in batches by the consumer, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that the consumer can make progress. As such, this is not a absolute maximum.
	ConsumerFetchMaxBytes *int64 `json:"consumer_fetch_max_bytes,omitempty"`

	// Transaction read isolation level. read_uncommitted is the default, but read_committed can be used if consume-exactly-once behavior is desired.
	ConsumerIsolationLevel string `json:"consumer_isolation_level,omitempty"`

	// Records are fetched in batches by the consumer.If the first record batch in the first non-empty partition of the fetch is larger than this limit, the batch will still be returned to ensure that the consumer can make progress.
	ConsumerMaxPartitionFetchBytes *int64 `json:"consumer_max_partition_fetch_bytes,omitempty"`

	// The maximum delay in milliseconds between invocations of poll() when using consumer group management (defaults to 300000).
	ConsumerMaxPollIntervalMs *int64 `json:"consumer_max_poll_interval_ms,omitempty"`

	// The maximum number of records returned in a single call to poll() (defaults to 500).
	ConsumerMaxPollRecords *int64 `json:"consumer_max_poll_records,omitempty"`

	// The interval at which to try committing offsets for tasks (defaults to 60000).
	OffsetFlushIntervalMs *int64 `json:"offset_flush_interval_ms,omitempty"`

	// This setting will limit the number of record batches the producer will send in a single request to avoid sending huge requests.
	ProducerMaxRequestSize *int64 `json:"producer_max_request_size,omitempty"`

	// The timeout in milliseconds used to detect failures when using Kafka’s group management facilities (defaults to 10000).
	SessionTimeoutMs *int64 `json:"session_timeout_ms,omitempty"`

	// Allow access to selected service ports from private networks
	PrivateAccess KafkaConnectPrivateAccessUserConfig `json:"private_access,omitempty"`

	// Allow access to selected service ports from the public Internet
	PublicAccess KafkaConnectPublicAccessUserConfig `json:"public_access,omitempty"`
}

type KafkaConnectPrivateAccessUserConfig struct {
	// Allow clients to connect to kafka_connect with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	KafkaConnect *bool `json:"kafka_connect,omitempty"`

	// Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Prometheus *bool `json:"prometheus,omitempty"`
}

type KafkaConnectPublicAccessUserConfig struct {
	// Allow clients to connect to kafka_connect from the public internet for service nodes that are in a project VPC or another type of private network
	KafkaConnect *bool `json:"kafka_connect,omitempty"`

	// Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network
	Prometheus *bool `json:"prometheus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaConnect is the Schema for the kafkaconnects API
type KafkaConnect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaConnectSpec   `json:"spec,omitempty"`
	Status KafkaConnectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KafkaConnectList contains a list of KafkaConnect
type KafkaConnectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaConnect `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaConnect{}, &KafkaConnectList{})
}
