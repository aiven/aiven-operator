// Code generated by user config generator. DO NOT EDIT.
// +kubebuilder:object:generate=true

package opensearchuserconfig

import "encoding/json"

// IndexPatterns Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like 'logs.?' and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note 'logs.?' does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.
type IndexPatterns struct {
	// +kubebuilder:validation:Minimum=0
	// MaxIndexCount Maximum number of indexes to keep
	MaxIndexCount int `groups:"create,update" json:"max_index_count"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9-_.*?]+$`
	// Pattern fnmatch pattern
	Pattern string `groups:"create,update" json:"pattern"`

	// +kubebuilder:validation:Enum=alphabetical;creation_date
	// SortingAlgorithm Deletion sorting algorithm
	SortingAlgorithm *string `groups:"create,update" json:"sorting_algorithm,omitempty"`
}

// IndexTemplate Template settings for all new indexes
type IndexTemplate struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100000
	// MappingNestedObjectsLimit The maximum number of nested JSON objects that a single document can contain across all nested types. This limit helps to prevent out of memory errors when a document contains too many nested objects. Default is 10000.
	MappingNestedObjectsLimit *int `groups:"create,update" json:"mapping_nested_objects_limit,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=29
	// NumberOfReplicas The number of replicas each primary shard has.
	NumberOfReplicas *int `groups:"create,update" json:"number_of_replicas,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1024
	// NumberOfShards The number of primary shards that an index should have.
	NumberOfShards *int `groups:"create,update" json:"number_of_shards,omitempty"`
}

func (ip *IpFilter) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		return nil
	}

	var s string
	err := json.Unmarshal(data, &s)
	if err == nil {
		ip.Network = s
		return nil
	}

	type this struct {
		Network     string  `json:"network"`
		Description *string `json:"description,omitempty" `
	}

	var t *this
	err = json.Unmarshal(data, &t)
	if err != nil {
		return err
	}
	ip.Network = t.Network
	ip.Description = t.Description
	return nil
}

// IpFilter CIDR address block, either as a string, or in a dict with an optional description field
type IpFilter struct {
	// +kubebuilder:validation:MaxLength=1024
	// Description for IP filter list entry
	Description *string `groups:"create,update" json:"description,omitempty"`

	// +kubebuilder:validation:MaxLength=43
	// Network CIDR address block
	Network string `groups:"create,update" json:"network"`
}

// Opensearch OpenSearch settings
type Opensearch struct {
	// ActionAutoCreateIndexEnabled Explicitly allow or block automatic creation of indices. Defaults to true
	ActionAutoCreateIndexEnabled *bool `groups:"create,update" json:"action_auto_create_index_enabled,omitempty"`

	// ActionDestructiveRequiresName Require explicit index names when deleting
	ActionDestructiveRequiresName *bool `groups:"create,update" json:"action_destructive_requires_name,omitempty"`

	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=10000
	// ClusterMaxShardsPerNode Controls the number of shards allowed in the cluster per data node
	ClusterMaxShardsPerNode *int `groups:"create,update" json:"cluster_max_shards_per_node,omitempty"`

	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=16
	// ClusterRoutingAllocationNodeConcurrentRecoveries How many concurrent incoming/outgoing shard recoveries (normally replicas) are allowed to happen on a node. Defaults to 2.
	ClusterRoutingAllocationNodeConcurrentRecoveries *int `groups:"create,update" json:"cluster_routing_allocation_node_concurrent_recoveries,omitempty"`

	// +kubebuilder:validation:MaxLength=40
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9-_]+$`
	// EmailSenderName Sender email name placeholder to be used in Opensearch Dashboards and Opensearch keystore
	EmailSenderName *string `groups:"create,update" json:"email_sender_name,omitempty"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[^\x00-\x1F]+$`
	// EmailSenderPassword Sender email password for Opensearch alerts to authenticate with SMTP server
	EmailSenderPassword *string `groups:"create,update" json:"email_sender_password,omitempty"`

	// +kubebuilder:validation:MaxLength=320
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9_\-\.+\'&]+@(([\da-zA-Z])([_\w-]{,62})\.){,127}(([\da-zA-Z])[_\w-]{,61})?([\da-zA-Z]\.((xn\-\-[a-zA-Z\d]+)|([a-zA-Z\d]{2,})))$`
	// EmailSenderUsername Sender email address for Opensearch alerts
	EmailSenderUsername *string `groups:"create,update" json:"email_sender_username,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2147483647
	// HttpMaxContentLength Maximum content length for HTTP requests to the OpenSearch HTTP API, in bytes.
	HttpMaxContentLength *int `groups:"create,update" json:"http_max_content_length,omitempty"`

	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=262144
	// HttpMaxHeaderSize The max size of allowed headers, in bytes
	HttpMaxHeaderSize *int `groups:"create,update" json:"http_max_header_size,omitempty"`

	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=65536
	// HttpMaxInitialLineLength The max length of an HTTP URL, in bytes
	HttpMaxInitialLineLength *int `groups:"create,update" json:"http_max_initial_line_length,omitempty"`

	// +kubebuilder:validation:Minimum=3
	// +kubebuilder:validation:Maximum=100
	// IndicesFielddataCacheSize Relative amount. Maximum amount of heap memory used for field data cache. This is an expert setting; decreasing the value too much will increase overhead of loading field data; too much memory used for field data cache will decrease amount of heap available for other operations.
	IndicesFielddataCacheSize *int `groups:"create,update" json:"indices_fielddata_cache_size,omitempty"`

	// +kubebuilder:validation:Minimum=3
	// +kubebuilder:validation:Maximum=40
	// IndicesMemoryIndexBufferSize Percentage value. Default is 10%. Total amount of heap used for indexing buffer, before writing segments to disk. This is an expert setting. Too low value will slow down indexing; too high value will increase indexing performance but causes performance issues for query performance.
	IndicesMemoryIndexBufferSize *int `groups:"create,update" json:"indices_memory_index_buffer_size,omitempty"`

	// +kubebuilder:validation:Minimum=3
	// +kubebuilder:validation:Maximum=40
	// IndicesQueriesCacheSize Percentage value. Default is 10%. Maximum amount of heap used for query cache. This is an expert setting. Too low value will decrease query performance and increase performance for other operations; too high value will cause issues with other OpenSearch functionality.
	IndicesQueriesCacheSize *int `groups:"create,update" json:"indices_queries_cache_size,omitempty"`

	// +kubebuilder:validation:Minimum=64
	// +kubebuilder:validation:Maximum=4096
	// IndicesQueryBoolMaxClauseCount Maximum number of clauses Lucene BooleanQuery can have. The default value (1024) is relatively high, and increasing it may cause performance issues. Investigate other approaches first before increasing this value.
	IndicesQueryBoolMaxClauseCount *int `groups:"create,update" json:"indices_query_bool_max_clause_count,omitempty"`

	// +kubebuilder:validation:Minimum=40
	// +kubebuilder:validation:Maximum=400
	// IndicesRecoveryMaxBytesPerSec Limits total inbound and outbound recovery traffic for each node. Applies to both peer recoveries as well as snapshot recoveries (i.e., restores from a snapshot). Defaults to 40mb
	IndicesRecoveryMaxBytesPerSec *int `groups:"create,update" json:"indices_recovery_max_bytes_per_sec,omitempty"`

	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=5
	// IndicesRecoveryMaxConcurrentFileChunks Number of file chunks sent in parallel for each recovery. Defaults to 2.
	IndicesRecoveryMaxConcurrentFileChunks *int `groups:"create,update" json:"indices_recovery_max_concurrent_file_chunks,omitempty"`

	// OverrideMainResponseVersion Compatibility mode sets OpenSearch to report its version as 7.10 so clients continue to work. Default is false
	OverrideMainResponseVersion *bool `groups:"create,update" json:"override_main_response_version,omitempty"`

	// +kubebuilder:validation:MaxItems=32
	// ReindexRemoteWhitelist Whitelisted addresses for reindexing. Changing this value will cause all OpenSearch instances to restart.
	ReindexRemoteWhitelist []string `groups:"create,update" json:"reindex_remote_whitelist,omitempty"`

	// +kubebuilder:validation:MaxLength=1024
	// ScriptMaxCompilationsRate Script compilation circuit breaker limits the number of inline script compilations within a period of time. Default is use-context
	ScriptMaxCompilationsRate *string `groups:"create,update" json:"script_max_compilations_rate,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=20000
	// SearchMaxBuckets Maximum number of aggregation buckets allowed in a single response. OpenSearch default value is used when this is not defined.
	SearchMaxBuckets *int `groups:"create,update" json:"search_max_buckets,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// ThreadPoolAnalyzeQueueSize Size for the thread pool queue. See documentation for exact details.
	ThreadPoolAnalyzeQueueSize *int `groups:"create,update" json:"thread_pool_analyze_queue_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// ThreadPoolAnalyzeSize Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolAnalyzeSize *int `groups:"create,update" json:"thread_pool_analyze_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// ThreadPoolForceMergeSize Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolForceMergeSize *int `groups:"create,update" json:"thread_pool_force_merge_size,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// ThreadPoolGetQueueSize Size for the thread pool queue. See documentation for exact details.
	ThreadPoolGetQueueSize *int `groups:"create,update" json:"thread_pool_get_queue_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// ThreadPoolGetSize Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolGetSize *int `groups:"create,update" json:"thread_pool_get_size,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// ThreadPoolSearchQueueSize Size for the thread pool queue. See documentation for exact details.
	ThreadPoolSearchQueueSize *int `groups:"create,update" json:"thread_pool_search_queue_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// ThreadPoolSearchSize Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolSearchSize *int `groups:"create,update" json:"thread_pool_search_size,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// ThreadPoolSearchThrottledQueueSize Size for the thread pool queue. See documentation for exact details.
	ThreadPoolSearchThrottledQueueSize *int `groups:"create,update" json:"thread_pool_search_throttled_queue_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// ThreadPoolSearchThrottledSize Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolSearchThrottledSize *int `groups:"create,update" json:"thread_pool_search_throttled_size,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// ThreadPoolWriteQueueSize Size for the thread pool queue. See documentation for exact details.
	ThreadPoolWriteQueueSize *int `groups:"create,update" json:"thread_pool_write_queue_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// ThreadPoolWriteSize Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolWriteSize *int `groups:"create,update" json:"thread_pool_write_size,omitempty"`
}

// OpensearchDashboards OpenSearch Dashboards settings
type OpensearchDashboards struct {
	// Enabled Enable or disable OpenSearch Dashboards
	Enabled *bool `groups:"create,update" json:"enabled,omitempty"`

	// +kubebuilder:validation:Minimum=64
	// +kubebuilder:validation:Maximum=2048
	// MaxOldSpaceSize Limits the maximum amount of memory (in MiB) the OpenSearch Dashboards process can use. This sets the max_old_space_size option of the nodejs running the OpenSearch Dashboards. Note: the memory reserved by OpenSearch Dashboards is not available for OpenSearch.
	MaxOldSpaceSize *int `groups:"create,update" json:"max_old_space_size,omitempty"`

	// +kubebuilder:validation:Minimum=5000
	// +kubebuilder:validation:Maximum=120000
	// OpensearchRequestTimeout Timeout in milliseconds for requests made by OpenSearch Dashboards towards OpenSearch
	OpensearchRequestTimeout *int `groups:"create,update" json:"opensearch_request_timeout,omitempty"`
}

// PrivateAccess Allow access to selected service ports from private networks
type PrivateAccess struct {
	// Opensearch Allow clients to connect to opensearch with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Opensearch *bool `groups:"create,update" json:"opensearch,omitempty"`

	// OpensearchDashboards Allow clients to connect to opensearch_dashboards with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	OpensearchDashboards *bool `groups:"create,update" json:"opensearch_dashboards,omitempty"`

	// Prometheus Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Prometheus *bool `groups:"create,update" json:"prometheus,omitempty"`
}

// PrivatelinkAccess Allow access to selected service components through Privatelink
type PrivatelinkAccess struct {
	// Opensearch Enable opensearch
	Opensearch *bool `groups:"create,update" json:"opensearch,omitempty"`

	// OpensearchDashboards Enable opensearch_dashboards
	OpensearchDashboards *bool `groups:"create,update" json:"opensearch_dashboards,omitempty"`

	// Prometheus Enable prometheus
	Prometheus *bool `groups:"create,update" json:"prometheus,omitempty"`
}

// PublicAccess Allow access to selected service ports from the public Internet
type PublicAccess struct {
	// Opensearch Allow clients to connect to opensearch from the public internet for service nodes that are in a project VPC or another type of private network
	Opensearch *bool `groups:"create,update" json:"opensearch,omitempty"`

	// OpensearchDashboards Allow clients to connect to opensearch_dashboards from the public internet for service nodes that are in a project VPC or another type of private network
	OpensearchDashboards *bool `groups:"create,update" json:"opensearch_dashboards,omitempty"`

	// Prometheus Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network
	Prometheus *bool `groups:"create,update" json:"prometheus,omitempty"`
}
type OpensearchUserConfig struct {
	// +kubebuilder:validation:MaxItems=1
	// AdditionalBackupRegions Additional Cloud Regions for Backup Replication
	AdditionalBackupRegions []string `groups:"create,update" json:"additional_backup_regions,omitempty"`

	// +kubebuilder:validation:MaxLength=255
	// CustomDomain Serve the web frontend using a custom CNAME pointing to the Aiven DNS name
	CustomDomain *string `groups:"create,update" json:"custom_domain,omitempty"`

	// DisableReplicationFactorAdjustment DEPRECATED: Disable automatic replication factor adjustment for multi-node services. By default, Aiven ensures all indexes are replicated at least to two nodes. Note: Due to potential data loss in case of losing a service node, this setting can no longer be activated.
	DisableReplicationFactorAdjustment *bool `groups:"create,update" json:"disable_replication_factor_adjustment,omitempty"`

	// +kubebuilder:validation:MaxItems=512
	// IndexPatterns Index patterns
	IndexPatterns []*IndexPatterns `groups:"create,update" json:"index_patterns,omitempty"`

	// IndexTemplate Template settings for all new indexes
	IndexTemplate *IndexTemplate `groups:"create,update" json:"index_template,omitempty"`

	// +kubebuilder:validation:MaxItems=1024
	// IpFilter Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'
	IpFilter []*IpFilter `groups:"create,update" json:"ip_filter,omitempty"`

	// KeepIndexRefreshInterval Aiven automation resets index.refresh_interval to default value for every index to be sure that indices are always visible to search. If it doesn't fit your case, you can disable this by setting up this flag to true.
	KeepIndexRefreshInterval *bool `groups:"create,update" json:"keep_index_refresh_interval,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// MaxIndexCount DEPRECATED: use index_patterns instead
	MaxIndexCount *int `groups:"create,update" json:"max_index_count,omitempty"`

	// Opensearch OpenSearch settings
	Opensearch *Opensearch `groups:"create,update" json:"opensearch,omitempty"`

	// OpensearchDashboards OpenSearch Dashboards settings
	OpensearchDashboards *OpensearchDashboards `groups:"create,update" json:"opensearch_dashboards,omitempty"`

	// +kubebuilder:validation:Enum=1;2
	// OpensearchVersion OpenSearch major version
	OpensearchVersion *string `groups:"create,update" json:"opensearch_version,omitempty"`

	// PrivateAccess Allow access to selected service ports from private networks
	PrivateAccess *PrivateAccess `groups:"create,update" json:"private_access,omitempty"`

	// PrivatelinkAccess Allow access to selected service components through Privatelink
	PrivatelinkAccess *PrivatelinkAccess `groups:"create,update" json:"privatelink_access,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// ProjectToForkFrom Name of another project to fork a service from. This has effect only when a new service is being created.
	ProjectToForkFrom *string `groups:"create" json:"project_to_fork_from,omitempty"`

	// PublicAccess Allow access to selected service ports from the public Internet
	PublicAccess *PublicAccess `groups:"create,update" json:"public_access,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9-_:.]+$`
	// RecoveryBasebackupName Name of the basebackup to restore in forked service
	RecoveryBasebackupName *string `groups:"create,update" json:"recovery_basebackup_name,omitempty"`

	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// ServiceToForkFrom Name of another service to fork from. This has effect only when a new service is being created.
	ServiceToForkFrom *string `groups:"create" json:"service_to_fork_from,omitempty"`

	// StaticIps Use static public IP addresses
	StaticIps *bool `groups:"create,update" json:"static_ips,omitempty"`
}
