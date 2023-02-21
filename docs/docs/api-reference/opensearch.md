---
title: "OpenSearch"
---

## Usage example

```yaml
apiVersion: aiven.io/v1alpha1
kind: OpenSearch
metadata:
  name: my-os
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: os-secret

  project: my-aiven-project

  cloudName: google-europe-west1
  plan: startup-4
  disk_space: 80Gib

  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
```

## Schema {: #Schema }

OpenSearch is the Schema for the opensearches API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Must be equal to `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Must be equal to `OpenSearch`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). OpenSearchSpec defines the desired state of OpenSearch. See below for [nested schema](#spec).

## spec {: #spec }

OpenSearchSpec defines the desired state of OpenSearch.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, MaxLength: 63). Target project.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, MaxLength: 256). Cloud the service runs in.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Information regarding secret creation. See below for [nested schema](#spec.connInfoSecretTarget).
- [`disk_space`](#spec.disk_space-property){: name='spec.disk_space-property'} (string). The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
- [`maintenanceWindowDow`](#spec.maintenanceWindowDow-property){: name='spec.maintenanceWindowDow-property'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
- [`maintenanceWindowTime`](#spec.maintenanceWindowTime-property){: name='spec.maintenanceWindowTime-property'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
- [`plan`](#spec.plan-property){: name='spec.plan-property'} (string, MaxLength: 128). Subscription plan.
- [`projectVPCRef`](#spec.projectVPCRef-property){: name='spec.projectVPCRef-property'} (object, Immutable). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See below for [nested schema](#spec.projectVPCRef).
- [`projectVpcId`](#spec.projectVpcId-property){: name='spec.projectVpcId-property'} (string, Immutable, MaxLength: 36). Identifier of the VPC the service should be in, if any.
- [`serviceIntegrations`](#spec.serviceIntegrations-property){: name='spec.serviceIntegrations-property'} (array of objects, Immutable, MaxItems: 1).  See below for [nested schema](#spec.serviceIntegrations).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize services.
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services.
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). OpenSearch specific user configuration options. See below for [nested schema](#spec.userConfig).

## authSecretRef {: #spec.authSecretRef }

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string). Name of the secret resource to be created. By default, is equal to the resource name.

## projectVPCRef {: #spec.projectVPCRef }

ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically.

**Required**

- [`name`](#spec.projectVPCRef.name-property){: name='spec.projectVPCRef.name-property'} (string, MinLength: 1). 

**Optional**

- [`namespace`](#spec.projectVPCRef.namespace-property){: name='spec.projectVPCRef.namespace-property'} (string, MinLength: 1). 

## serviceIntegrations {: #spec.serviceIntegrations }

**Required**

- [`integrationType`](#spec.serviceIntegrations.integrationType-property){: name='spec.serviceIntegrations.integrationType-property'} (string, Enum: `read_replica`). 
- [`sourceServiceName`](#spec.serviceIntegrations.sourceServiceName-property){: name='spec.serviceIntegrations.sourceServiceName-property'} (string, MinLength: 1, MaxLength: 64). 

## userConfig {: #spec.userConfig }

OpenSearch specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Additional Cloud Regions for Backup Replication.
- [`custom_domain`](#spec.userConfig.custom_domain-property){: name='spec.userConfig.custom_domain-property'} (string, MaxLength: 255). Serve the web frontend using a custom CNAME pointing to the Aiven DNS name.
- [`disable_replication_factor_adjustment`](#spec.userConfig.disable_replication_factor_adjustment-property){: name='spec.userConfig.disable_replication_factor_adjustment-property'} (boolean). DEPRECATED: Disable automatic replication factor adjustment for multi-node services. By default, Aiven ensures all indexes are replicated at least to two nodes. Note: Due to potential data loss in case of losing a service node, this setting can no longer be activated.
- [`index_patterns`](#spec.userConfig.index_patterns-property){: name='spec.userConfig.index_patterns-property'} (array of objects, MaxItems: 512). Index patterns. See below for [nested schema](#spec.userConfig.index_patterns).
- [`index_template`](#spec.userConfig.index_template-property){: name='spec.userConfig.index_template-property'} (object). Template settings for all new indexes. See below for [nested schema](#spec.userConfig.index_template).
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See below for [nested schema](#spec.userConfig.ip_filter).
- [`keep_index_refresh_interval`](#spec.userConfig.keep_index_refresh_interval-property){: name='spec.userConfig.keep_index_refresh_interval-property'} (boolean). Aiven automation resets index.refresh_interval to default value for every index to be sure that indices are always visible to search. If it doesn't fit your case, you can disable this by setting up this flag to true.
- [`max_index_count`](#spec.userConfig.max_index_count-property){: name='spec.userConfig.max_index_count-property'} (integer, Minimum: 0). DEPRECATED: use index_patterns instead.
- [`opensearch`](#spec.userConfig.opensearch-property){: name='spec.userConfig.opensearch-property'} (object). OpenSearch settings. See below for [nested schema](#spec.userConfig.opensearch).
- [`opensearch_dashboards`](#spec.userConfig.opensearch_dashboards-property){: name='spec.userConfig.opensearch_dashboards-property'} (object). OpenSearch Dashboards settings. See below for [nested schema](#spec.userConfig.opensearch_dashboards).
- [`opensearch_version`](#spec.userConfig.opensearch_version-property){: name='spec.userConfig.opensearch_version-property'} (string, Enum: `1`, `2`). OpenSearch major version.
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`project_to_fork_from`](#spec.userConfig.project_to_fork_from-property){: name='spec.userConfig.project_to_fork_from-property'} (string, Immutable, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created.
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`recovery_basebackup_name`](#spec.userConfig.recovery_basebackup_name-property){: name='spec.userConfig.recovery_basebackup_name-property'} (string, Pattern: `^[a-zA-Z0-9-_:.]+$`, MaxLength: 128). Name of the basebackup to restore in forked service.
- [`service_to_fork_from`](#spec.userConfig.service_to_fork_from-property){: name='spec.userConfig.service_to_fork_from-property'} (string, Immutable, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

### index_patterns {: #spec.userConfig.index_patterns }

Index patterns.

**Required**

- [`max_index_count`](#spec.userConfig.index_patterns.max_index_count-property){: name='spec.userConfig.index_patterns.max_index_count-property'} (integer, Minimum: 0). Maximum number of indexes to keep.
- [`pattern`](#spec.userConfig.index_patterns.pattern-property){: name='spec.userConfig.index_patterns.pattern-property'} (string, Pattern: `^[A-Za-z0-9-_.*?]+$`, MaxLength: 1024). fnmatch pattern.

**Optional**

- [`sorting_algorithm`](#spec.userConfig.index_patterns.sorting_algorithm-property){: name='spec.userConfig.index_patterns.sorting_algorithm-property'} (string, Enum: `alphabetical`, `creation_date`). Deletion sorting algorithm.

### index_template {: #spec.userConfig.index_template }

Template settings for all new indexes.

**Optional**

- [`mapping_nested_objects_limit`](#spec.userConfig.index_template.mapping_nested_objects_limit-property){: name='spec.userConfig.index_template.mapping_nested_objects_limit-property'} (integer, Minimum: 0, Maximum: 100000). The maximum number of nested JSON objects that a single document can contain across all nested types. This limit helps to prevent out of memory errors when a document contains too many nested objects. Default is 10000.
- [`number_of_replicas`](#spec.userConfig.index_template.number_of_replicas-property){: name='spec.userConfig.index_template.number_of_replicas-property'} (integer, Minimum: 0, Maximum: 29). The number of replicas each primary shard has.
- [`number_of_shards`](#spec.userConfig.index_template.number_of_shards-property){: name='spec.userConfig.index_template.number_of_shards-property'} (integer, Minimum: 1, Maximum: 1024). The number of primary shards that an index should have.

### ip_filter {: #spec.userConfig.ip_filter }

Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### opensearch {: #spec.userConfig.opensearch }

OpenSearch settings.

**Optional**

- [`action_auto_create_index_enabled`](#spec.userConfig.opensearch.action_auto_create_index_enabled-property){: name='spec.userConfig.opensearch.action_auto_create_index_enabled-property'} (boolean). Explicitly allow or block automatic creation of indices. Defaults to true.
- [`action_destructive_requires_name`](#spec.userConfig.opensearch.action_destructive_requires_name-property){: name='spec.userConfig.opensearch.action_destructive_requires_name-property'} (boolean). Require explicit index names when deleting.
- [`cluster_max_shards_per_node`](#spec.userConfig.opensearch.cluster_max_shards_per_node-property){: name='spec.userConfig.opensearch.cluster_max_shards_per_node-property'} (integer, Minimum: 100, Maximum: 10000). Controls the number of shards allowed in the cluster per data node.
- [`cluster_routing_allocation_node_concurrent_recoveries`](#spec.userConfig.opensearch.cluster_routing_allocation_node_concurrent_recoveries-property){: name='spec.userConfig.opensearch.cluster_routing_allocation_node_concurrent_recoveries-property'} (integer, Minimum: 2, Maximum: 16). How many concurrent incoming/outgoing shard recoveries (normally replicas) are allowed to happen on a node. Defaults to 2.
- [`email_sender_name`](#spec.userConfig.opensearch.email_sender_name-property){: name='spec.userConfig.opensearch.email_sender_name-property'} (string, Pattern: `^[a-zA-Z0-9-_]+$`, MaxLength: 40). Sender email name placeholder to be used in Opensearch Dashboards and Opensearch keystore.
- [`email_sender_password`](#spec.userConfig.opensearch.email_sender_password-property){: name='spec.userConfig.opensearch.email_sender_password-property'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 1024). Sender email password for Opensearch alerts to authenticate with SMTP server.
- [`email_sender_username`](#spec.userConfig.opensearch.email_sender_username-property){: name='spec.userConfig.opensearch.email_sender_username-property'} (string, MaxLength: 320). Sender email address for Opensearch alerts.
- [`http_max_content_length`](#spec.userConfig.opensearch.http_max_content_length-property){: name='spec.userConfig.opensearch.http_max_content_length-property'} (integer, Minimum: 1, Maximum: 2147483647). Maximum content length for HTTP requests to the OpenSearch HTTP API, in bytes.
- [`http_max_header_size`](#spec.userConfig.opensearch.http_max_header_size-property){: name='spec.userConfig.opensearch.http_max_header_size-property'} (integer, Minimum: 1024, Maximum: 262144). The max size of allowed headers, in bytes.
- [`http_max_initial_line_length`](#spec.userConfig.opensearch.http_max_initial_line_length-property){: name='spec.userConfig.opensearch.http_max_initial_line_length-property'} (integer, Minimum: 1024, Maximum: 65536). The max length of an HTTP URL, in bytes.
- [`indices_fielddata_cache_size`](#spec.userConfig.opensearch.indices_fielddata_cache_size-property){: name='spec.userConfig.opensearch.indices_fielddata_cache_size-property'} (integer, Minimum: 3, Maximum: 100). Relative amount. Maximum amount of heap memory used for field data cache. This is an expert setting; decreasing the value too much will increase overhead of loading field data; too much memory used for field data cache will decrease amount of heap available for other operations.
- [`indices_memory_index_buffer_size`](#spec.userConfig.opensearch.indices_memory_index_buffer_size-property){: name='spec.userConfig.opensearch.indices_memory_index_buffer_size-property'} (integer, Minimum: 3, Maximum: 40). Percentage value. Default is 10%. Total amount of heap used for indexing buffer, before writing segments to disk. This is an expert setting. Too low value will slow down indexing; too high value will increase indexing performance but causes performance issues for query performance.
- [`indices_queries_cache_size`](#spec.userConfig.opensearch.indices_queries_cache_size-property){: name='spec.userConfig.opensearch.indices_queries_cache_size-property'} (integer, Minimum: 3, Maximum: 40). Percentage value. Default is 10%. Maximum amount of heap used for query cache. This is an expert setting. Too low value will decrease query performance and increase performance for other operations; too high value will cause issues with other OpenSearch functionality.
- [`indices_query_bool_max_clause_count`](#spec.userConfig.opensearch.indices_query_bool_max_clause_count-property){: name='spec.userConfig.opensearch.indices_query_bool_max_clause_count-property'} (integer, Minimum: 64, Maximum: 4096). Maximum number of clauses Lucene BooleanQuery can have. The default value (1024) is relatively high, and increasing it may cause performance issues. Investigate other approaches first before increasing this value.
- [`indices_recovery_max_bytes_per_sec`](#spec.userConfig.opensearch.indices_recovery_max_bytes_per_sec-property){: name='spec.userConfig.opensearch.indices_recovery_max_bytes_per_sec-property'} (integer, Minimum: 40, Maximum: 400). Limits total inbound and outbound recovery traffic for each node. Applies to both peer recoveries as well as snapshot recoveries (i.e., restores from a snapshot). Defaults to 40mb.
- [`indices_recovery_max_concurrent_file_chunks`](#spec.userConfig.opensearch.indices_recovery_max_concurrent_file_chunks-property){: name='spec.userConfig.opensearch.indices_recovery_max_concurrent_file_chunks-property'} (integer, Minimum: 2, Maximum: 5). Number of file chunks sent in parallel for each recovery. Defaults to 2.
- [`override_main_response_version`](#spec.userConfig.opensearch.override_main_response_version-property){: name='spec.userConfig.opensearch.override_main_response_version-property'} (boolean). Compatibility mode sets OpenSearch to report its version as 7.10 so clients continue to work. Default is false.
- [`reindex_remote_whitelist`](#spec.userConfig.opensearch.reindex_remote_whitelist-property){: name='spec.userConfig.opensearch.reindex_remote_whitelist-property'} (array of strings, MaxItems: 32). Whitelisted addresses for reindexing. Changing this value will cause all OpenSearch instances to restart.
- [`script_max_compilations_rate`](#spec.userConfig.opensearch.script_max_compilations_rate-property){: name='spec.userConfig.opensearch.script_max_compilations_rate-property'} (string, MaxLength: 1024). Script compilation circuit breaker limits the number of inline script compilations within a period of time. Default is use-context.
- [`search_max_buckets`](#spec.userConfig.opensearch.search_max_buckets-property){: name='spec.userConfig.opensearch.search_max_buckets-property'} (integer, Minimum: 1, Maximum: 20000). Maximum number of aggregation buckets allowed in a single response. OpenSearch default value is used when this is not defined.
- [`thread_pool_analyze_queue_size`](#spec.userConfig.opensearch.thread_pool_analyze_queue_size-property){: name='spec.userConfig.opensearch.thread_pool_analyze_queue_size-property'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details.
- [`thread_pool_analyze_size`](#spec.userConfig.opensearch.thread_pool_analyze_size-property){: name='spec.userConfig.opensearch.thread_pool_analyze_size-property'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
- [`thread_pool_force_merge_size`](#spec.userConfig.opensearch.thread_pool_force_merge_size-property){: name='spec.userConfig.opensearch.thread_pool_force_merge_size-property'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
- [`thread_pool_get_queue_size`](#spec.userConfig.opensearch.thread_pool_get_queue_size-property){: name='spec.userConfig.opensearch.thread_pool_get_queue_size-property'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details.
- [`thread_pool_get_size`](#spec.userConfig.opensearch.thread_pool_get_size-property){: name='spec.userConfig.opensearch.thread_pool_get_size-property'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
- [`thread_pool_search_queue_size`](#spec.userConfig.opensearch.thread_pool_search_queue_size-property){: name='spec.userConfig.opensearch.thread_pool_search_queue_size-property'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details.
- [`thread_pool_search_size`](#spec.userConfig.opensearch.thread_pool_search_size-property){: name='spec.userConfig.opensearch.thread_pool_search_size-property'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
- [`thread_pool_search_throttled_queue_size`](#spec.userConfig.opensearch.thread_pool_search_throttled_queue_size-property){: name='spec.userConfig.opensearch.thread_pool_search_throttled_queue_size-property'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details.
- [`thread_pool_search_throttled_size`](#spec.userConfig.opensearch.thread_pool_search_throttled_size-property){: name='spec.userConfig.opensearch.thread_pool_search_throttled_size-property'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.
- [`thread_pool_write_queue_size`](#spec.userConfig.opensearch.thread_pool_write_queue_size-property){: name='spec.userConfig.opensearch.thread_pool_write_queue_size-property'} (integer, Minimum: 10, Maximum: 2000). Size for the thread pool queue. See documentation for exact details.
- [`thread_pool_write_size`](#spec.userConfig.opensearch.thread_pool_write_size-property){: name='spec.userConfig.opensearch.thread_pool_write_size-property'} (integer, Minimum: 1, Maximum: 128). Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.

### opensearch_dashboards {: #spec.userConfig.opensearch_dashboards }

OpenSearch Dashboards settings.

**Optional**

- [`enabled`](#spec.userConfig.opensearch_dashboards.enabled-property){: name='spec.userConfig.opensearch_dashboards.enabled-property'} (boolean). Enable or disable OpenSearch Dashboards.
- [`max_old_space_size`](#spec.userConfig.opensearch_dashboards.max_old_space_size-property){: name='spec.userConfig.opensearch_dashboards.max_old_space_size-property'} (integer, Minimum: 64, Maximum: 2048). Limits the maximum amount of memory (in MiB) the OpenSearch Dashboards process can use. This sets the max_old_space_size option of the nodejs running the OpenSearch Dashboards. Note: the memory reserved by OpenSearch Dashboards is not available for OpenSearch.
- [`opensearch_request_timeout`](#spec.userConfig.opensearch_dashboards.opensearch_request_timeout-property){: name='spec.userConfig.opensearch_dashboards.opensearch_request_timeout-property'} (integer, Minimum: 5000, Maximum: 120000). Timeout in milliseconds for requests made by OpenSearch Dashboards towards OpenSearch.

### private_access {: #spec.userConfig.private_access }

Allow access to selected service ports from private networks.

**Optional**

- [`opensearch`](#spec.userConfig.private_access.opensearch-property){: name='spec.userConfig.private_access.opensearch-property'} (boolean). Allow clients to connect to opensearch with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`opensearch_dashboards`](#spec.userConfig.private_access.opensearch_dashboards-property){: name='spec.userConfig.private_access.opensearch_dashboards-property'} (boolean). Allow clients to connect to opensearch_dashboards with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

Allow access to selected service components through Privatelink.

**Optional**

- [`opensearch`](#spec.userConfig.privatelink_access.opensearch-property){: name='spec.userConfig.privatelink_access.opensearch-property'} (boolean). Enable opensearch.
- [`opensearch_dashboards`](#spec.userConfig.privatelink_access.opensearch_dashboards-property){: name='spec.userConfig.privatelink_access.opensearch_dashboards-property'} (boolean). Enable opensearch_dashboards.
- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.

### public_access {: #spec.userConfig.public_access }

Allow access to selected service ports from the public Internet.

**Optional**

- [`opensearch`](#spec.userConfig.public_access.opensearch-property){: name='spec.userConfig.public_access.opensearch-property'} (boolean). Allow clients to connect to opensearch from the public internet for service nodes that are in a project VPC or another type of private network.
- [`opensearch_dashboards`](#spec.userConfig.public_access.opensearch_dashboards-property){: name='spec.userConfig.public_access.opensearch_dashboards-property'} (boolean). Allow clients to connect to opensearch_dashboards from the public internet for service nodes that are in a project VPC or another type of private network.
- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.

