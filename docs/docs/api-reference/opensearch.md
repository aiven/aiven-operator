---
title: "OpenSearch"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | OpenSearch |

OpenSearchSpec defines the desired state of OpenSearch.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`cloudName`](#cloudName){: name='cloudName'} (string, MaxLength: 256). Cloud the service runs in. 
- [`connInfoSecretTarget`](#connInfoSecretTarget){: name='connInfoSecretTarget'} (object). Information regarding secret creation. See [below for nested schema](#connInfoSecretTarget).
- [`disk_space`](#disk_space){: name='disk_space'} (string). The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing. 
- [`maintenanceWindowDow`](#maintenanceWindowDow){: name='maintenanceWindowDow'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc. 
- [`maintenanceWindowTime`](#maintenanceWindowTime){: name='maintenanceWindowTime'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format. 
- [`plan`](#plan){: name='plan'} (string, MaxLength: 128). Subscription plan. 
- [`project`](#project){: name='project'} (string, Immutable, MaxLength: 63). Target project. 
- [`projectVPCRef`](#projectVPCRef){: name='projectVPCRef'} (object, Immutable). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See [below for nested schema](#projectVPCRef).
- [`projectVpcId`](#projectVpcId){: name='projectVpcId'} (string, Immutable, MaxLength: 36). Identifier of the VPC the service should be in, if any. 
- [`serviceIntegrations`](#serviceIntegrations){: name='serviceIntegrations'} (array, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See [below for nested schema](#serviceIntegrations).
- [`tags`](#tags){: name='tags'} (object). Tags are key-value pairs that allow you to categorize services. 
- [`terminationProtection`](#terminationProtection){: name='terminationProtection'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services. 
- [`userConfig`](#userConfig){: name='userConfig'} (object). OpenSearch specific user configuration options. See [below for nested schema](#userConfig).

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

OpenSearch specific user configuration options.

**Optional**

- [`additional_backup_regions`](#additional_backup_regions){: name='additional_backup_regions'} (array, MaxItems: 1). Additional Cloud Regions for Backup Replication. 
- [`custom_domain`](#custom_domain){: name='custom_domain'} (string, MaxLength: 255). Serve the web frontend using a custom CNAME pointing to the Aiven DNS name. 
- [`disable_replication_factor_adjustment`](#disable_replication_factor_adjustment){: name='disable_replication_factor_adjustment'} (boolean). DEPRECATED: Disable automatic replication factor adjustment for multi-node services. By default, Aiven ensures all indexes are replicated at least to two nodes. Note: Due to potential data loss in case of losing a service node, this setting can no longer be activated. 
- [`index_patterns`](#index_patterns){: name='index_patterns'} (array, MaxItems: 512). Index patterns. See [below for nested schema](#index_patterns).
- [`index_template`](#index_template){: name='index_template'} (object). Template settings for all new indexes. See [below for nested schema](#index_template).
- [`ip_filter`](#ip_filter){: name='ip_filter'} (array, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See [below for nested schema](#ip_filter).
- [`keep_index_refresh_interval`](#keep_index_refresh_interval){: name='keep_index_refresh_interval'} (boolean). Aiven automation resets index.refresh_interval to default value for every index to be sure that indices are always visible to search. If it doesn't fit your case, you can disable this by setting up this flag to true. 
- [`max_index_count`](#max_index_count){: name='max_index_count'} (integer, Minimum: 0). DEPRECATED: use index_patterns instead. 
- [`opensearch`](#opensearch){: name='opensearch'} (object). OpenSearch settings. See [below for nested schema](#opensearch).
- [`opensearch_dashboards`](#opensearch_dashboards){: name='opensearch_dashboards'} (object). OpenSearch Dashboards settings. See [below for nested schema](#opensearch_dashboards).
- [`opensearch_version`](#opensearch_version){: name='opensearch_version'} (string, Enum: `1`, `2`). OpenSearch major version. 
- [`private_access`](#private_access){: name='private_access'} (object). Allow access to selected service ports from private networks. See [below for nested schema](#private_access).
- [`privatelink_access`](#privatelink_access){: name='privatelink_access'} (object). Allow access to selected service components through Privatelink. See [below for nested schema](#privatelink_access).
- [`project_to_fork_from`](#project_to_fork_from){: name='project_to_fork_from'} (string, Immutable, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created. 
- [`public_access`](#public_access){: name='public_access'} (object). Allow access to selected service ports from the public Internet. See [below for nested schema](#public_access).
- [`recovery_basebackup_name`](#recovery_basebackup_name){: name='recovery_basebackup_name'} (string, Pattern: `^[a-zA-Z0-9-_:.]+$`, MaxLength: 128). Name of the basebackup to restore in forked service. 
- [`service_to_fork_from`](#service_to_fork_from){: name='service_to_fork_from'} (string, Immutable, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created. 
- [`static_ips`](#static_ips){: name='static_ips'} (boolean). Use static public IP addresses. 

### index_patterns {: #index_patterns }

Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like 'logs.?' and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note 'logs.?' does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.

**Required**

- [`max_index_count`](#max_index_count){: name='max_index_count'} (integer, Minimum: 0). Maximum number of indexes to keep. 
- [`pattern`](#pattern){: name='pattern'} (string, Pattern: `^[A-Za-z0-9-_.*?]+$`, MaxLength: 1024). fnmatch pattern. 

**Optional**

- [`sorting_algorithm`](#sorting_algorithm){: name='sorting_algorithm'} (string, Enum: `alphabetical`, `creation_date`). Deletion sorting algorithm. 

### index_template {: #index_template }

Template settings for all new indexes.

**Optional**

- [`mapping_nested_objects_limit`](#mapping_nested_objects_limit){: name='mapping_nested_objects_limit'} (integer, Minimum: 0, Maximum: 100000). The maximum number of nested JSON objects that a single document can contain across all nested types. This limit helps to prevent out of memory errors when a document contains too many nested objects. Default is 10000. 
- [`number_of_replicas`](#number_of_replicas){: name='number_of_replicas'} (integer, Minimum: 0, Maximum: 29). The number of replicas each primary shard has. 
- [`number_of_shards`](#number_of_shards){: name='number_of_shards'} (integer, Minimum: 1, Maximum: 1024). The number of primary shards that an index should have. 

### ip_filter {: #ip_filter }

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#network){: name='network'} (string, MaxLength: 43). CIDR address block. 

**Optional**

- [`description`](#description){: name='description'} (string, MaxLength: 1024). Description for IP filter list entry. 

### opensearch {: #opensearch }

OpenSearch settings.

**Optional**

- [`action_auto_create_index_enabled`](#action_auto_create_index_enabled){: name='action_auto_create_index_enabled'} (boolean). Explicitly allow or block automatic creation of indices. Defaults to true. 
- [`action_destructive_requires_name`](#action_destructive_requires_name){: name='action_destructive_requires_name'} (boolean). Require explicit index names when deleting. 
- [`cluster_max_shards_per_node`](#cluster_max_shards_per_node){: name='cluster_max_shards_per_node'} (integer, Minimum: 100, Maximum: 10000). Controls the number of shards allowed in the cluster per data node. 
- [`cluster_routing_allocation_node_concurrent_recoveries`](#cluster_routing_allocation_node_concurrent_recoveries){: name='cluster_routing_allocation_node_concurrent_recoveries'} (integer, Minimum: 2, Maximum: 16). How many concurrent incoming/outgoing shard recoveries (normally replicas) are allowed to happen on a node. Defaults to 2. 
- [`email_sender_name`](#email_sender_name){: name='email_sender_name'} (string, Pattern: `^[a-zA-Z0-9-_]+$`, MaxLength: 40). Sender email name placeholder to be used in Opensearch Dashboards and Opensearch keystore. 
- [`email_sender_password`](#email_sender_password){: name='email_sender_password'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 1024). Sender email password for Opensearch alerts to authenticate with SMTP server. 
- [`email_sender_username`](#email_sender_username){: name='email_sender_username'} (string, MaxLength: 320). Sender email address for Opensearch alerts. 
- [`http_max_content_length`](#http_max_content_length){: name='http_max_content_length'} (integer, Minimum: 1, Maximum: 2147483647). Maximum content length for HTTP requests to the OpenSearch HTTP API, in bytes. 
- [`http_max_header_size`](#http_max_header_size){: name='http_max_header_size'} (integer, Minimum: 1024, Maximum: 262144). The max size of allowed headers, in bytes. 
- [`http_max_initial_line_length`](#http_max_initial_line_length){: name='http_max_initial_line_length'} (integer, Minimum: 1024, Maximum: 65536). The max length of an HTTP URL, in bytes. 
- [`indices_fielddata_cache_size`](#indices_fielddata_cache_size){: name='indices_fielddata_cache_size'} (integer, Minimum: 3, Maximum: 100). Relative amount. Maximum amount of heap memory used for field data cache. This is an expert setting; decreasing the value too much will increase overhead of loading field data; too much memory used for field data cache will decrease amount of heap available for other operations. 
- [`indices_memory_index_buffer_size`](#indices_memory_index_buffer_size){: name='indices_memory_index_buffer_size'} (integer, Minimum: 3, Maximum: 40). Percentage value. Default is 10%. Total amount of heap used for indexing buffer, before writing segments to disk. This is an expert setting. Too low value will slow down indexing; too high value will increase indexing performance but causes performance issues for query performance. 
- [`indices_queries_cache_size`](#indices_queries_cache_size){: name='indices_queries_cache_size'} (integer, Minimum: 3, Maximum: 40). Percentage value. Default is 10%. Maximum amount of heap used for query cache. This is an expert setting. Too low value will decrease query performance and increase performance for other operations; too high value will cause issues with other OpenSearch functionality. 
- [`indices_query_bool_max_clause_count`](#indices_query_bool_max_clause_count){: name='indices_query_bool_max_clause_count'} (integer, Minimum: 64, Maximum: 4096). Maximum number of clauses Lucene BooleanQuery can have. The default value (1024) is relatively high, and increasing it may cause performance issues. Investigate other approaches first before increasing this value. 
- [`indices_recovery_max_bytes_per_sec`](#indices_recovery_max_bytes_per_sec){: name='indices_recovery_max_bytes_per_sec'} (integer, Minimum: 40, Maximum: 400). Limits total inbound and outbound recovery traffic for each node. Applies to both peer recoveries as well as snapshot recoveries (i.e., restores from a snapshot). Defaults to 40mb. 
- [`indices_recovery_max_concurrent_file_chunks`](#indices_recovery_max_concurrent_file_chunks){: name='indices_recovery_max_concurrent_file_chunks'} (integer, Minimum: 2, Maximum: 5). Number of file chunks sent in parallel for each recovery. Defaults to 2. 
- [`override_main_response_version`](#override_main_response_version){: name='override_main_response_version'} (boolean). Compatibility mode sets OpenSearch to report its version as 7.10 so clients continue to work. Default is false. 
- [`reindex_remote_whitelist`](#reindex_remote_whitelist){: name='reindex_remote_whitelist'} (array, MaxItems: 32). Whitelisted addresses for reindexing. Changing this value will cause all OpenSearch instances to restart. 
- [`script_max_compilations_rate`](#script_max_compilations_rate){: name='script_max_compilations_rate'} (string, MaxLength: 1024). Script compilation circuit breaker limits the number of inline script compilations within a period of time. Default is use-context. 
- [`search_max_buckets`](#search_max_buckets){: name='search_max_buckets'} (integer, Minimum: 1, Maximum: 20000). Maximum number of aggregation buckets allowed in a single response. OpenSearch default value is used when this is not defined. 
- [`thread_pool_analyze_queue_size`](#thread_pool_analyze_queue_size){: name='thread_pool_analyze_queue_size'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details. 
- [`thread_pool_analyze_size`](#thread_pool_analyze_size){: name='thread_pool_analyze_size'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value. 
- [`thread_pool_force_merge_size`](#thread_pool_force_merge_size){: name='thread_pool_force_merge_size'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value. 
- [`thread_pool_get_queue_size`](#thread_pool_get_queue_size){: name='thread_pool_get_queue_size'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details. 
- [`thread_pool_get_size`](#thread_pool_get_size){: name='thread_pool_get_size'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value. 
- [`thread_pool_search_queue_size`](#thread_pool_search_queue_size){: name='thread_pool_search_queue_size'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details. 
- [`thread_pool_search_size`](#thread_pool_search_size){: name='thread_pool_search_size'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value. 
- [`thread_pool_search_throttled_queue_size`](#thread_pool_search_throttled_queue_size){: name='thread_pool_search_throttled_queue_size'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details. 
- [`thread_pool_search_throttled_size`](#thread_pool_search_throttled_size){: name='thread_pool_search_throttled_size'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value. 
- [`thread_pool_write_queue_size`](#thread_pool_write_queue_size){: name='thread_pool_write_queue_size'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details. 
- [`thread_pool_write_size`](#thread_pool_write_size){: name='thread_pool_write_size'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value. 

### opensearch_dashboards {: #opensearch_dashboards }

OpenSearch Dashboards settings.

**Optional**

- [`enabled`](#enabled){: name='enabled'} (boolean). Enable or disable OpenSearch Dashboards. 
- [`max_old_space_size`](#max_old_space_size){: name='max_old_space_size'} (integer, Minimum: 64, Maximum: 2048). Limits the maximum amount of memory (in MiB) the OpenSearch Dashboards process can use. This sets the max_old_space_size option of the nodejs running the OpenSearch Dashboards. Note: the memory reserved by OpenSearch Dashboards is not available for OpenSearch. 
- [`opensearch_request_timeout`](#opensearch_request_timeout){: name='opensearch_request_timeout'} (integer, Minimum: 5000, Maximum: 120000). Timeout in milliseconds for requests made by OpenSearch Dashboards towards OpenSearch. 

### private_access {: #private_access }

Allow access to selected service ports from private networks.

**Optional**

- [`opensearch`](#opensearch){: name='opensearch'} (boolean). Allow clients to connect to opensearch with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`opensearch_dashboards`](#opensearch_dashboards){: name='opensearch_dashboards'} (boolean). Allow clients to connect to opensearch_dashboards with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 

### privatelink_access {: #privatelink_access }

Allow access to selected service components through Privatelink.

**Optional**

- [`opensearch`](#opensearch){: name='opensearch'} (boolean). Enable opensearch. 
- [`opensearch_dashboards`](#opensearch_dashboards){: name='opensearch_dashboards'} (boolean). Enable opensearch_dashboards. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Enable prometheus. 

### public_access {: #public_access }

Allow access to selected service ports from the public Internet.

**Optional**

- [`opensearch`](#opensearch){: name='opensearch'} (boolean). Allow clients to connect to opensearch from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`opensearch_dashboards`](#opensearch_dashboards){: name='opensearch_dashboards'} (boolean). Allow clients to connect to opensearch_dashboards from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network. 

