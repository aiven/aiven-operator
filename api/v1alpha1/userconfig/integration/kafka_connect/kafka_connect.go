// Code generated by user config generator. DO NOT EDIT.
// +kubebuilder:object:generate=true

package kafkaconnectuserconfig

// Kafka Connect service configuration values
type KafkaConnect struct {
	// +kubebuilder:validation:MaxLength=249
	// The name of the topic where connector and task configuration data are stored.This must be the same for all workers with the same group_id.
	ConfigStorageTopic *string `groups:"create,update" json:"config_storage_topic,omitempty"`

	// +kubebuilder:validation:MaxLength=249
	// A unique string that identifies the Connect cluster group this worker belongs to.
	GroupId *string `groups:"create,update" json:"group_id,omitempty"`

	// +kubebuilder:validation:MaxLength=249
	// The name of the topic where connector and task configuration offsets are stored.This must be the same for all workers with the same group_id.
	OffsetStorageTopic *string `groups:"create,update" json:"offset_storage_topic,omitempty"`

	// +kubebuilder:validation:MaxLength=249
	// The name of the topic where connector and task configuration status updates are stored.This must be the same for all workers with the same group_id.
	StatusStorageTopic *string `groups:"create,update" json:"status_storage_topic,omitempty"`
}

// Integration user config
type KafkaConnectUserConfig struct {
	// Kafka Connect service configuration values
	KafkaConnect *KafkaConnect `groups:"create,update" json:"kafka_connect,omitempty"`
}
