// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenSearchSpec defines the desired state of OpenSearch
type OpenSearchSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef AuthSecretReference `json:"authSecretRef"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// OpenSearch specific user configuration options
	UserConfig OpenSearchUserConfig `json:"userConfig,omitempty"`
}

type OpenSearchIndexTemplate struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=29
	// index.number_of_replicas The number of replicas each primary shard has.
	NumberOfReplicas *int64 `json:"number_of_replicas,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1024
	// index.number_of_shards The number of primary shards that an index should have.
	NumberOfShards *int64 `json:"number_of_shards,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100000
	// index.mapping.nested_objects.limit The maximum number of nested JSON objects that a single document can contain across all nested types. This limit helps to prevent out of memory errors when a document contains too many nested objects. Default is 10000.
	MappingNestedObjectsLimit *int64 `json:"mapping_nested_objects_limit,omitempty"`
}

type OpenSearchUserConfigOpenSearch struct {
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// analyze thread pool queue size for the thread pool queue. See documentation for exact details.
	ThreadPoolAnalyzeQueueSize *int64 `json:"thread_pool_analyze_queue_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// analyze thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolAnalyzeSize *int64 `json:"thread_pool_analyze_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// force_merge thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolForceMergeSize *int64 `json:"thread_pool_force_merge_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// get thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolGetSize *int64 `json:"thread_pool_get_size,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// search_throttled thread pool queue size for the thread pool queue. See documentation for exact details.
	ThreadPoolSearchThrottledQueueSize *int64 `json:"thread_pool_search_throttled_queue_size,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// write thread pool queue size for the thread pool queue. See documentation for exact details.
	ThreadPoolWriteQueueSize *int64 `json:"thread_pool_write_queue_size,omitempty"`

	// reindex_remote_whitelist Whitelisted addresses for reindexing. Changing this value will cause all OpenSearch instances to restart.
	// Address (hostname:port or IP:port)
	ReindexRemoteWhitelist []string `json:"reindex_remote_whitelist,omitempty"`

	// +kubebuilder:validation:Minimum=3
	// +kubebuilder:validation:Maximum=100
	// indices.fielddata.cache.size Relative amount. Maximum amount of heap memory used for field data cache. This is an expert setting; decreasing the value too much will increase overhead of loading field data; too much memory used for field data cache will decrease amount of heap available for other operations.
	IndicesFielddataCacheSize *int64 `json:"indices_fielddata_cache_size,omitempty"`

	// +kubebuilder:validation:Minimum=64
	// +kubebuilder:validation:Maximum=4096
	// indices.query.bool.max_clause_count Maximum number of clauses Lucene BooleanQuery can have. The default value (1024) is relatively high, and increasing it may cause performance issues. Investigate other approaches first before increasing this value.
	IndicesQueryBoolMaxClauseCount *int64 `json:"indices_query_bool_max_clause_count,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// get thread pool queue size for the thread pool queue. See documentation for exact details.
	ThreadPoolGetQueueSize *int64 `json:"thread_pool_get_queue_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// index thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolIndexSize *int64 `json:"thread_pool_index_size,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=2000
	// search thread pool queue size for the thread pool queue. See documentation for exact details.
	ThreadPoolSearchQueueSize *int64 `json:"thread_pool_search_queue_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// write thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolWriteSize *int64 `json:"thread_pool_write_size,omitempty"`

	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=262144
	// http.max_header_size The max size of allowed headers, in bytes
	HttpMaxHeaderSize *int64 `json:"http_max_header_size,omitempty"`

	// +kubebuilder:validation:Minimum=3
	// +kubebuilder:validation:Maximum=40
	// indices.memory.index_buffer_size Percentage value. Default is 10%. Total amount of heap used for indexing buffer, before writing segments to disk. This is an expert setting. Too low value will slow down indexing; too high value will increase indexing performance but causes performance issues for query performance.
	IndicesMemoryIndexBufferSize *int64 `json:"indices_memory_index_buffer_size,omitempty"`

	// +kubebuilder:validation:Minimum=3
	// +kubebuilder:validation:Maximum=40
	// indices.queries.cache.size Percentage value. Default is 10%. Maximum amount of heap used for query cache. This is an expert setting. Too low value will decrease query performance and increase performance for other operations; too high value will cause issues with other OpenSearch functionality.
	IndicesQueriesCacheSize *int64 `json:"indices_queries_cache_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// search thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolSearchSize *int64 `json:"thread_pool_search_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=128
	// search_throttled thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
	ThreadPoolSearchThrottledSize *int64 `json:"thread_pool_search_throttled_size,omitempty"`

	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=65536
	// http.max_initial_line_length The max length of an HTTP URL, in bytes
	HttpMaxInitialLineLength *int64 `json:"http_max_initial_line_length,omitempty"`

	// Require explicit index names when deleting
	ActionDestructiveRequiresName *bool `json:"action_destructive_requires_name,omitempty"`

	// +kubebuilder:validation:Minimum=100
	// +kubebuilder:validation:Maximum=10000
	// cluster.max_shards_per_node Controls the number of shards allowed in the cluster per data node
	ClusterMaxShardsPerNode *int64 `json:"cluster_max_shards_per_node,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=2147483647
	// http.max_content_length Maximum content length for HTTP requests to the OpenSearch HTTP API, in bytes.
	HttpMaxContentLength *int64 `json:"http_max_content_length,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=20000
	// search.max_buckets Maximum number of aggregation buckets allowed in a single response. OpenSearch default value is used when this is not defined.
	SearchMaxBuckets *int64 `json:"search_max_buckets,omitempty"`

	// action.auto_create_index Explicitly allow or block automatic creation of indices. Defaults to true
	ActionAutoCreateIndexEnabled *bool `json:"action_auto_create_index_enabled,omitempty"`
}

type OpenSearchPrivateAccess struct {
	// Allow clients to connect to opensearch with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Opensearch *bool `json:"opensearch,omitempty"`

	// Allow clients to connect to opensearch_dashboards with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	OpensearchDashboards *bool `json:"opensearch_dashboards,omitempty"`

	// Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Prometheus *bool `json:"prometheus,omitempty"`
}

type OpenSearchPublicAccess struct {
	// Allow clients to connect to opensearch_dashboards from the public internet for service nodes that are in a project VPC or another type of private network
	OpensearchDashboards *bool `json:"opensearch_dashboards,omitempty"`

	// Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network
	Prometheus *bool `json:"prometheus,omitempty"`

	// Allow clients to connect to opensearch from the public internet for service nodes that are in a project VPC or another type of private network
	Opensearch *bool `json:"opensearch,omitempty"`
}

type OpenSearchIndexPatterns struct {
	// +kubebuilder:validation:Minimum=0
	// Maximum number of indexes to keep
	MaxIndexCount *int64 `json:"max_index_count,omitempty"`

	// +kubebuilder:validation:MaxLength=1024
	// Must consist of alpha-numeric characters, dashes, underscores, dots and glob characters (* and ?)
	Pattern string `json:"pattern,omitempty"`
}
type OpensearchDashboards struct {
	// Enable or disable OpenSearch Dashboards
	Enabled *bool `json:"enabled,omitempty"`

	// +kubebuilder:validation:Minimum=64
	// +kubebuilder:validation:Maximum=1024
	// max_old_space_size Limits the maximum amount of memory (in MiB) the OpenSearch Dashboards process can use. This sets the max_old_space_size option of the nodejs running the OpenSearch Dashboards. Note: the memory reserved by OpenSearch Dashboards is not available for OpenSearch.
	MaxOldSpaceSize *int64 `json:"max_old_space_size,omitempty"`

	// +kubebuilder:validation:Minimum=5000
	// +kubebuilder:validation:Maximum=120000
	// Timeout in milliseconds for requests made by OpenSearch Dashboards towards OpenSearch
	OpensearchRequestTimeout *int64 `json:"opensearch_request_timeout,omitempty"`
}

type OpenSearchPrivatelinkAccess struct {
	// Enable opensearch
	Opensearch *bool `json:"opensearch,omitempty"`

	// Enable opensearch_dashboards
	OpensearchDashboards *bool `json:"opensearch_dashboards,omitempty"`
}

type OpenSearchUserConfig struct {
	// +kubebuilder:validation:Enum=1
	// OpenSearch major version
	OpensearchVersion string `json:"opensearch_version,omitempty"`

	// OpenSearch Dashboards settings
	OpensearchDashboards OpensearchDashboards `json:"opensearch_dashboards,omitempty"`

	// +kubebuilder:validation:Format="^[a-zA-Z0-9-_:.]+$"
	// +kubebuilder:validation:MaxLength=128
	// Name of the basebackup to restore in forked service
	RecoveryBasebackupName string `json:"recovery_basebackup_name,omitempty"`

	// Allow access to selected service components through Privatelink
	PrivatelinkAccess OpenSearchPrivatelinkAccess `json:"privatelink_access,omitempty"`

	// Static IP addresses Use static public IP addresses
	StaticIps *bool `json:"static_ips,omitempty"`

	// Allow access to selected service ports from the public Internet
	PublicAccess OpenSearchPublicAccess `json:"public_access,omitempty"`

	// +kubebuilder:validation:MaxLength=255
	// Custom domain Serve the web frontend using a custom CNAME pointing to the Aiven DNS name
	CustomDomain string `json:"custom_domain,omitempty"`

	// Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like 'logs.?' and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note 'logs.?' does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.
	IndexPatterns []OpenSearchIndexPatterns `json:"index_patterns,omitempty"`

	// Glob pattern and number of indexes matching that pattern to be kept Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like 'logs.?' and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note 'logs.?' does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.
	// IP filter Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'
	IpFilter []string `json:"ip_filter,omitempty"`

	// Allow access to selected service ports from private networks
	PrivateAccess OpenSearchPrivateAccess `json:"private_access,omitempty"`

	// Template settings for all new indexes
	IndexTemplate OpenSearchIndexTemplate `json:"index_template,omitempty"`

	// OpenSearch settings
	Opensearch OpenSearchUserConfigOpenSearch `json:"opensearch,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// Maximum index count Maximum number of indexes to keep before deleting the oldest one
	MaxIndexCount *int64 `json:"max_index_count,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Name of another project to fork a service from. This has effect only when a new service is being created.
	ProjectToForkFrom string `json:"project_to_fork_from,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Name of another service to fork from. This has effect only when a new service is being created.
	ServiceToForkFrom string `json:"service_to_fork_from,omitempty"`

	// Disable replication factor adjustment DEPRECATED: Disable automatic replication factor adjustment for multi-node services. By default, Aiven ensures all indexes are replicated at least to two nodes. Note: Due to potential data loss in case of losing a service node, this setting can no longer be activated.
	DisableReplicationFactorAdjustment *bool `json:"disable_replication_factor_adjustment,omitempty"`

	// Don't reset index.refresh_interval to the default value Aiven automation resets index.refresh_interval to default value for every index to be sure that indices are always visible to search. If it doesn't fit your case, you can disable this by setting up this flag to true.
	KeepIndexRefreshInterval *bool `json:"keep_index_refresh_interval,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenSearch is the Schema for the opensearches API
type OpenSearch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenSearchSpec `json:"spec,omitempty"`
	Status ServiceStatus  `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenSearchList contains a list of OpenSearch
type OpenSearchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenSearch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenSearch{}, &OpenSearchList{})
}
