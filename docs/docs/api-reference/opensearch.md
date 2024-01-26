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
    prefix: MY_SECRET_PREFIX_
    annotations:
      foo: bar
    labels:
      baz: egg

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: startup-4
  disk_space: 80Gib

  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
```

## OpenSearch {: #OpenSearch }

OpenSearch is the Schema for the opensearches API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `OpenSearch`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). OpenSearchSpec defines the desired state of OpenSearch. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`OpenSearch`](#OpenSearch)._

OpenSearchSpec defines the desired state of OpenSearch.

**Required**

- [`plan`](#spec.plan-property){: name='spec.plan-property'} (string, MaxLength: 128). Subscription plan.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, MaxLength: 63, Format: `^[a-zA-Z0-9_-]*$`). Target project.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, MaxLength: 256). Cloud the service runs in.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Information regarding secret creation. Exposed keys: `OPENSEARCH_HOST`, `OPENSEARCH_PORT`, `OPENSEARCH_USER`, `OPENSEARCH_PASSWORD`. See below for [nested schema](#spec.connInfoSecretTarget).
- [`connInfoSecretTargetDisabled`](#spec.connInfoSecretTargetDisabled-property){: name='spec.connInfoSecretTargetDisabled-property'} (boolean, Immutable). When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
- [`disk_space`](#spec.disk_space-property){: name='spec.disk_space-property'} (string, Format: `^[1-9][0-9]*(GiB|G)*`). The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
- [`maintenanceWindowDow`](#spec.maintenanceWindowDow-property){: name='spec.maintenanceWindowDow-property'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
- [`maintenanceWindowTime`](#spec.maintenanceWindowTime-property){: name='spec.maintenanceWindowTime-property'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
- [`projectVPCRef`](#spec.projectVPCRef-property){: name='spec.projectVPCRef-property'} (object). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See below for [nested schema](#spec.projectVPCRef).
- [`projectVpcId`](#spec.projectVpcId-property){: name='spec.projectVpcId-property'} (string, MaxLength: 36). Identifier of the VPC the service should be in, if any.
- [`serviceIntegrations`](#spec.serviceIntegrations-property){: name='spec.serviceIntegrations-property'} (array of objects, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See below for [nested schema](#spec.serviceIntegrations).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize services.
- [`technicalEmails`](#spec.technicalEmails-property){: name='spec.technicalEmails-property'} (array of objects, MaxItems: 10). Defines the email addresses that will receive alerts about upcoming maintenance updates or warnings about service instability. See below for [nested schema](#spec.technicalEmails).
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services.
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). OpenSearch specific user configuration options. See below for [nested schema](#spec.userConfig).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

_Appears on [`spec`](#spec)._

Information regarding secret creation. Exposed keys: `OPENSEARCH_HOST`, `OPENSEARCH_PORT`, `OPENSEARCH_USER`, `OPENSEARCH_PASSWORD`.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string). Name of the secret resource to be created. By default, is equal to the resource name.

**Optional**

- [`annotations`](#spec.connInfoSecretTarget.annotations-property){: name='spec.connInfoSecretTarget.annotations-property'} (object, AdditionalProperties: string). Annotations added to the secret.
- [`labels`](#spec.connInfoSecretTarget.labels-property){: name='spec.connInfoSecretTarget.labels-property'} (object, AdditionalProperties: string). Labels added to the secret.
- [`prefix`](#spec.connInfoSecretTarget.prefix-property){: name='spec.connInfoSecretTarget.prefix-property'} (string). Prefix for the secret's keys. Added "as is" without any transformations. By default, is equal to the kind name in uppercase + underscore, e.g. `KAFKA_`, `REDIS_`, etc.

## projectVPCRef {: #spec.projectVPCRef }

_Appears on [`spec`](#spec)._

ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically.

**Required**

- [`name`](#spec.projectVPCRef.name-property){: name='spec.projectVPCRef.name-property'} (string, MinLength: 1).

**Optional**

- [`namespace`](#spec.projectVPCRef.namespace-property){: name='spec.projectVPCRef.namespace-property'} (string, MinLength: 1).

## serviceIntegrations {: #spec.serviceIntegrations }

_Appears on [`spec`](#spec)._

Service integrations to specify when creating a service. Not applied after initial service creation.

**Required**

- [`integrationType`](#spec.serviceIntegrations.integrationType-property){: name='spec.serviceIntegrations.integrationType-property'} (string, Enum: `read_replica`).
- [`sourceServiceName`](#spec.serviceIntegrations.sourceServiceName-property){: name='spec.serviceIntegrations.sourceServiceName-property'} (string, MinLength: 1, MaxLength: 64).

## technicalEmails {: #spec.technicalEmails }

_Appears on [`spec`](#spec)._

Defines the email addresses that will receive alerts about upcoming maintenance updates or warnings about service instability.

**Required**

- [`email`](#spec.technicalEmails.email-property){: name='spec.technicalEmails.email-property'} (string, Format: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`). Email address.

## userConfig {: #spec.userConfig }

_Appears on [`spec`](#spec)._

OpenSearch specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Deprecated. Additional Cloud Regions for Backup Replication.
- [`custom_domain`](#spec.userConfig.custom_domain-property){: name='spec.userConfig.custom_domain-property'} (string, MaxLength: 255). Serve the web frontend using a custom CNAME pointing to the Aiven DNS name.
- [`disable_replication_factor_adjustment`](#spec.userConfig.disable_replication_factor_adjustment-property){: name='spec.userConfig.disable_replication_factor_adjustment-property'} (boolean). DEPRECATED: Disable automatic replication factor adjustment for multi-node services. By default, Aiven ensures all indexes are replicated at least to two nodes. Note: Due to potential data loss in case of losing a service node, this setting can no longer be activated.
- [`index_patterns`](#spec.userConfig.index_patterns-property){: name='spec.userConfig.index_patterns-property'} (array of objects, MaxItems: 512). Index patterns. See below for [nested schema](#spec.userConfig.index_patterns).
- [`index_template`](#spec.userConfig.index_template-property){: name='spec.userConfig.index_template-property'} (object). Template settings for all new indexes. See below for [nested schema](#spec.userConfig.index_template).
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`keep_index_refresh_interval`](#spec.userConfig.keep_index_refresh_interval-property){: name='spec.userConfig.keep_index_refresh_interval-property'} (boolean). Aiven automation resets index.refresh_interval to default value for every index to be sure that indices are always visible to search. If it doesn't fit your case, you can disable this by setting up this flag to true.
- [`max_index_count`](#spec.userConfig.max_index_count-property){: name='spec.userConfig.max_index_count-property'} (integer, Minimum: 0). DEPRECATED: use index_patterns instead.
- [`openid`](#spec.userConfig.openid-property){: name='spec.userConfig.openid-property'} (object). OpenSearch OpenID Connect Configuration. See below for [nested schema](#spec.userConfig.openid).
- [`opensearch`](#spec.userConfig.opensearch-property){: name='spec.userConfig.opensearch-property'} (object). OpenSearch settings. See below for [nested schema](#spec.userConfig.opensearch).
- [`opensearch_dashboards`](#spec.userConfig.opensearch_dashboards-property){: name='spec.userConfig.opensearch_dashboards-property'} (object). OpenSearch Dashboards settings. See below for [nested schema](#spec.userConfig.opensearch_dashboards).
- [`opensearch_version`](#spec.userConfig.opensearch_version-property){: name='spec.userConfig.opensearch_version-property'} (string, Enum: `1`, `2`). OpenSearch major version.
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`project_to_fork_from`](#spec.userConfig.project_to_fork_from-property){: name='spec.userConfig.project_to_fork_from-property'} (string, Immutable, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created.
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`recovery_basebackup_name`](#spec.userConfig.recovery_basebackup_name-property){: name='spec.userConfig.recovery*basebackup_name-property'} (string, Pattern: `^[a-zA-Z0-9-*:.]+$`, MaxLength: 128). Name of the basebackup to restore in forked service.
- [`saml`](#spec.userConfig.saml-property){: name='spec.userConfig.saml-property'} (object). OpenSearch SAML configuration. See below for [nested schema](#spec.userConfig.saml).
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`service_to_fork_from`](#spec.userConfig.service_to_fork_from-property){: name='spec.userConfig.service_to_fork_from-property'} (string, Immutable, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

### index_patterns {: #spec.userConfig.index_patterns }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Index patterns.

**Required**

- [`max_index_count`](#spec.userConfig.index_patterns.max_index_count-property){: name='spec.userConfig.index_patterns.max_index_count-property'} (integer, Minimum: 0). Maximum number of indexes to keep.
- [`pattern`](#spec.userConfig.index_patterns.pattern-property){: name='spec.userConfig.index*patterns.pattern-property'} (string, Pattern: `^[A-Za-z0-9-*.\*?]+$`, MaxLength: 1024). fnmatch pattern.

**Optional**

- [`sorting_algorithm`](#spec.userConfig.index_patterns.sorting_algorithm-property){: name='spec.userConfig.index_patterns.sorting_algorithm-property'} (string, Enum: `alphabetical`, `creation_date`). Deletion sorting algorithm.

### index_template {: #spec.userConfig.index_template }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Template settings for all new indexes.

**Optional**

- [`mapping_nested_objects_limit`](#spec.userConfig.index_template.mapping_nested_objects_limit-property){: name='spec.userConfig.index_template.mapping_nested_objects_limit-property'} (integer, Minimum: 0, Maximum: 100000). The maximum number of nested JSON objects that a single document can contain across all nested types. This limit helps to prevent out of memory errors when a document contains too many nested objects. Default is 10000.
- [`number_of_replicas`](#spec.userConfig.index_template.number_of_replicas-property){: name='spec.userConfig.index_template.number_of_replicas-property'} (integer, Minimum: 0, Maximum: 29). The number of replicas each primary shard has.
- [`number_of_shards`](#spec.userConfig.index_template.number_of_shards-property){: name='spec.userConfig.index_template.number_of_shards-property'} (integer, Minimum: 1, Maximum: 1024). The number of primary shards that an index should have.

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### openid {: #spec.userConfig.openid }

_Appears on [`spec.userConfig`](#spec.userConfig)._

OpenSearch OpenID Connect Configuration.

**Required**

- [`client_id`](#spec.userConfig.openid.client_id-property){: name='spec.userConfig.openid.client_id-property'} (string, MinLength: 1, MaxLength: 1024). The ID of the OpenID Connect client configured in your IdP. Required.
- [`client_secret`](#spec.userConfig.openid.client_secret-property){: name='spec.userConfig.openid.client_secret-property'} (string, MinLength: 1, MaxLength: 1024). The client secret of the OpenID Connect client configured in your IdP. Required.
- [`connect_url`](#spec.userConfig.openid.connect_url-property){: name='spec.userConfig.openid.connect_url-property'} (string, MaxLength: 2048). The URL of your IdP where the Security plugin can find the OpenID Connect metadata/configuration settings.
- [`enabled`](#spec.userConfig.openid.enabled-property){: name='spec.userConfig.openid.enabled-property'} (boolean). Enables or disables OpenID Connect authentication for OpenSearch. When enabled, users can authenticate using OpenID Connect with an Identity Provider.

**Optional**

- [`header`](#spec.userConfig.openid.header-property){: name='spec.userConfig.openid.header-property'} (string, MinLength: 1, MaxLength: 1024). HTTP header name of the JWT token. Optional. Default is Authorization.
- [`jwt_header`](#spec.userConfig.openid.jwt_header-property){: name='spec.userConfig.openid.jwt_header-property'} (string, MinLength: 1, MaxLength: 1024). The HTTP header that stores the token. Typically the Authorization header with the Bearer schema: Authorization: Bearer <token>. Optional. Default is Authorization.
- [`jwt_url_parameter`](#spec.userConfig.openid.jwt_url_parameter-property){: name='spec.userConfig.openid.jwt_url_parameter-property'} (string, MinLength: 1, MaxLength: 1024). If the token is not transmitted in the HTTP header, but as an URL parameter, define the name of the parameter here. Optional.
- [`refresh_rate_limit_count`](#spec.userConfig.openid.refresh_rate_limit_count-property){: name='spec.userConfig.openid.refresh_rate_limit_count-property'} (integer, Minimum: 10). The maximum number of unknown key IDs in the time frame. Default is 10. Optional.
- [`refresh_rate_limit_time_window_ms`](#spec.userConfig.openid.refresh_rate_limit_time_window_ms-property){: name='spec.userConfig.openid.refresh_rate_limit_time_window_ms-property'} (integer, Minimum: 10000). The time frame to use when checking the maximum number of unknown key IDs, in milliseconds. Optional.Default is 10000 (10 seconds).
- [`roles_key`](#spec.userConfig.openid.roles_key-property){: name='spec.userConfig.openid.roles_key-property'} (string, MinLength: 1, MaxLength: 1024). The key in the JSON payload that stores the user’s roles. The value of this key must be a comma-separated list of roles. Required only if you want to use roles in the JWT.
- [`scope`](#spec.userConfig.openid.scope-property){: name='spec.userConfig.openid.scope-property'} (string, MinLength: 1, MaxLength: 1024). The scope of the identity token issued by the IdP. Optional. Default is openid profile email address phone.
- [`subject_key`](#spec.userConfig.openid.subject_key-property){: name='spec.userConfig.openid.subject_key-property'} (string, MinLength: 1, MaxLength: 1024). The key in the JSON payload that stores the user’s name. If not defined, the subject registered claim is used. Most IdP providers use the preferred_username claim. Optional.

### opensearch {: #spec.userConfig.opensearch }

_Appears on [`spec.userConfig`](#spec.userConfig)._

OpenSearch settings.

**Optional**

- [`action_auto_create_index_enabled`](#spec.userConfig.opensearch.action_auto_create_index_enabled-property){: name='spec.userConfig.opensearch.action_auto_create_index_enabled-property'} (boolean). Explicitly allow or block automatic creation of indices. Defaults to true.
- [`action_destructive_requires_name`](#spec.userConfig.opensearch.action_destructive_requires_name-property){: name='spec.userConfig.opensearch.action_destructive_requires_name-property'} (boolean). Require explicit index names when deleting.
- [`auth_failure_listeners`](#spec.userConfig.opensearch.auth_failure_listeners-property){: name='spec.userConfig.opensearch.auth_failure_listeners-property'} (object). Opensearch Security Plugin Settings. See below for [nested schema](#spec.userConfig.opensearch.auth_failure_listeners).
- [`cluster_max_shards_per_node`](#spec.userConfig.opensearch.cluster_max_shards_per_node-property){: name='spec.userConfig.opensearch.cluster_max_shards_per_node-property'} (integer, Minimum: 100, Maximum: 10000). Controls the number of shards allowed in the cluster per data node.
- [`cluster_routing_allocation_node_concurrent_recoveries`](#spec.userConfig.opensearch.cluster_routing_allocation_node_concurrent_recoveries-property){: name='spec.userConfig.opensearch.cluster_routing_allocation_node_concurrent_recoveries-property'} (integer, Minimum: 2, Maximum: 16). How many concurrent incoming/outgoing shard recoveries (normally replicas) are allowed to happen on a node. Defaults to 2.
- [`email_sender_name`](#spec.userConfig.opensearch.email_sender_name-property){: name='spec.userConfig.opensearch.email*sender_name-property'} (string, Pattern: `^[a-zA-Z0-9-*]+$`, MaxLength: 40). Sender name placeholder to be used in Opensearch Dashboards and Opensearch keystore.
- [`email_sender_password`](#spec.userConfig.opensearch.email_sender_password-property){: name='spec.userConfig.opensearch.email_sender_password-property'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 1024). Sender password for Opensearch alerts to authenticate with SMTP server.
- [`email_sender_username`](#spec.userConfig.opensearch.email_sender_username-property){: name='spec.userConfig.opensearch.email_sender_username-property'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 320). Sender username for Opensearch alerts.
- [`enable_security_audit`](#spec.userConfig.opensearch.enable_security_audit-property){: name='spec.userConfig.opensearch.enable_security_audit-property'} (boolean). Enable/Disable security audit.
- [`http_max_content_length`](#spec.userConfig.opensearch.http_max_content_length-property){: name='spec.userConfig.opensearch.http_max_content_length-property'} (integer, Minimum: 1, Maximum: 2147483647). Maximum content length for HTTP requests to the OpenSearch HTTP API, in bytes.
- [`http_max_header_size`](#spec.userConfig.opensearch.http_max_header_size-property){: name='spec.userConfig.opensearch.http_max_header_size-property'} (integer, Minimum: 1024, Maximum: 262144). The max size of allowed headers, in bytes.
- [`http_max_initial_line_length`](#spec.userConfig.opensearch.http_max_initial_line_length-property){: name='spec.userConfig.opensearch.http_max_initial_line_length-property'} (integer, Minimum: 1024, Maximum: 65536). The max length of an HTTP URL, in bytes.
- [`indices_fielddata_cache_size`](#spec.userConfig.opensearch.indices_fielddata_cache_size-property){: name='spec.userConfig.opensearch.indices_fielddata_cache_size-property'} (integer, Minimum: 3, Maximum: 100). Relative amount. Maximum amount of heap memory used for field data cache. This is an expert setting; decreasing the value too much will increase overhead of loading field data; too much memory used for field data cache will decrease amount of heap available for other operations.
- [`indices_memory_index_buffer_size`](#spec.userConfig.opensearch.indices_memory_index_buffer_size-property){: name='spec.userConfig.opensearch.indices_memory_index_buffer_size-property'} (integer, Minimum: 3, Maximum: 40). Percentage value. Default is 10%. Total amount of heap used for indexing buffer, before writing segments to disk. This is an expert setting. Too low value will slow down indexing; too high value will increase indexing performance but causes performance issues for query performance.
- [`indices_memory_max_index_buffer_size`](#spec.userConfig.opensearch.indices_memory_max_index_buffer_size-property){: name='spec.userConfig.opensearch.indices_memory_max_index_buffer_size-property'} (integer, Minimum: 3, Maximum: 2048). Absolute value. Default is unbound. Doesn't work without indices.memory.index_buffer_size. Maximum amount of heap used for query cache, an absolute indices.memory.index_buffer_size maximum hard limit.
- [`indices_memory_min_index_buffer_size`](#spec.userConfig.opensearch.indices_memory_min_index_buffer_size-property){: name='spec.userConfig.opensearch.indices_memory_min_index_buffer_size-property'} (integer, Minimum: 3, Maximum: 2048). Absolute value. Default is 48mb. Doesn't work without indices.memory.index_buffer_size. Minimum amount of heap used for query cache, an absolute indices.memory.index_buffer_size minimal hard limit.
- [`indices_queries_cache_size`](#spec.userConfig.opensearch.indices_queries_cache_size-property){: name='spec.userConfig.opensearch.indices_queries_cache_size-property'} (integer, Minimum: 3, Maximum: 40). Percentage value. Default is 10%. Maximum amount of heap used for query cache. This is an expert setting. Too low value will decrease query performance and increase performance for other operations; too high value will cause issues with other OpenSearch functionality.
- [`indices_query_bool_max_clause_count`](#spec.userConfig.opensearch.indices_query_bool_max_clause_count-property){: name='spec.userConfig.opensearch.indices_query_bool_max_clause_count-property'} (integer, Minimum: 64, Maximum: 4096). Maximum number of clauses Lucene BooleanQuery can have. The default value (1024) is relatively high, and increasing it may cause performance issues. Investigate other approaches first before increasing this value.
- [`indices_recovery_max_bytes_per_sec`](#spec.userConfig.opensearch.indices_recovery_max_bytes_per_sec-property){: name='spec.userConfig.opensearch.indices_recovery_max_bytes_per_sec-property'} (integer, Minimum: 40, Maximum: 400). Limits total inbound and outbound recovery traffic for each node. Applies to both peer recoveries as well as snapshot recoveries (i.e., restores from a snapshot). Defaults to 40mb.
- [`indices_recovery_max_concurrent_file_chunks`](#spec.userConfig.opensearch.indices_recovery_max_concurrent_file_chunks-property){: name='spec.userConfig.opensearch.indices_recovery_max_concurrent_file_chunks-property'} (integer, Minimum: 2, Maximum: 5). Number of file chunks sent in parallel for each recovery. Defaults to 2.
- [`ism_enabled`](#spec.userConfig.opensearch.ism_enabled-property){: name='spec.userConfig.opensearch.ism_enabled-property'} (boolean). Specifies whether ISM is enabled or not.
- [`ism_history_enabled`](#spec.userConfig.opensearch.ism_history_enabled-property){: name='spec.userConfig.opensearch.ism_history_enabled-property'} (boolean). Specifies whether audit history is enabled or not. The logs from ISM are automatically indexed to a logs document.
- [`ism_history_max_age`](#spec.userConfig.opensearch.ism_history_max_age-property){: name='spec.userConfig.opensearch.ism_history_max_age-property'} (integer, Minimum: 1, Maximum: 2147483647). The maximum age before rolling over the audit history index in hours.
- [`ism_history_max_docs`](#spec.userConfig.opensearch.ism_history_max_docs-property){: name='spec.userConfig.opensearch.ism_history_max_docs-property'} (integer, Minimum: 1). The maximum number of documents before rolling over the audit history index.
- [`ism_history_rollover_check_period`](#spec.userConfig.opensearch.ism_history_rollover_check_period-property){: name='spec.userConfig.opensearch.ism_history_rollover_check_period-property'} (integer, Minimum: 1, Maximum: 2147483647). The time between rollover checks for the audit history index in hours.
- [`ism_history_rollover_retention_period`](#spec.userConfig.opensearch.ism_history_rollover_retention_period-property){: name='spec.userConfig.opensearch.ism_history_rollover_retention_period-property'} (integer, Minimum: 1, Maximum: 2147483647). How long audit history indices are kept in days.
- [`override_main_response_version`](#spec.userConfig.opensearch.override_main_response_version-property){: name='spec.userConfig.opensearch.override_main_response_version-property'} (boolean). Compatibility mode sets OpenSearch to report its version as 7.10 so clients continue to work. Default is false.
- [`reindex_remote_whitelist`](#spec.userConfig.opensearch.reindex_remote_whitelist-property){: name='spec.userConfig.opensearch.reindex_remote_whitelist-property'} (array of strings, MaxItems: 32). Whitelisted addresses for reindexing. Changing this value will cause all OpenSearch instances to restart.
- [`script_max_compilations_rate`](#spec.userConfig.opensearch.script_max_compilations_rate-property){: name='spec.userConfig.opensearch.script_max_compilations_rate-property'} (string, MaxLength: 1024). Script compilation circuit breaker limits the number of inline script compilations within a period of time. Default is use-context.
- [`search_max_buckets`](#spec.userConfig.opensearch.search_max_buckets-property){: name='spec.userConfig.opensearch.search_max_buckets-property'} (integer, Minimum: 1, Maximum: 1000000). Maximum number of aggregation buckets allowed in a single response. OpenSearch default value is used when this is not defined.
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

#### auth_failure_listeners {: #spec.userConfig.opensearch.auth_failure_listeners }

_Appears on [`spec.userConfig.opensearch`](#spec.userConfig.opensearch)._

Opensearch Security Plugin Settings.

**Optional**

- [`internal_authentication_backend_limiting`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting-property'} (object). See below for [nested schema](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting).
- [`ip_rate_limiting`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting-property'} (object). IP address rate limiting settings. See below for [nested schema](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting).

##### internal_authentication_backend_limiting {: #spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting }

_Appears on [`spec.userConfig.opensearch.auth_failure_listeners`](#spec.userConfig.opensearch.auth_failure_listeners)._

**Optional**

- [`allowed_tries`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.allowed_tries-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.allowed_tries-property'} (integer, Minimum: 0, Maximum: 2147483647). The number of login attempts allowed before login is blocked.
- [`authentication_backend`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.authentication_backend-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.authentication_backend-property'} (string, Enum: `internal`, MaxLength: 1024). internal_authentication_backend_limiting.authentication_backend.
- [`block_expiry_seconds`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.block_expiry_seconds-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.block_expiry_seconds-property'} (integer, Minimum: 0, Maximum: 2147483647). The duration of time that login remains blocked after a failed login.
- [`max_blocked_clients`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.max_blocked_clients-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.max_blocked_clients-property'} (integer, Minimum: 0, Maximum: 2147483647). internal_authentication_backend_limiting.max_blocked_clients.
- [`max_tracked_clients`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.max_tracked_clients-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.max_tracked_clients-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum number of tracked IP addresses that have failed login.
- [`time_window_seconds`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.time_window_seconds-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.time_window_seconds-property'} (integer, Minimum: 0, Maximum: 2147483647). The window of time in which the value for `allowed_tries` is enforced.
- [`type`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.type-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.type-property'} (string, Enum: `username`, MaxLength: 1024). internal_authentication_backend_limiting.type.

##### ip_rate_limiting {: #spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting }

_Appears on [`spec.userConfig.opensearch.auth_failure_listeners`](#spec.userConfig.opensearch.auth_failure_listeners)._

IP address rate limiting settings.

**Optional**

- [`allowed_tries`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.allowed_tries-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.allowed_tries-property'} (integer, Minimum: 1, Maximum: 2147483647). The number of login attempts allowed before login is blocked.
- [`block_expiry_seconds`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.block_expiry_seconds-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.block_expiry_seconds-property'} (integer, Minimum: 1, Maximum: 36000). The duration of time that login remains blocked after a failed login.
- [`max_blocked_clients`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.max_blocked_clients-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.max_blocked_clients-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum number of blocked IP addresses.
- [`max_tracked_clients`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.max_tracked_clients-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.max_tracked_clients-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum number of tracked IP addresses that have failed login.
- [`time_window_seconds`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.time_window_seconds-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.time_window_seconds-property'} (integer, Minimum: 1, Maximum: 36000). The window of time in which the value for `allowed_tries` is enforced.
- [`type`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.type-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.type-property'} (string, Enum: `ip`, MaxLength: 1024). The type of rate limiting.

### opensearch_dashboards {: #spec.userConfig.opensearch_dashboards }

_Appears on [`spec.userConfig`](#spec.userConfig)._

OpenSearch Dashboards settings.

**Optional**

- [`enabled`](#spec.userConfig.opensearch_dashboards.enabled-property){: name='spec.userConfig.opensearch_dashboards.enabled-property'} (boolean). Enable or disable OpenSearch Dashboards.
- [`max_old_space_size`](#spec.userConfig.opensearch_dashboards.max_old_space_size-property){: name='spec.userConfig.opensearch_dashboards.max_old_space_size-property'} (integer, Minimum: 64, Maximum: 2048). Limits the maximum amount of memory (in MiB) the OpenSearch Dashboards process can use. This sets the max_old_space_size option of the nodejs running the OpenSearch Dashboards. Note: the memory reserved by OpenSearch Dashboards is not available for OpenSearch.
- [`opensearch_request_timeout`](#spec.userConfig.opensearch_dashboards.opensearch_request_timeout-property){: name='spec.userConfig.opensearch_dashboards.opensearch_request_timeout-property'} (integer, Minimum: 5000, Maximum: 120000). Timeout in milliseconds for requests made by OpenSearch Dashboards towards OpenSearch.

### private_access {: #spec.userConfig.private_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from private networks.

**Optional**

- [`opensearch`](#spec.userConfig.private_access.opensearch-property){: name='spec.userConfig.private_access.opensearch-property'} (boolean). Allow clients to connect to opensearch with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`opensearch_dashboards`](#spec.userConfig.private_access.opensearch_dashboards-property){: name='spec.userConfig.private_access.opensearch_dashboards-property'} (boolean). Allow clients to connect to opensearch_dashboards with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service components through Privatelink.

**Optional**

- [`opensearch`](#spec.userConfig.privatelink_access.opensearch-property){: name='spec.userConfig.privatelink_access.opensearch-property'} (boolean). Enable opensearch.
- [`opensearch_dashboards`](#spec.userConfig.privatelink_access.opensearch_dashboards-property){: name='spec.userConfig.privatelink_access.opensearch_dashboards-property'} (boolean). Enable opensearch_dashboards.
- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Optional**

- [`opensearch`](#spec.userConfig.public_access.opensearch-property){: name='spec.userConfig.public_access.opensearch-property'} (boolean). Allow clients to connect to opensearch from the public internet for service nodes that are in a project VPC or another type of private network.
- [`opensearch_dashboards`](#spec.userConfig.public_access.opensearch_dashboards-property){: name='spec.userConfig.public_access.opensearch_dashboards-property'} (boolean). Allow clients to connect to opensearch_dashboards from the public internet for service nodes that are in a project VPC or another type of private network.
- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.

### saml {: #spec.userConfig.saml }

_Appears on [`spec.userConfig`](#spec.userConfig)._

OpenSearch SAML configuration.

**Required**

- [`enabled`](#spec.userConfig.saml.enabled-property){: name='spec.userConfig.saml.enabled-property'} (boolean). Enables or disables SAML-based authentication for OpenSearch. When enabled, users can authenticate using SAML with an Identity Provider.
- [`idp_entity_id`](#spec.userConfig.saml.idp_entity_id-property){: name='spec.userConfig.saml.idp_entity_id-property'} (string, MinLength: 1, MaxLength: 1024). The unique identifier for the Identity Provider (IdP) entity that is used for SAML authentication. This value is typically provided by the IdP.
- [`idp_metadata_url`](#spec.userConfig.saml.idp_metadata_url-property){: name='spec.userConfig.saml.idp_metadata_url-property'} (string, MinLength: 1, MaxLength: 2048). The URL of the SAML metadata for the Identity Provider (IdP). This is used to configure SAML-based authentication with the IdP.
- [`sp_entity_id`](#spec.userConfig.saml.sp_entity_id-property){: name='spec.userConfig.saml.sp_entity_id-property'} (string, MinLength: 1, MaxLength: 1024). The unique identifier for the Service Provider (SP) entity that is used for SAML authentication. This value is typically provided by the SP.

**Optional**

- [`idp_pemtrustedcas_content`](#spec.userConfig.saml.idp_pemtrustedcas_content-property){: name='spec.userConfig.saml.idp_pemtrustedcas_content-property'} (string, MaxLength: 16384). This parameter specifies the PEM-encoded root certificate authority (CA) content for the SAML identity provider (IdP) server verification. The root CA content is used to verify the SSL/TLS certificate presented by the server.
- [`roles_key`](#spec.userConfig.saml.roles_key-property){: name='spec.userConfig.saml.roles_key-property'} (string, MinLength: 1, MaxLength: 256). Optional. Specifies the attribute in the SAML response where role information is stored, if available. Role attributes are not required for SAML authentication, but can be included in SAML assertions by most Identity Providers (IdPs) to determine user access levels or permissions.
- [`subject_key`](#spec.userConfig.saml.subject_key-property){: name='spec.userConfig.saml.subject_key-property'} (string, MinLength: 1, MaxLength: 256). Optional. Specifies the attribute in the SAML response where the subject identifier is stored. If not configured, the NameID attribute is used by default.
