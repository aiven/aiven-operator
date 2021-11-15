// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisSpec defines the desired state of Redis
type RedisSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef AuthSecretReference `json:"authSecretRef"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// Redis specific user configuration options
	UserConfig RedisUserConfig `json:"userConfig,omitempty"`
}

type RedisPrivatelinkAccess struct {
	// Enable redis
	Redis *bool `json:"redis,omitempty"`
}

type RedisPublicAccess struct {
	// Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network
	Prometheus *bool `json:"prometheus,omitempty"`

	// Allow clients to connect to redis from the public internet for service nodes that are in a project VPC or another type of private network
	Redis *bool `json:"redis,omitempty"`
}

type RedisMigration struct {
	// +kubebuilder:validation:MaxLength=2048
	// Comma-separated list of databases, which should be ignored during migration (supported by MySQL only at the moment)
	IgnoreDbs string `json:"ignore_dbs,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Password for authentication with the server where to migrate data from
	Password string `json:"password,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// Port number of the server where to migrate data from
	Port *int64 `json:"port,omitempty"`

	// The server where to migrate data from is secured with SSL
	Ssl *bool `json:"ssl,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// User name for authentication with the server where to migrate data from
	Username string `json:"username,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Database name for bootstrapping the initial connection
	Dbname string `json:"dbname,omitempty"`

	// +kubebuilder:validation:MaxLength=255
	// Hostname or IP address of the server where to migrate data from
	Host string `json:"host,omitempty"`
}

type RedisPrivateAccess struct {
	// Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Prometheus *bool `json:"prometheus,omitempty"`

	// Allow clients to connect to redis with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Redis *bool `json:"redis,omitempty"`
}

type RedisUserConfig struct {
	// Migrate data from existing server
	Migration RedisMigration `json:"migration,omitempty"`

	// Allow access to selected service ports from the public internet
	PublicAccess RedisPublicAccess `json:"public_access,omitempty"`

	// Allow access to selected service ports from private networks
	PrivateAccess RedisPrivateAccess `json:"private_access,omitempty"`

	// Allow access to selected service components through Privatelink
	PrivatelinkAccess RedisPrivatelinkAccess `json:"privatelink_access,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Name of another service to fork from. This has effect only when a new service is being created.
	ServiceToForkFrom string `json:"service_to_fork_from,omitempty"`

	// IP filter Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'
	IPFilter []string `json:"ip_filter,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Name of another project to fork a service from. This has effect only when a new service is being created.
	ProjectToForkFrom string `json:"project_to_fork_from,omitempty"`

	// +kubebuilder:validation:Enum=allchannels;resetchannels
	// Default ACL for pub/sub channels used when Redis user is created Determines default pub/sub channels' ACL for new users if ACL is not supplied. When this option is not defined, all_channels is assumed to keep backward compatibility. This option doesn't affect Redis configuration acl-pubsub-default.
	RedisAclChannelsDefault string `json:"redis_acl_channels_default,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=120
	// LFU maxmemory-policy counter decay time in minutes
	RedisLfuDecayTime *int64 `json:"redis_lfu_decay_time,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// Counter logarithm factor for volatile-lfu and allkeys-lfu maxmemory-policies
	RedisLfuLogFactor *int64 `json:"redis_lfu_log_factor,omitempty"`

	// +kubebuilder:validation:Enum=off;rdb
	// Redis persistence When persistence is 'rdb', Redis does RDB dumps each 10 minutes if any key is changed. Also RDB dumps are done according to backup schedule for backup purposes. When persistence is 'off', no RDB dumps and backups are done, so data can be lost at any moment if service is restarted for any reason, or if service is powered off. Also service can't be forked.
	RedisPersistence string `json:"redis_persistence,omitempty"`

	// +kubebuilder:validation:Minimum=32
	// +kubebuilder:validation:Maximum=512
	// Pub/sub client output buffer hard limit in MB Set output buffer limit for pub / sub clients in MB. The value is the hard limit, the soft limit is 1/4 of the hard limit. When setting the limit, be mindful of the available memory in the selected service plan.
	RedisPubsubClientOutputBufferLimit *int64 `json:"redis_pubsub_client_output_buffer_limit,omitempty"`

	// Static IP addresses Use static public IP addresses
	StaticIps *bool `json:"static_ips,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=32
	// Redis IO thread count
	RedisIoThreads *int64 `json:"redis_io_threads,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=31536000
	// Redis idle connection timeout
	RedisTimeout *int64 `json:"redis_timeout,omitempty"`

	// +kubebuilder:validation:Format="^[a-zA-Z0-9-_:.]+$"
	// +kubebuilder:validation:MaxLength=128
	// Name of the basebackup to restore in forked service
	RecoveryBasebackupName string `json:"recovery_basebackup_name,omitempty"`

	// +kubebuilder:validation:Enum=noeviction;allkeys-lru;volatile-lru;allkeys-random;volatile-random;volatile-ttl;volatile-lfu;allkeys-lfu
	// Redis maxmemory-policy
	RedisMaxmemoryPolicy string `json:"redis_maxmemory_policy,omitempty"`

	// +kubebuilder:validation:MaxLength=32
	// Set notify-keyspace-events option
	RedisNotifyKeyspaceEvents string `json:"redis_notify_keyspace_events,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// Number of redis databases Set number of redis databases. Changing this will cause a restart of redis service.
	RedisNumberOfDatabases *int64 `json:"redis_number_of_databases,omitempty"`

	// Require SSL to access Redis
	RedisSsl *bool `json:"redis_ssl,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Redis is the Schema for the redis API
// +kubebuilder:subresource:status
type Redis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisSpec     `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

func (r Redis) AuthSecretRef() AuthSecretReference {
	return r.Spec.AuthSecretRef
}

//+kubebuilder:object:root=true

// RedisList contains a list of Redis
type RedisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Redis `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Redis{}, &RedisList{})
}
