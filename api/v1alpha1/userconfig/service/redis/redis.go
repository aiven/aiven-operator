// Code generated by user config generator. DO NOT EDIT.
// +kubebuilder:object:generate=true

package redisuserconfig

// CIDR address block, either as a string, or in a dict with an optional description field
type IpFilter struct {
	// +kubebuilder:validation:MaxLength=1024
	// Description for IP filter list entry
	Description *string `groups:"create,update" json:"description,omitempty"`

	// +kubebuilder:validation:MaxLength=43
	// CIDR address block
	Network string `groups:"create,update" json:"network"`
}

// Migrate data from existing server
type Migration struct {
	// +kubebuilder:validation:MaxLength=63
	// Database name for bootstrapping the initial connection
	Dbname *string `groups:"create,update" json:"dbname,omitempty"`

	// +kubebuilder:validation:MaxLength=255
	// Hostname or IP address of the server where to migrate data from
	Host string `groups:"create,update" json:"host"`

	// +kubebuilder:validation:MaxLength=2048
	// Comma-separated list of databases, which should be ignored during migration (supported by MySQL and PostgreSQL only at the moment)
	IgnoreDbs *string `groups:"create,update" json:"ignore_dbs,omitempty"`

	// +kubebuilder:validation:MaxLength=2048
	// Comma-separated list of database roles, which should be ignored during migration (supported by PostgreSQL only at the moment)
	IgnoreRoles *string `groups:"create,update" json:"ignore_roles,omitempty"`

	// +kubebuilder:validation:Enum="dump";"replication"
	// The migration method to be used (currently supported only by Redis, Dragonfly, MySQL and PostgreSQL service types)
	Method *string `groups:"create,update" json:"method,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Password for authentication with the server where to migrate data from
	Password *string `groups:"create,update" json:"password,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// Port number of the server where to migrate data from
	Port int `groups:"create,update" json:"port"`

	// The server where to migrate data from is secured with SSL
	Ssl *bool `groups:"create,update" json:"ssl,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// User name for authentication with the server where to migrate data from
	Username *string `groups:"create,update" json:"username,omitempty"`
}

// Allow access to selected service ports from private networks
type PrivateAccess struct {
	// Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Prometheus *bool `groups:"create,update" json:"prometheus,omitempty"`

	// Allow clients to connect to redis with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Redis *bool `groups:"create,update" json:"redis,omitempty"`
}

// Allow access to selected service components through Privatelink
type PrivatelinkAccess struct {
	// Enable prometheus
	Prometheus *bool `groups:"create,update" json:"prometheus,omitempty"`

	// Enable redis
	Redis *bool `groups:"create,update" json:"redis,omitempty"`
}

// Allow access to selected service ports from the public Internet
type PublicAccess struct {
	// Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network
	Prometheus *bool `groups:"create,update" json:"prometheus,omitempty"`

	// Allow clients to connect to redis from the public internet for service nodes that are in a project VPC or another type of private network
	Redis *bool `groups:"create,update" json:"redis,omitempty"`
}
type RedisUserConfig struct {
	// +kubebuilder:validation:MaxItems=1
	// Additional Cloud Regions for Backup Replication
	AdditionalBackupRegions []string `groups:"create,update" json:"additional_backup_regions,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=23
	// The hour of day (in UTC) when backup for the service is started. New backup is only started if previous backup has already completed.
	BackupHour *int `groups:"create,update" json:"backup_hour,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=59
	// The minute of an hour when backup for the service is started. New backup is only started if previous backup has already completed.
	BackupMinute *int `groups:"create,update" json:"backup_minute,omitempty"`

	// +kubebuilder:validation:MaxItems=8000
	// Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'
	IpFilter []*IpFilter `groups:"create,update" json:"ip_filter,omitempty"`

	// Migrate data from existing server
	Migration *Migration `groups:"create,update" json:"migration,omitempty"`

	// Allow access to selected service ports from private networks
	PrivateAccess *PrivateAccess `groups:"create,update" json:"private_access,omitempty"`

	// Allow access to selected service components through Privatelink
	PrivatelinkAccess *PrivatelinkAccess `groups:"create,update" json:"privatelink_access,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z][-a-z0-9]{0,63}$|^$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Name of another project to fork a service from. This has effect only when a new service is being created.
	ProjectToForkFrom *string `groups:"create" json:"project_to_fork_from,omitempty"`

	// Allow access to selected service ports from the public Internet
	PublicAccess *PublicAccess `groups:"create,update" json:"public_access,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9-_:.]+$`
	// Name of the basebackup to restore in forked service
	RecoveryBasebackupName *string `groups:"create,update" json:"recovery_basebackup_name,omitempty"`

	// +kubebuilder:validation:Enum="allchannels";"resetchannels"
	// Determines default pub/sub channels' ACL for new users if ACL is not supplied. When this option is not defined, all_channels is assumed to keep backward compatibility. This option doesn't affect Redis configuration acl-pubsub-default.
	RedisAclChannelsDefault *string `groups:"create,update" json:"redis_acl_channels_default,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=32
	// Set Redis IO thread count. Changing this will cause a restart of the Redis service.
	RedisIoThreads *int `groups:"create,update" json:"redis_io_threads,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=120
	// LFU maxmemory-policy counter decay time in minutes
	RedisLfuDecayTime *int `groups:"create,update" json:"redis_lfu_decay_time,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	// Counter logarithm factor for volatile-lfu and allkeys-lfu maxmemory-policies
	RedisLfuLogFactor *int `groups:"create,update" json:"redis_lfu_log_factor,omitempty"`

	// +kubebuilder:validation:Enum="allkeys-lfu";"allkeys-lru";"allkeys-random";"noeviction";"volatile-lfu";"volatile-lru";"volatile-random";"volatile-ttl"
	// Redis maxmemory-policy
	RedisMaxmemoryPolicy *string `groups:"create,update" json:"redis_maxmemory_policy,omitempty"`

	// +kubebuilder:validation:MaxLength=32
	// +kubebuilder:validation:Pattern=`^[KEg\$lshzxentdmA]*$`
	// Set notify-keyspace-events option
	RedisNotifyKeyspaceEvents *string `groups:"create,update" json:"redis_notify_keyspace_events,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// Set number of Redis databases. Changing this will cause a restart of the Redis service.
	RedisNumberOfDatabases *int `groups:"create,update" json:"redis_number_of_databases,omitempty"`

	// +kubebuilder:validation:Enum="off";"rdb"
	// When persistence is 'rdb', Redis does RDB dumps each 10 minutes if any key is changed. Also RDB dumps are done according to the backup schedule for backup purposes. When persistence is 'off', no RDB dumps or backups are done, so data can be lost at any moment if the service is restarted for any reason, or if the service is powered off. Also, the service can't be forked.
	RedisPersistence *string `groups:"create,update" json:"redis_persistence,omitempty"`

	// +kubebuilder:validation:Minimum=32
	// +kubebuilder:validation:Maximum=512
	// Set output buffer limit for pub / sub clients in MB. The value is the hard limit, the soft limit is 1/4 of the hard limit. When setting the limit, be mindful of the available memory in the selected service plan.
	RedisPubsubClientOutputBufferLimit *int `groups:"create,update" json:"redis_pubsub_client_output_buffer_limit,omitempty"`

	// Require SSL to access Redis
	RedisSsl *bool `groups:"create,update" json:"redis_ssl,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2073600
	// Redis idle connection timeout in seconds
	RedisTimeout *int `groups:"create,update" json:"redis_timeout,omitempty"`

	// +kubebuilder:validation:Enum="7.0"
	// Redis major version
	RedisVersion *string `groups:"create,update" json:"redis_version,omitempty"`

	// Store logs for the service so that they are available in the HTTP API and console.
	ServiceLog *bool `groups:"create,update" json:"service_log,omitempty"`

	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`^[a-z][-a-z0-9]{0,63}$|^$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Name of another service to fork from. This has effect only when a new service is being created.
	ServiceToForkFrom *string `groups:"create" json:"service_to_fork_from,omitempty"`

	// Use static public IP addresses
	StaticIps *bool `groups:"create,update" json:"static_ips,omitempty"`
}
