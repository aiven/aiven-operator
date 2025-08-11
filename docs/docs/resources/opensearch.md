---
title: "OpenSearch"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

```yaml linenums="1"
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
  disk_space: 80GiB

  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `OpenSearch`:

```shell
kubectl get opensearches my-os
```

The output is similar to the following:
```shell
Name     Project             Region                 Plan         State      
my-os    my-aiven-project    google-europe-west1    startup-4    RUNNING    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret os-secret
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret os-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"OPENSEARCH_HOST": "<secret>",
	"OPENSEARCH_PORT": "<secret>",
	"OPENSEARCH_USER": "<secret>",
	"OPENSEARCH_PASSWORD": "<secret>",
	"OPENSEARCH_URI": "<secret>",
}
```

---

## OpenSearch {: #OpenSearch }

OpenSearch is the Schema for the opensearches API.

!!! Info "Exposes secret keys"

    `OPENSEARCH_HOST`, `OPENSEARCH_PORT`, `OPENSEARCH_USER`, `OPENSEARCH_PASSWORD`, `OPENSEARCH_URI`.

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
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, MaxLength: 256). Cloud the service runs in.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Secret configuration. See below for [nested schema](#spec.connInfoSecretTarget).
- [`connInfoSecretTargetDisabled`](#spec.connInfoSecretTargetDisabled-property){: name='spec.connInfoSecretTargetDisabled-property'} (boolean, Immutable). When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
- [`disk_space`](#spec.disk_space-property){: name='spec.disk_space-property'} (string, Pattern: `(?i)^[1-9][0-9]*(GiB|G)?$`). The disk space of the service, possible values depend on the service type, the cloud provider and the project.
    Reducing will result in the service re-balancing.
    The removal of this field does not change the value.
- [`maintenanceWindowDow`](#spec.maintenanceWindowDow-property){: name='spec.maintenanceWindowDow-property'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
- [`maintenanceWindowTime`](#spec.maintenanceWindowTime-property){: name='spec.maintenanceWindowTime-property'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
- [`powered`](#spec.powered-property){: name='spec.powered-property'} (boolean, Default value: `true`). Determines the power state of the service. When `true` (default), the service is running.
    When `false`, the service is powered off.
    For more information please see [Aiven documentation](https://aiven.io/docs/platform/concepts/service-power-cycle).
    Note that:
    - When set to `false` the annotation `controllers.aiven.io/instance-is-running` is also set to `false`.
    - Services cannot be created in a powered off state. The value is ignored during creation.
    - It is highly recommended to not run dependent resources when the service is powered off.
      Creating a new resource or updating an existing resource that depends on a powered off service will result in an error.
      Existing resources will need to be manually recreated after the service is powered on.
    - Existing secrets will not be updated or removed when the service is powered off.
    - For Kafka services with backups: Topic configuration, schemas and connectors are all backed up, but not the data in topics. All topic data is lost on power off.
    - For Kafka services without backups: Topic configurations including all topic data is lost on power off.
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

Secret configuration.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string, Immutable). Name of the secret resource to be created. By default, it is equal to the resource name.

**Optional**

- [`annotations`](#spec.connInfoSecretTarget.annotations-property){: name='spec.connInfoSecretTarget.annotations-property'} (object, AdditionalProperties: string). Annotations added to the secret.
- [`labels`](#spec.connInfoSecretTarget.labels-property){: name='spec.connInfoSecretTarget.labels-property'} (object, AdditionalProperties: string). Labels added to the secret.
- [`prefix`](#spec.connInfoSecretTarget.prefix-property){: name='spec.connInfoSecretTarget.prefix-property'} (string). Prefix for the secret's keys.
    Added "as is" without any transformations.
    By default, is equal to the kind name in uppercase + underscore, e.g. `KAFKA_`, `REDIS_`, etc.

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

- [`email`](#spec.technicalEmails.email-property){: name='spec.technicalEmails.email-property'} (string). Email address.

## userConfig {: #spec.userConfig }

_Appears on [`spec`](#spec)._

OpenSearch specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Additional Cloud Regions for Backup Replication.
- [`azure_migration`](#spec.userConfig.azure_migration-property){: name='spec.userConfig.azure_migration-property'} (object). Azure migration settings. See below for [nested schema](#spec.userConfig.azure_migration).
- [`custom_domain`](#spec.userConfig.custom_domain-property){: name='spec.userConfig.custom_domain-property'} (string, MaxLength: 255). Serve the web frontend using a custom CNAME pointing to the Aiven DNS name.
- [`disable_replication_factor_adjustment`](#spec.userConfig.disable_replication_factor_adjustment-property){: name='spec.userConfig.disable_replication_factor_adjustment-property'} (boolean). Disable automatic replication factor adjustment for multi-node services. By default, Aiven ensures all indexes are replicated at least to two nodes. Note: Due to potential data loss in case of losing a service node, this setting can not be activated unless specifically allowed for the project.
- [`gcs_migration`](#spec.userConfig.gcs_migration-property){: name='spec.userConfig.gcs_migration-property'} (object). Google Cloud Storage migration settings. See below for [nested schema](#spec.userConfig.gcs_migration).
- [`index_patterns`](#spec.userConfig.index_patterns-property){: name='spec.userConfig.index_patterns-property'} (array of objects, MaxItems: 512). Index patterns. See below for [nested schema](#spec.userConfig.index_patterns).
- [`index_rollup`](#spec.userConfig.index_rollup-property){: name='spec.userConfig.index_rollup-property'} (object). Index rollup settings. See below for [nested schema](#spec.userConfig.index_rollup).
- [`index_template`](#spec.userConfig.index_template-property){: name='spec.userConfig.index_template-property'} (object). Template settings for all new indexes. See below for [nested schema](#spec.userConfig.index_template).
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 8000). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`keep_index_refresh_interval`](#spec.userConfig.keep_index_refresh_interval-property){: name='spec.userConfig.keep_index_refresh_interval-property'} (boolean). Aiven automation resets index.refresh_interval to default value for every index to be sure that indices are always visible to search. If it doesn't fit your case, you can disable this by setting up this flag to true.
- [`max_index_count`](#spec.userConfig.max_index_count-property){: name='spec.userConfig.max_index_count-property'} (integer, Minimum: 0). DEPRECATED: use index_patterns instead.
- [`openid`](#spec.userConfig.openid-property){: name='spec.userConfig.openid-property'} (object). OpenSearch OpenID Connect Configuration. See below for [nested schema](#spec.userConfig.openid).
- [`opensearch`](#spec.userConfig.opensearch-property){: name='spec.userConfig.opensearch-property'} (object). OpenSearch settings. See below for [nested schema](#spec.userConfig.opensearch).
- [`opensearch_dashboards`](#spec.userConfig.opensearch_dashboards-property){: name='spec.userConfig.opensearch_dashboards-property'} (object). OpenSearch Dashboards settings. See below for [nested schema](#spec.userConfig.opensearch_dashboards).
- [`opensearch_version`](#spec.userConfig.opensearch_version-property){: name='spec.userConfig.opensearch_version-property'} (string, Enum: `1`, `2`). OpenSearch major version.
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`project_to_fork_from`](#spec.userConfig.project_to_fork_from-property){: name='spec.userConfig.project_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created.
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`recovery_basebackup_name`](#spec.userConfig.recovery_basebackup_name-property){: name='spec.userConfig.recovery_basebackup_name-property'} (string, Pattern: `^[a-zA-Z0-9-_:.]+$`, MaxLength: 128). Name of the basebackup to restore in forked service.
- [`s3_migration`](#spec.userConfig.s3_migration-property){: name='spec.userConfig.s3_migration-property'} (object). AWS S3 / AWS S3 compatible migration settings. See below for [nested schema](#spec.userConfig.s3_migration).
- [`saml`](#spec.userConfig.saml-property){: name='spec.userConfig.saml-property'} (object). OpenSearch SAML configuration. See below for [nested schema](#spec.userConfig.saml).
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`service_to_fork_from`](#spec.userConfig.service_to_fork_from-property){: name='spec.userConfig.service_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

### azure_migration {: #spec.userConfig.azure_migration }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Azure migration settings.

**Required**

- [`account`](#spec.userConfig.azure_migration.account-property){: name='spec.userConfig.azure_migration.account-property'} (string, Pattern: `^[^\r\n]*$`). Account name.
- [`base_path`](#spec.userConfig.azure_migration.base_path-property){: name='spec.userConfig.azure_migration.base_path-property'} (string, Pattern: `^[^\r\n]*$`). The path to the repository data within its container. The value of this setting should not start or end with a /.
- [`container`](#spec.userConfig.azure_migration.container-property){: name='spec.userConfig.azure_migration.container-property'} (string, Pattern: `^[^\r\n]*$`). Azure container name.
- [`indices`](#spec.userConfig.azure_migration.indices-property){: name='spec.userConfig.azure_migration.indices-property'} (string). A comma-delimited list of indices to restore from the snapshot. Multi-index syntax is supported.
- [`snapshot_name`](#spec.userConfig.azure_migration.snapshot_name-property){: name='spec.userConfig.azure_migration.snapshot_name-property'} (string, Pattern: `^[^\r\n]*$`). The snapshot name to restore from.

**Optional**

- [`chunk_size`](#spec.userConfig.azure_migration.chunk_size-property){: name='spec.userConfig.azure_migration.chunk_size-property'} (string, Pattern: `^[^\r\n]*$`). Big files can be broken down into chunks during snapshotting if needed. Should be the same as for the 3rd party repository.
- [`compress`](#spec.userConfig.azure_migration.compress-property){: name='spec.userConfig.azure_migration.compress-property'} (boolean). when set to true metadata files are stored in compressed format.
- [`endpoint_suffix`](#spec.userConfig.azure_migration.endpoint_suffix-property){: name='spec.userConfig.azure_migration.endpoint_suffix-property'} (string, Pattern: `^[^\r\n]*$`). Defines the DNS suffix for Azure Storage endpoints.
- [`include_aliases`](#spec.userConfig.azure_migration.include_aliases-property){: name='spec.userConfig.azure_migration.include_aliases-property'} (boolean). Whether to restore aliases alongside their associated indexes. Default is true.
- [`key`](#spec.userConfig.azure_migration.key-property){: name='spec.userConfig.azure_migration.key-property'} (string, Pattern: `^[^\r\n]*$`). Azure account secret key. One of key or sas_token should be specified.
- [`readonly`](#spec.userConfig.azure_migration.readonly-property){: name='spec.userConfig.azure_migration.readonly-property'} (boolean). Whether the repository is read-only.
- [`restore_global_state`](#spec.userConfig.azure_migration.restore_global_state-property){: name='spec.userConfig.azure_migration.restore_global_state-property'} (boolean). If true, restore the cluster state. Defaults to false.
- [`sas_token`](#spec.userConfig.azure_migration.sas_token-property){: name='spec.userConfig.azure_migration.sas_token-property'} (string, Pattern: `^[^\r\n]*$`). A shared access signatures (SAS) token. One of key or sas_token should be specified.

### gcs_migration {: #spec.userConfig.gcs_migration }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Google Cloud Storage migration settings.

**Required**

- [`base_path`](#spec.userConfig.gcs_migration.base_path-property){: name='spec.userConfig.gcs_migration.base_path-property'} (string, Pattern: `^[^\r\n]*$`). The path to the repository data within its container. The value of this setting should not start or end with a /.
- [`bucket`](#spec.userConfig.gcs_migration.bucket-property){: name='spec.userConfig.gcs_migration.bucket-property'} (string, Pattern: `^[^\r\n]*$`). The path to the repository data within its container.
- [`credentials`](#spec.userConfig.gcs_migration.credentials-property){: name='spec.userConfig.gcs_migration.credentials-property'} (string, Pattern: `^[^\r\n]*$`). Google Cloud Storage credentials file content.
- [`indices`](#spec.userConfig.gcs_migration.indices-property){: name='spec.userConfig.gcs_migration.indices-property'} (string). A comma-delimited list of indices to restore from the snapshot. Multi-index syntax is supported.
- [`snapshot_name`](#spec.userConfig.gcs_migration.snapshot_name-property){: name='spec.userConfig.gcs_migration.snapshot_name-property'} (string, Pattern: `^[^\r\n]*$`). The snapshot name to restore from.

**Optional**

- [`chunk_size`](#spec.userConfig.gcs_migration.chunk_size-property){: name='spec.userConfig.gcs_migration.chunk_size-property'} (string, Pattern: `^[^\r\n]*$`). Big files can be broken down into chunks during snapshotting if needed. Should be the same as for the 3rd party repository.
- [`compress`](#spec.userConfig.gcs_migration.compress-property){: name='spec.userConfig.gcs_migration.compress-property'} (boolean). when set to true metadata files are stored in compressed format.
- [`include_aliases`](#spec.userConfig.gcs_migration.include_aliases-property){: name='spec.userConfig.gcs_migration.include_aliases-property'} (boolean). Whether to restore aliases alongside their associated indexes. Default is true.
- [`readonly`](#spec.userConfig.gcs_migration.readonly-property){: name='spec.userConfig.gcs_migration.readonly-property'} (boolean). Whether the repository is read-only.
- [`restore_global_state`](#spec.userConfig.gcs_migration.restore_global_state-property){: name='spec.userConfig.gcs_migration.restore_global_state-property'} (boolean). If true, restore the cluster state. Defaults to false.

### index_patterns {: #spec.userConfig.index_patterns }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like `logs.?` and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note `logs.?` does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.

**Required**

- [`max_index_count`](#spec.userConfig.index_patterns.max_index_count-property){: name='spec.userConfig.index_patterns.max_index_count-property'} (integer, Minimum: 0). Maximum number of indexes to keep.
- [`pattern`](#spec.userConfig.index_patterns.pattern-property){: name='spec.userConfig.index_patterns.pattern-property'} (string, Pattern: `^[A-Za-z0-9-_.*?]+$`, MaxLength: 1024). fnmatch pattern.

**Optional**

- [`sorting_algorithm`](#spec.userConfig.index_patterns.sorting_algorithm-property){: name='spec.userConfig.index_patterns.sorting_algorithm-property'} (string, Enum: `alphabetical`, `creation_date`). Deletion sorting algorithm.

### index_rollup {: #spec.userConfig.index_rollup }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Index rollup settings.

**Optional**

- [`rollup_dashboards_enabled`](#spec.userConfig.index_rollup.rollup_dashboards_enabled-property){: name='spec.userConfig.index_rollup.rollup_dashboards_enabled-property'} (boolean). Whether rollups are enabled in OpenSearch Dashboards. Defaults to true.
- [`rollup_enabled`](#spec.userConfig.index_rollup.rollup_enabled-property){: name='spec.userConfig.index_rollup.rollup_enabled-property'} (boolean). Whether the rollup plugin is enabled. Defaults to true.
- [`rollup_search_backoff_count`](#spec.userConfig.index_rollup.rollup_search_backoff_count-property){: name='spec.userConfig.index_rollup.rollup_search_backoff_count-property'} (integer, Minimum: 1). How many retries the plugin should attempt for failed rollup jobs. Defaults to 5.
- [`rollup_search_backoff_millis`](#spec.userConfig.index_rollup.rollup_search_backoff_millis-property){: name='spec.userConfig.index_rollup.rollup_search_backoff_millis-property'} (integer, Minimum: 1). The backoff time between retries for failed rollup jobs. Defaults to 1000ms.
- [`rollup_search_search_all_jobs`](#spec.userConfig.index_rollup.rollup_search_search_all_jobs-property){: name='spec.userConfig.index_rollup.rollup_search_search_all_jobs-property'} (boolean). Whether OpenSearch should return all jobs that match all specified search terms. If disabled, OpenSearch returns just one, as opposed to all, of the jobs that matches the search terms. Defaults to false.

### index_template {: #spec.userConfig.index_template }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Template settings for all new indexes.

**Optional**

- [`mapping_nested_objects_limit`](#spec.userConfig.index_template.mapping_nested_objects_limit-property){: name='spec.userConfig.index_template.mapping_nested_objects_limit-property'} (integer, Minimum: 0, Maximum: 100000). The maximum number of nested JSON objects that a single document can contain across all nested types. This limit helps to prevent out of memory errors when a document contains too many nested objects. Default is 10000. Deprecated, use an index template instead.
- [`number_of_replicas`](#spec.userConfig.index_template.number_of_replicas-property){: name='spec.userConfig.index_template.number_of_replicas-property'} (integer, Minimum: 0, Maximum: 29). The number of replicas each primary shard has. Deprecated, use an index template instead.
- [`number_of_shards`](#spec.userConfig.index_template.number_of_shards-property){: name='spec.userConfig.index_template.number_of_shards-property'} (integer, Minimum: 1, Maximum: 1024). The number of primary shards that an index should have. Deprecated, use an index template instead.

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### openid {: #spec.userConfig.openid }

_Appears on [`spec.userConfig`](#spec.userConfig)._

OpenSearch OpenID Connect Configuration.

**Required**

- [`client_id`](#spec.userConfig.openid.client_id-property){: name='spec.userConfig.openid.client_id-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). The ID of the OpenID Connect client configured in your IdP. Required.
- [`client_secret`](#spec.userConfig.openid.client_secret-property){: name='spec.userConfig.openid.client_secret-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). The client secret of the OpenID Connect client configured in your IdP. Required.
- [`connect_url`](#spec.userConfig.openid.connect_url-property){: name='spec.userConfig.openid.connect_url-property'} (string, Pattern: `^[^\r\n]*$`, MaxLength: 2048). The URL of your IdP where the Security plugin can find the OpenID Connect metadata/configuration settings.
- [`enabled`](#spec.userConfig.openid.enabled-property){: name='spec.userConfig.openid.enabled-property'} (boolean). Enables or disables OpenID Connect authentication for OpenSearch. When enabled, users can authenticate using OpenID Connect with an Identity Provider.

**Optional**

- [`header`](#spec.userConfig.openid.header-property){: name='spec.userConfig.openid.header-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). HTTP header name of the JWT token. Optional. Default is Authorization.
- [`jwt_header`](#spec.userConfig.openid.jwt_header-property){: name='spec.userConfig.openid.jwt_header-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). The HTTP header that stores the token. Typically the Authorization header with the Bearer schema: Authorization: Bearer <token>. Optional. Default is Authorization.
- [`jwt_url_parameter`](#spec.userConfig.openid.jwt_url_parameter-property){: name='spec.userConfig.openid.jwt_url_parameter-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). If the token is not transmitted in the HTTP header, but as an URL parameter, define the name of the parameter here. Optional.
- [`refresh_rate_limit_count`](#spec.userConfig.openid.refresh_rate_limit_count-property){: name='spec.userConfig.openid.refresh_rate_limit_count-property'} (integer, Minimum: 10). The maximum number of unknown key IDs in the time frame. Default is 10. Optional.
- [`refresh_rate_limit_time_window_ms`](#spec.userConfig.openid.refresh_rate_limit_time_window_ms-property){: name='spec.userConfig.openid.refresh_rate_limit_time_window_ms-property'} (integer, Minimum: 10000). The time frame to use when checking the maximum number of unknown key IDs, in milliseconds. Optional.Default is 10000 (10 seconds).
- [`roles_key`](#spec.userConfig.openid.roles_key-property){: name='spec.userConfig.openid.roles_key-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). The key in the JSON payload that stores the user’s roles. The value of this key must be a comma-separated list of roles. Required only if you want to use roles in the JWT.
- [`scope`](#spec.userConfig.openid.scope-property){: name='spec.userConfig.openid.scope-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). The scope of the identity token issued by the IdP. Optional. Default is openid profile email address phone.
- [`subject_key`](#spec.userConfig.openid.subject_key-property){: name='spec.userConfig.openid.subject_key-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). The key in the JSON payload that stores the user’s name. If not defined, the subject registered claim is used. Most IdP providers use the preferred_username claim. Optional.

### opensearch {: #spec.userConfig.opensearch }

_Appears on [`spec.userConfig`](#spec.userConfig)._

OpenSearch settings.

**Optional**

- [`action_auto_create_index_enabled`](#spec.userConfig.opensearch.action_auto_create_index_enabled-property){: name='spec.userConfig.opensearch.action_auto_create_index_enabled-property'} (boolean). Explicitly allow or block automatic creation of indices. Defaults to true.
- [`action_destructive_requires_name`](#spec.userConfig.opensearch.action_destructive_requires_name-property){: name='spec.userConfig.opensearch.action_destructive_requires_name-property'} (boolean). Require explicit index names when deleting.
- [`auth_failure_listeners`](#spec.userConfig.opensearch.auth_failure_listeners-property){: name='spec.userConfig.opensearch.auth_failure_listeners-property'} (object). Opensearch Security Plugin Settings. See below for [nested schema](#spec.userConfig.opensearch.auth_failure_listeners).
- [`cluster.filecache.remote_data_ratio`](#spec.userConfig.opensearch.cluster.filecache.remote_data_ratio-property){: name='spec.userConfig.opensearch.cluster.filecache.remote_data_ratio-property'} (number, Minimum: 0, Maximum: 100). Defines a limit of how much total remote data can be referenced as a ratio of the size of the disk reserved for the file cache. This is designed to be a safeguard to prevent oversubscribing a cluster. Defaults to 0.
- [`cluster.remote_store`](#spec.userConfig.opensearch.cluster.remote_store-property){: name='spec.userConfig.opensearch.cluster.remote_store-property'} (object). See below for [nested schema](#spec.userConfig.opensearch.cluster.remote_store).
- [`cluster.routing.allocation.balance.prefer_primary`](#spec.userConfig.opensearch.cluster.routing.allocation.balance.prefer_primary-property){: name='spec.userConfig.opensearch.cluster.routing.allocation.balance.prefer_primary-property'} (boolean). When set to true, OpenSearch attempts to evenly distribute the primary shards between the cluster nodes. Enabling this setting does not always guarantee an equal number of primary shards on each node, especially in the event of a failover. Changing this setting to false after it was set to true does not invoke redistribution of primary shards. Default is false.
- [`cluster.search.request.slowlog`](#spec.userConfig.opensearch.cluster.search.request.slowlog-property){: name='spec.userConfig.opensearch.cluster.search.request.slowlog-property'} (object). See below for [nested schema](#spec.userConfig.opensearch.cluster.search.request.slowlog).
- [`cluster_max_shards_per_node`](#spec.userConfig.opensearch.cluster_max_shards_per_node-property){: name='spec.userConfig.opensearch.cluster_max_shards_per_node-property'} (integer, Minimum: 100, Maximum: 10000). Controls the number of shards allowed in the cluster per data node.
- [`cluster_routing_allocation_node_concurrent_recoveries`](#spec.userConfig.opensearch.cluster_routing_allocation_node_concurrent_recoveries-property){: name='spec.userConfig.opensearch.cluster_routing_allocation_node_concurrent_recoveries-property'} (integer, Minimum: 2, Maximum: 16). How many concurrent incoming/outgoing shard recoveries (normally replicas) are allowed to happen on a node. Defaults to node cpu count * 2.
- [`disk_watermarks`](#spec.userConfig.opensearch.disk_watermarks-property){: name='spec.userConfig.opensearch.disk_watermarks-property'} (object). Watermark settings. See below for [nested schema](#spec.userConfig.opensearch.disk_watermarks).
- [`email_sender_name`](#spec.userConfig.opensearch.email_sender_name-property){: name='spec.userConfig.opensearch.email_sender_name-property'} (string, Pattern: `^[a-zA-Z0-9-_]+$`, MaxLength: 40). Sender name placeholder to be used in Opensearch Dashboards and Opensearch keystore.
- [`email_sender_password`](#spec.userConfig.opensearch.email_sender_password-property){: name='spec.userConfig.opensearch.email_sender_password-property'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 1024). Sender password for Opensearch alerts to authenticate with SMTP server.
- [`email_sender_username`](#spec.userConfig.opensearch.email_sender_username-property){: name='spec.userConfig.opensearch.email_sender_username-property'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 320). Sender username for Opensearch alerts.
- [`enable_remote_backed_storage`](#spec.userConfig.opensearch.enable_remote_backed_storage-property){: name='spec.userConfig.opensearch.enable_remote_backed_storage-property'} (boolean). Enable remote-backed storage.
- [`enable_searchable_snapshots`](#spec.userConfig.opensearch.enable_searchable_snapshots-property){: name='spec.userConfig.opensearch.enable_searchable_snapshots-property'} (boolean). Enable searchable snapshots.
- [`enable_security_audit`](#spec.userConfig.opensearch.enable_security_audit-property){: name='spec.userConfig.opensearch.enable_security_audit-property'} (boolean). Enable/Disable security audit.
- [`enable_snapshot_api`](#spec.userConfig.opensearch.enable_snapshot_api-property){: name='spec.userConfig.opensearch.enable_snapshot_api-property'} (boolean). Enable/Disable snapshot API for custom repositories, this requires security management to be enabled.
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
- [`knn_memory_circuit_breaker_enabled`](#spec.userConfig.opensearch.knn_memory_circuit_breaker_enabled-property){: name='spec.userConfig.opensearch.knn_memory_circuit_breaker_enabled-property'} (boolean). Enable or disable KNN memory circuit breaker. Defaults to true.
- [`knn_memory_circuit_breaker_limit`](#spec.userConfig.opensearch.knn_memory_circuit_breaker_limit-property){: name='spec.userConfig.opensearch.knn_memory_circuit_breaker_limit-property'} (integer, Minimum: 3, Maximum: 100). Maximum amount of memory that can be used for KNN index. Defaults to 50% of the JVM heap size.
- [`node.search.cache.size`](#spec.userConfig.opensearch.node.search.cache.size-property){: name='spec.userConfig.opensearch.node.search.cache.size-property'} (string, Pattern: `\d+(?:b|kb|mb|gb|tb)`). Defines a limit of how much total remote data can be referenced as a ratio of the size of the disk reserved for the file cache. This is designed to be a safeguard to prevent oversubscribing a cluster. Defaults to 5gb. Requires restarting all OpenSearch nodes.
- [`override_main_response_version`](#spec.userConfig.opensearch.override_main_response_version-property){: name='spec.userConfig.opensearch.override_main_response_version-property'} (boolean). Compatibility mode sets OpenSearch to report its version as 7.10 so clients continue to work. Default is false.
- [`plugins_alerting_filter_by_backend_roles`](#spec.userConfig.opensearch.plugins_alerting_filter_by_backend_roles-property){: name='spec.userConfig.opensearch.plugins_alerting_filter_by_backend_roles-property'} (boolean). Enable or disable filtering of alerting by backend roles. Requires Security plugin. Defaults to false.
- [`reindex_remote_whitelist`](#spec.userConfig.opensearch.reindex_remote_whitelist-property){: name='spec.userConfig.opensearch.reindex_remote_whitelist-property'} (array of strings, MaxItems: 32). Whitelisted addresses for reindexing. Changing this value will cause all OpenSearch instances to restart.
- [`remote_store`](#spec.userConfig.opensearch.remote_store-property){: name='spec.userConfig.opensearch.remote_store-property'} (object). See below for [nested schema](#spec.userConfig.opensearch.remote_store).
- [`script_max_compilations_rate`](#spec.userConfig.opensearch.script_max_compilations_rate-property){: name='spec.userConfig.opensearch.script_max_compilations_rate-property'} (string, Pattern: `^[^\r\n]*$`, MaxLength: 1024). Script compilation circuit breaker limits the number of inline script compilations within a period of time. Default is use-context.
- [`search.insights.top_queries`](#spec.userConfig.opensearch.search.insights.top_queries-property){: name='spec.userConfig.opensearch.search.insights.top_queries-property'} (object). See below for [nested schema](#spec.userConfig.opensearch.search.insights.top_queries).
- [`search_backpressure`](#spec.userConfig.opensearch.search_backpressure-property){: name='spec.userConfig.opensearch.search_backpressure-property'} (object). Search Backpressure Settings. See below for [nested schema](#spec.userConfig.opensearch.search_backpressure).
- [`search_max_buckets`](#spec.userConfig.opensearch.search_max_buckets-property){: name='spec.userConfig.opensearch.search_max_buckets-property'} (integer, Minimum: 1, Maximum: 1000000). Maximum number of aggregation buckets allowed in a single response. OpenSearch default value is used when this is not defined.
- [`segrep`](#spec.userConfig.opensearch.segrep-property){: name='spec.userConfig.opensearch.segrep-property'} (object). Segment Replication Backpressure Settings. See below for [nested schema](#spec.userConfig.opensearch.segrep).
- [`shard_indexing_pressure`](#spec.userConfig.opensearch.shard_indexing_pressure-property){: name='spec.userConfig.opensearch.shard_indexing_pressure-property'} (object). Shard indexing back pressure settings. See below for [nested schema](#spec.userConfig.opensearch.shard_indexing_pressure).
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
- [`ip_rate_limiting`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting-property'} (object). Deprecated. IP address rate limiting settings. See below for [nested schema](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting).

##### internal_authentication_backend_limiting {: #spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting }

_Appears on [`spec.userConfig.opensearch.auth_failure_listeners`](#spec.userConfig.opensearch.auth_failure_listeners)._

**Optional**

- [`allowed_tries`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.allowed_tries-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.allowed_tries-property'} (integer, Minimum: 1, Maximum: 32767). The number of login attempts allowed before login is blocked.
- [`authentication_backend`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.authentication_backend-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.authentication_backend-property'} (string, Enum: `internal`, MaxLength: 1024). internal_authentication_backend_limiting.authentication_backend.
- [`block_expiry_seconds`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.block_expiry_seconds-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.block_expiry_seconds-property'} (integer, Minimum: 0, Maximum: 2147483647). The duration of time that login remains blocked after a failed login.
- [`max_blocked_clients`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.max_blocked_clients-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.max_blocked_clients-property'} (integer, Minimum: 0, Maximum: 2147483647). internal_authentication_backend_limiting.max_blocked_clients.
- [`max_tracked_clients`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.max_tracked_clients-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.max_tracked_clients-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum number of tracked IP addresses that have failed login.
- [`time_window_seconds`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.time_window_seconds-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.time_window_seconds-property'} (integer, Minimum: 0, Maximum: 2147483647). The window of time in which the value for `allowed_tries` is enforced.
- [`type`](#spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.type-property){: name='spec.userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.type-property'} (string, Enum: `username`, MaxLength: 1024). internal_authentication_backend_limiting.type.

##### ip_rate_limiting {: #spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting }

_Appears on [`spec.userConfig.opensearch.auth_failure_listeners`](#spec.userConfig.opensearch.auth_failure_listeners)._

Deprecated. IP address rate limiting settings.

**Optional**

- [`allowed_tries`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.allowed_tries-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.allowed_tries-property'} (integer, Minimum: 1, Maximum: 2147483647). The number of login attempts allowed before login is blocked.
- [`block_expiry_seconds`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.block_expiry_seconds-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.block_expiry_seconds-property'} (integer, Minimum: 0, Maximum: 36000). The duration of time that login remains blocked after a failed login.
- [`max_blocked_clients`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.max_blocked_clients-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.max_blocked_clients-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum number of blocked IP addresses.
- [`max_tracked_clients`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.max_tracked_clients-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.max_tracked_clients-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum number of tracked IP addresses that have failed login.
- [`time_window_seconds`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.time_window_seconds-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.time_window_seconds-property'} (integer, Minimum: 0, Maximum: 36000). The window of time in which the value for `allowed_tries` is enforced.
- [`type`](#spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.type-property){: name='spec.userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.type-property'} (string, Enum: `ip`, MaxLength: 1024). The type of rate limiting.

#### cluster.remote_store {: #spec.userConfig.opensearch.cluster.remote_store }

_Appears on [`spec.userConfig.opensearch`](#spec.userConfig.opensearch)._

**Optional**

- [`state.global_metadata.upload_timeout`](#spec.userConfig.opensearch.cluster.remote_store.state.global_metadata.upload_timeout-property){: name='spec.userConfig.opensearch.cluster.remote_store.state.global_metadata.upload_timeout-property'} (string, Pattern: `\d+(?:d|h|m|s|ms|micros|nanos)`). The amount of time to wait for the cluster state upload to complete. Defaults to 20s.
- [`state.metadata_manifest.upload_timeout`](#spec.userConfig.opensearch.cluster.remote_store.state.metadata_manifest.upload_timeout-property){: name='spec.userConfig.opensearch.cluster.remote_store.state.metadata_manifest.upload_timeout-property'} (string, Pattern: `\d+(?:d|h|m|s|ms|micros|nanos)`). The amount of time to wait for the manifest file upload to complete. The manifest file contains the details of each of the files uploaded for a single cluster state, both index metadata files and global metadata files. Defaults to 20s.
- [`translog.buffer_interval`](#spec.userConfig.opensearch.cluster.remote_store.translog.buffer_interval-property){: name='spec.userConfig.opensearch.cluster.remote_store.translog.buffer_interval-property'} (string, Pattern: `\d+(?:d|h|m|s|ms|micros|nanos)`). The default value of the translog buffer interval used when performing periodic translog updates. This setting is only effective when the index setting `index.remote_store.translog.buffer_interval` is not present. Defaults to 650ms.
- [`translog.max_readers`](#spec.userConfig.opensearch.cluster.remote_store.translog.max_readers-property){: name='spec.userConfig.opensearch.cluster.remote_store.translog.max_readers-property'} (integer, Minimum: 100, Maximum: 2147483647). Sets the maximum number of open translog files for remote-backed indexes. This limits the total number of translog files per shard. After reaching this limit, the remote store flushes the translog files. Default is 1000. The minimum required is 100.

#### cluster.search.request.slowlog {: #spec.userConfig.opensearch.cluster.search.request.slowlog }

_Appears on [`spec.userConfig.opensearch`](#spec.userConfig.opensearch)._

**Optional**

- [`level`](#spec.userConfig.opensearch.cluster.search.request.slowlog.level-property){: name='spec.userConfig.opensearch.cluster.search.request.slowlog.level-property'} (string, Enum: `debug`, `info`, `trace`, `warn`). Log level.
- [`threshold`](#spec.userConfig.opensearch.cluster.search.request.slowlog.threshold-property){: name='spec.userConfig.opensearch.cluster.search.request.slowlog.threshold-property'} (object). See below for [nested schema](#spec.userConfig.opensearch.cluster.search.request.slowlog.threshold).

##### threshold {: #spec.userConfig.opensearch.cluster.search.request.slowlog.threshold }

_Appears on [`spec.userConfig.opensearch.cluster.search.request.slowlog`](#spec.userConfig.opensearch.cluster.search.request.slowlog)._

**Optional**

- [`debug`](#spec.userConfig.opensearch.cluster.search.request.slowlog.threshold.debug-property){: name='spec.userConfig.opensearch.cluster.search.request.slowlog.threshold.debug-property'} (string, Pattern: `^[^\r\n]*$`). Debug threshold for total request took time. The value should be in the form count and unit, where unit one of (s,m,h,d,nanos,ms,micros) or -1. Default is -1.
- [`info`](#spec.userConfig.opensearch.cluster.search.request.slowlog.threshold.info-property){: name='spec.userConfig.opensearch.cluster.search.request.slowlog.threshold.info-property'} (string, Pattern: `^[^\r\n]*$`). Info threshold for total request took time. The value should be in the form count and unit, where unit one of (s,m,h,d,nanos,ms,micros) or -1. Default is -1.
- [`trace`](#spec.userConfig.opensearch.cluster.search.request.slowlog.threshold.trace-property){: name='spec.userConfig.opensearch.cluster.search.request.slowlog.threshold.trace-property'} (string, Pattern: `^[^\r\n]*$`). Trace threshold for total request took time. The value should be in the form count and unit, where unit one of (s,m,h,d,nanos,ms,micros) or -1. Default is -1.
- [`warn`](#spec.userConfig.opensearch.cluster.search.request.slowlog.threshold.warn-property){: name='spec.userConfig.opensearch.cluster.search.request.slowlog.threshold.warn-property'} (string, Pattern: `^[^\r\n]*$`). Warning threshold for total request took time. The value should be in the form count and unit, where unit one of (s,m,h,d,nanos,ms,micros) or -1. Default is -1.

#### disk_watermarks {: #spec.userConfig.opensearch.disk_watermarks }

_Appears on [`spec.userConfig.opensearch`](#spec.userConfig.opensearch)._

Watermark settings.

**Required**

- [`flood_stage`](#spec.userConfig.opensearch.disk_watermarks.flood_stage-property){: name='spec.userConfig.opensearch.disk_watermarks.flood_stage-property'} (integer). The flood stage watermark for disk usage.
- [`high`](#spec.userConfig.opensearch.disk_watermarks.high-property){: name='spec.userConfig.opensearch.disk_watermarks.high-property'} (integer). The high watermark for disk usage.
- [`low`](#spec.userConfig.opensearch.disk_watermarks.low-property){: name='spec.userConfig.opensearch.disk_watermarks.low-property'} (integer). The low watermark for disk usage.

#### remote_store {: #spec.userConfig.opensearch.remote_store }

_Appears on [`spec.userConfig.opensearch`](#spec.userConfig.opensearch)._

**Optional**

- [`segment.pressure.bytes_lag.variance_factor`](#spec.userConfig.opensearch.remote_store.segment.pressure.bytes_lag.variance_factor-property){: name='spec.userConfig.opensearch.remote_store.segment.pressure.bytes_lag.variance_factor-property'} (number, Minimum: 1). The variance factor that is used together with the moving average to calculate the dynamic bytes lag threshold for activating remote segment backpressure. Defaults to 10.
- [`segment.pressure.consecutive_failures.limit`](#spec.userConfig.opensearch.remote_store.segment.pressure.consecutive_failures.limit-property){: name='spec.userConfig.opensearch.remote_store.segment.pressure.consecutive_failures.limit-property'} (integer, Minimum: 1, Maximum: 2147483647). The minimum consecutive failure count for activating remote segment backpressure. Defaults to 5.
- [`segment.pressure.enabled`](#spec.userConfig.opensearch.remote_store.segment.pressure.enabled-property){: name='spec.userConfig.opensearch.remote_store.segment.pressure.enabled-property'} (boolean). Enables remote segment backpressure. Default is `true`.
- [`segment.pressure.time_lag.variance_factor`](#spec.userConfig.opensearch.remote_store.segment.pressure.time_lag.variance_factor-property){: name='spec.userConfig.opensearch.remote_store.segment.pressure.time_lag.variance_factor-property'} (number, Minimum: 1). The variance factor that is used together with the moving average to calculate the dynamic time lag threshold for activating remote segment backpressure. Defaults to 10.

#### search.insights.top_queries {: #spec.userConfig.opensearch.search.insights.top_queries }

_Appears on [`spec.userConfig.opensearch`](#spec.userConfig.opensearch)._

**Optional**

- [`cpu`](#spec.userConfig.opensearch.search.insights.top_queries.cpu-property){: name='spec.userConfig.opensearch.search.insights.top_queries.cpu-property'} (object). Top N queries monitoring by CPU. See below for [nested schema](#spec.userConfig.opensearch.search.insights.top_queries.cpu).
- [`latency`](#spec.userConfig.opensearch.search.insights.top_queries.latency-property){: name='spec.userConfig.opensearch.search.insights.top_queries.latency-property'} (object). Top N queries monitoring by latency. See below for [nested schema](#spec.userConfig.opensearch.search.insights.top_queries.latency).
- [`memory`](#spec.userConfig.opensearch.search.insights.top_queries.memory-property){: name='spec.userConfig.opensearch.search.insights.top_queries.memory-property'} (object). Top N queries monitoring by memory. See below for [nested schema](#spec.userConfig.opensearch.search.insights.top_queries.memory).

##### cpu {: #spec.userConfig.opensearch.search.insights.top_queries.cpu }

_Appears on [`spec.userConfig.opensearch.search.insights.top_queries`](#spec.userConfig.opensearch.search.insights.top_queries)._

Top N queries monitoring by CPU.

**Optional**

- [`enabled`](#spec.userConfig.opensearch.search.insights.top_queries.cpu.enabled-property){: name='spec.userConfig.opensearch.search.insights.top_queries.cpu.enabled-property'} (boolean). Enable or disable top N query monitoring by the metric.
- [`top_n_size`](#spec.userConfig.opensearch.search.insights.top_queries.cpu.top_n_size-property){: name='spec.userConfig.opensearch.search.insights.top_queries.cpu.top_n_size-property'} (integer, Minimum: 1). Specify the value of N for the top N queries by the metric.
- [`window_size`](#spec.userConfig.opensearch.search.insights.top_queries.cpu.window_size-property){: name='spec.userConfig.opensearch.search.insights.top_queries.cpu.window_size-property'} (string). The window size of the top N queries by the metric.

##### latency {: #spec.userConfig.opensearch.search.insights.top_queries.latency }

_Appears on [`spec.userConfig.opensearch.search.insights.top_queries`](#spec.userConfig.opensearch.search.insights.top_queries)._

Top N queries monitoring by latency.

**Optional**

- [`enabled`](#spec.userConfig.opensearch.search.insights.top_queries.latency.enabled-property){: name='spec.userConfig.opensearch.search.insights.top_queries.latency.enabled-property'} (boolean). Enable or disable top N query monitoring by the metric.
- [`top_n_size`](#spec.userConfig.opensearch.search.insights.top_queries.latency.top_n_size-property){: name='spec.userConfig.opensearch.search.insights.top_queries.latency.top_n_size-property'} (integer, Minimum: 1). Specify the value of N for the top N queries by the metric.
- [`window_size`](#spec.userConfig.opensearch.search.insights.top_queries.latency.window_size-property){: name='spec.userConfig.opensearch.search.insights.top_queries.latency.window_size-property'} (string). The window size of the top N queries by the metric.

##### memory {: #spec.userConfig.opensearch.search.insights.top_queries.memory }

_Appears on [`spec.userConfig.opensearch.search.insights.top_queries`](#spec.userConfig.opensearch.search.insights.top_queries)._

Top N queries monitoring by memory.

**Optional**

- [`enabled`](#spec.userConfig.opensearch.search.insights.top_queries.memory.enabled-property){: name='spec.userConfig.opensearch.search.insights.top_queries.memory.enabled-property'} (boolean). Enable or disable top N query monitoring by the metric.
- [`top_n_size`](#spec.userConfig.opensearch.search.insights.top_queries.memory.top_n_size-property){: name='spec.userConfig.opensearch.search.insights.top_queries.memory.top_n_size-property'} (integer, Minimum: 1). Specify the value of N for the top N queries by the metric.
- [`window_size`](#spec.userConfig.opensearch.search.insights.top_queries.memory.window_size-property){: name='spec.userConfig.opensearch.search.insights.top_queries.memory.window_size-property'} (string). The window size of the top N queries by the metric.

#### search_backpressure {: #spec.userConfig.opensearch.search_backpressure }

_Appears on [`spec.userConfig.opensearch`](#spec.userConfig.opensearch)._

Search Backpressure Settings.

**Optional**

- [`mode`](#spec.userConfig.opensearch.search_backpressure.mode-property){: name='spec.userConfig.opensearch.search_backpressure.mode-property'} (string, Enum: `disabled`, `enforced`, `monitor_only`). The search backpressure mode. Valid values are monitor_only, enforced, or disabled. Default is monitor_only.
- [`node_duress`](#spec.userConfig.opensearch.search_backpressure.node_duress-property){: name='spec.userConfig.opensearch.search_backpressure.node_duress-property'} (object). Node duress settings. See below for [nested schema](#spec.userConfig.opensearch.search_backpressure.node_duress).
- [`search_shard_task`](#spec.userConfig.opensearch.search_backpressure.search_shard_task-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task-property'} (object). Search shard settings. See below for [nested schema](#spec.userConfig.opensearch.search_backpressure.search_shard_task).
- [`search_task`](#spec.userConfig.opensearch.search_backpressure.search_task-property){: name='spec.userConfig.opensearch.search_backpressure.search_task-property'} (object). Search task settings. See below for [nested schema](#spec.userConfig.opensearch.search_backpressure.search_task).

##### node_duress {: #spec.userConfig.opensearch.search_backpressure.node_duress }

_Appears on [`spec.userConfig.opensearch.search_backpressure`](#spec.userConfig.opensearch.search_backpressure)._

Node duress settings.

**Optional**

- [`cpu_threshold`](#spec.userConfig.opensearch.search_backpressure.node_duress.cpu_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.node_duress.cpu_threshold-property'} (number, Minimum: 0, Maximum: 1). The CPU usage threshold (as a percentage) required for a node to be considered to be under duress. Default is 0.9.
- [`heap_threshold`](#spec.userConfig.opensearch.search_backpressure.node_duress.heap_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.node_duress.heap_threshold-property'} (number, Minimum: 0, Maximum: 1). The heap usage threshold (as a percentage) required for a node to be considered to be under duress. Default is 0.7.
- [`num_successive_breaches`](#spec.userConfig.opensearch.search_backpressure.node_duress.num_successive_breaches-property){: name='spec.userConfig.opensearch.search_backpressure.node_duress.num_successive_breaches-property'} (integer, Minimum: 1). The number of successive limit breaches after which the node is considered to be under duress. Default is 3.

##### search_shard_task {: #spec.userConfig.opensearch.search_backpressure.search_shard_task }

_Appears on [`spec.userConfig.opensearch.search_backpressure`](#spec.userConfig.opensearch.search_backpressure)._

Search shard settings.

**Optional**

- [`cancellation_burst`](#spec.userConfig.opensearch.search_backpressure.search_shard_task.cancellation_burst-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task.cancellation_burst-property'} (number, Minimum: 1). The maximum number of search tasks to cancel in a single iteration of the observer thread. Default is 10.0.
- [`cancellation_rate`](#spec.userConfig.opensearch.search_backpressure.search_shard_task.cancellation_rate-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task.cancellation_rate-property'} (number, Minimum: 0). The maximum number of tasks to cancel per millisecond of elapsed time. Default is 0.003.
- [`cancellation_ratio`](#spec.userConfig.opensearch.search_backpressure.search_shard_task.cancellation_ratio-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task.cancellation_ratio-property'} (number, Minimum: 0, Maximum: 1). The maximum number of tasks to cancel, as a percentage of successful task completions. Default is 0.1.
- [`cpu_time_millis_threshold`](#spec.userConfig.opensearch.search_backpressure.search_shard_task.cpu_time_millis_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task.cpu_time_millis_threshold-property'} (integer, Minimum: 0). The CPU usage threshold (in milliseconds) required for a single search shard task before it is considered for cancellation. Default is 15000.
- [`elapsed_time_millis_threshold`](#spec.userConfig.opensearch.search_backpressure.search_shard_task.elapsed_time_millis_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task.elapsed_time_millis_threshold-property'} (integer, Minimum: 0). The elapsed time threshold (in milliseconds) required for a single search shard task before it is considered for cancellation. Default is 30000.
- [`heap_moving_average_window_size`](#spec.userConfig.opensearch.search_backpressure.search_shard_task.heap_moving_average_window_size-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task.heap_moving_average_window_size-property'} (integer, Minimum: 0). The number of previously completed search shard tasks to consider when calculating the rolling average of heap usage. Default is 100.
- [`heap_percent_threshold`](#spec.userConfig.opensearch.search_backpressure.search_shard_task.heap_percent_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task.heap_percent_threshold-property'} (number, Minimum: 0, Maximum: 1). The heap usage threshold (as a percentage) required for a single search shard task before it is considered for cancellation. Default is 0.5.
- [`heap_variance`](#spec.userConfig.opensearch.search_backpressure.search_shard_task.heap_variance-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task.heap_variance-property'} (number, Minimum: 0). The minimum variance required for a single search shard task’s heap usage compared to the rolling average of previously completed tasks before it is considered for cancellation. Default is 2.0.
- [`total_heap_percent_threshold`](#spec.userConfig.opensearch.search_backpressure.search_shard_task.total_heap_percent_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.search_shard_task.total_heap_percent_threshold-property'} (number, Minimum: 0, Maximum: 1). The heap usage threshold (as a percentage) required for the sum of heap usages of all search shard tasks before cancellation is applied. Default is 0.5.

##### search_task {: #spec.userConfig.opensearch.search_backpressure.search_task }

_Appears on [`spec.userConfig.opensearch.search_backpressure`](#spec.userConfig.opensearch.search_backpressure)._

Search task settings.

**Optional**

- [`cancellation_burst`](#spec.userConfig.opensearch.search_backpressure.search_task.cancellation_burst-property){: name='spec.userConfig.opensearch.search_backpressure.search_task.cancellation_burst-property'} (number, Minimum: 1). The maximum number of search tasks to cancel in a single iteration of the observer thread. Default is 5.0.
- [`cancellation_rate`](#spec.userConfig.opensearch.search_backpressure.search_task.cancellation_rate-property){: name='spec.userConfig.opensearch.search_backpressure.search_task.cancellation_rate-property'} (number, Minimum: 0). The maximum number of search tasks to cancel per millisecond of elapsed time. Default is 0.003.
- [`cancellation_ratio`](#spec.userConfig.opensearch.search_backpressure.search_task.cancellation_ratio-property){: name='spec.userConfig.opensearch.search_backpressure.search_task.cancellation_ratio-property'} (number, Minimum: 0, Maximum: 1). The maximum number of search tasks to cancel, as a percentage of successful search task completions. Default is 0.1.
- [`cpu_time_millis_threshold`](#spec.userConfig.opensearch.search_backpressure.search_task.cpu_time_millis_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.search_task.cpu_time_millis_threshold-property'} (integer, Minimum: 0). The CPU usage threshold (in milliseconds) required for an individual parent task before it is considered for cancellation. Default is 30000.
- [`elapsed_time_millis_threshold`](#spec.userConfig.opensearch.search_backpressure.search_task.elapsed_time_millis_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.search_task.elapsed_time_millis_threshold-property'} (integer, Minimum: 0). The elapsed time threshold (in milliseconds) required for an individual parent task before it is considered for cancellation. Default is 45000.
- [`heap_moving_average_window_size`](#spec.userConfig.opensearch.search_backpressure.search_task.heap_moving_average_window_size-property){: name='spec.userConfig.opensearch.search_backpressure.search_task.heap_moving_average_window_size-property'} (integer, Minimum: 0). The window size used to calculate the rolling average of the heap usage for the completed parent tasks. Default is 10.
- [`heap_percent_threshold`](#spec.userConfig.opensearch.search_backpressure.search_task.heap_percent_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.search_task.heap_percent_threshold-property'} (number, Minimum: 0, Maximum: 1). The heap usage threshold (as a percentage) required for an individual parent task before it is considered for cancellation. Default is 0.2.
- [`heap_variance`](#spec.userConfig.opensearch.search_backpressure.search_task.heap_variance-property){: name='spec.userConfig.opensearch.search_backpressure.search_task.heap_variance-property'} (number, Minimum: 0). The heap usage variance required for an individual parent task before it is considered for cancellation. A task is considered for cancellation when taskHeapUsage is greater than or equal to heapUsageMovingAverage * variance. Default is 2.0.
- [`total_heap_percent_threshold`](#spec.userConfig.opensearch.search_backpressure.search_task.total_heap_percent_threshold-property){: name='spec.userConfig.opensearch.search_backpressure.search_task.total_heap_percent_threshold-property'} (number, Minimum: 0, Maximum: 1). The heap usage threshold (as a percentage) required for the sum of heap usages of all search tasks before cancellation is applied. Default is 0.5.

#### segrep {: #spec.userConfig.opensearch.segrep }

_Appears on [`spec.userConfig.opensearch`](#spec.userConfig.opensearch)._

Segment Replication Backpressure Settings.

**Optional**

- [`pressure.checkpoint.limit`](#spec.userConfig.opensearch.segrep.pressure.checkpoint.limit-property){: name='spec.userConfig.opensearch.segrep.pressure.checkpoint.limit-property'} (integer, Minimum: 0). The maximum number of indexing checkpoints that a replica shard can fall behind when copying from primary. Once `segrep.pressure.checkpoint.limit` is breached along with `segrep.pressure.time.limit`, the segment replication backpressure mechanism is initiated. Default is 4 checkpoints.
- [`pressure.enabled`](#spec.userConfig.opensearch.segrep.pressure.enabled-property){: name='spec.userConfig.opensearch.segrep.pressure.enabled-property'} (boolean). Enables the segment replication backpressure mechanism. Default is false.
- [`pressure.replica.stale.limit`](#spec.userConfig.opensearch.segrep.pressure.replica.stale.limit-property){: name='spec.userConfig.opensearch.segrep.pressure.replica.stale.limit-property'} (number, Minimum: 0, Maximum: 1). The maximum number of stale replica shards that can exist in a replication group. Once `segrep.pressure.replica.stale.limit` is breached, the segment replication backpressure mechanism is initiated. Default is .5, which is 50% of a replication group.
- [`pressure.time.limit`](#spec.userConfig.opensearch.segrep.pressure.time.limit-property){: name='spec.userConfig.opensearch.segrep.pressure.time.limit-property'} (string, Pattern: `^\d+\s*(?:[dhms]|ms|micros|nanos)$`). The maximum amount of time that a replica shard can take to copy from the primary shard. Once segrep.pressure.time.limit is breached along with segrep.pressure.checkpoint.limit, the segment replication backpressure mechanism is initiated. Default is 5 minutes.

#### shard_indexing_pressure {: #spec.userConfig.opensearch.shard_indexing_pressure }

_Appears on [`spec.userConfig.opensearch`](#spec.userConfig.opensearch)._

Shard indexing back pressure settings.

**Optional**

- [`enabled`](#spec.userConfig.opensearch.shard_indexing_pressure.enabled-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.enabled-property'} (boolean). Enable or disable shard indexing backpressure. Default is false.
- [`enforced`](#spec.userConfig.opensearch.shard_indexing_pressure.enforced-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.enforced-property'} (boolean). Run shard indexing backpressure in shadow mode or enforced mode. In shadow mode (value set as false), shard indexing backpressure tracks all granular-level metrics, but it doesn’t actually reject any indexing requests. In enforced mode (value set as true), shard indexing backpressure rejects any requests to the cluster that might cause a dip in its performance. Default is false.
- [`operating_factor`](#spec.userConfig.opensearch.shard_indexing_pressure.operating_factor-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.operating_factor-property'} (object). Operating factor. See below for [nested schema](#spec.userConfig.opensearch.shard_indexing_pressure.operating_factor).
- [`primary_parameter`](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter-property'} (object). Primary parameter. See below for [nested schema](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter).

##### operating_factor {: #spec.userConfig.opensearch.shard_indexing_pressure.operating_factor }

_Appears on [`spec.userConfig.opensearch.shard_indexing_pressure`](#spec.userConfig.opensearch.shard_indexing_pressure)._

Operating factor.

**Optional**

- [`lower`](#spec.userConfig.opensearch.shard_indexing_pressure.operating_factor.lower-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.operating_factor.lower-property'} (number, Minimum: 0). Specify the lower occupancy limit of the allocated quota of memory for the shard. If the total memory usage of a shard is below this limit, shard indexing backpressure decreases the current allocated memory for that shard. Default is 0.75.
- [`optimal`](#spec.userConfig.opensearch.shard_indexing_pressure.operating_factor.optimal-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.operating_factor.optimal-property'} (number, Minimum: 0). Specify the optimal occupancy of the allocated quota of memory for the shard. If the total memory usage of a shard is at this level, shard indexing backpressure doesn’t change the current allocated memory for that shard. Default is 0.85.
- [`upper`](#spec.userConfig.opensearch.shard_indexing_pressure.operating_factor.upper-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.operating_factor.upper-property'} (number, Minimum: 0). Specify the upper occupancy limit of the allocated quota of memory for the shard. If the total memory usage of a shard is above this limit, shard indexing backpressure increases the current allocated memory for that shard. Default is 0.95.

##### primary_parameter {: #spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter }

_Appears on [`spec.userConfig.opensearch.shard_indexing_pressure`](#spec.userConfig.opensearch.shard_indexing_pressure)._

Primary parameter.

**Optional**

- [`node`](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.node-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.node-property'} (object). See below for [nested schema](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.node).
- [`shard`](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.shard-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.shard-property'} (object). See below for [nested schema](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.shard).

###### node {: #spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.node }

_Appears on [`spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter`](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter)._

**Required**

- [`soft_limit`](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.node.soft_limit-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.node.soft_limit-property'} (number, Minimum: 0). Define the percentage of the node-level memory threshold that acts as a soft indicator for strain on a node. Default is 0.7.

###### shard {: #spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.shard }

_Appears on [`spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter`](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter)._

**Required**

- [`min_limit`](#spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.shard.min_limit-property){: name='spec.userConfig.opensearch.shard_indexing_pressure.primary_parameter.shard.min_limit-property'} (number, Minimum: 0). Specify the minimum assigned quota for a new shard in any role (coordinator, primary, or replica). Shard indexing backpressure increases or decreases this allocated quota based on the inflow of traffic for the shard. Default is 0.001.

### opensearch_dashboards {: #spec.userConfig.opensearch_dashboards }

_Appears on [`spec.userConfig`](#spec.userConfig)._

OpenSearch Dashboards settings.

**Optional**

- [`enabled`](#spec.userConfig.opensearch_dashboards.enabled-property){: name='spec.userConfig.opensearch_dashboards.enabled-property'} (boolean). Enable or disable OpenSearch Dashboards.
- [`max_old_space_size`](#spec.userConfig.opensearch_dashboards.max_old_space_size-property){: name='spec.userConfig.opensearch_dashboards.max_old_space_size-property'} (integer, Minimum: 64, Maximum: 4096). Limits the maximum amount of memory (in MiB) the OpenSearch Dashboards process can use. This sets the max_old_space_size option of the nodejs running the OpenSearch Dashboards. Note: the memory reserved by OpenSearch Dashboards is not available for OpenSearch.
- [`multiple_data_source_enabled`](#spec.userConfig.opensearch_dashboards.multiple_data_source_enabled-property){: name='spec.userConfig.opensearch_dashboards.multiple_data_source_enabled-property'} (boolean). Enable or disable multiple data sources in OpenSearch Dashboards.
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

### s3_migration {: #spec.userConfig.s3_migration }

_Appears on [`spec.userConfig`](#spec.userConfig)._

AWS S3 / AWS S3 compatible migration settings.

**Required**

- [`access_key`](#spec.userConfig.s3_migration.access_key-property){: name='spec.userConfig.s3_migration.access_key-property'} (string, Pattern: `^[^\r\n]*$`). AWS Access key.
- [`base_path`](#spec.userConfig.s3_migration.base_path-property){: name='spec.userConfig.s3_migration.base_path-property'} (string, Pattern: `^[^\r\n]*$`). The path to the repository data within its container. The value of this setting should not start or end with a /.
- [`bucket`](#spec.userConfig.s3_migration.bucket-property){: name='spec.userConfig.s3_migration.bucket-property'} (string, Pattern: `^[^\r\n]*$`). S3 bucket name.
- [`indices`](#spec.userConfig.s3_migration.indices-property){: name='spec.userConfig.s3_migration.indices-property'} (string). A comma-delimited list of indices to restore from the snapshot. Multi-index syntax is supported.
- [`region`](#spec.userConfig.s3_migration.region-property){: name='spec.userConfig.s3_migration.region-property'} (string, Pattern: `^[^\r\n]*$`). S3 region.
- [`secret_key`](#spec.userConfig.s3_migration.secret_key-property){: name='spec.userConfig.s3_migration.secret_key-property'} (string, Pattern: `^[^\r\n]*$`). AWS secret key.
- [`snapshot_name`](#spec.userConfig.s3_migration.snapshot_name-property){: name='spec.userConfig.s3_migration.snapshot_name-property'} (string, Pattern: `^[^\r\n]*$`). The snapshot name to restore from.

**Optional**

- [`chunk_size`](#spec.userConfig.s3_migration.chunk_size-property){: name='spec.userConfig.s3_migration.chunk_size-property'} (string, Pattern: `^[^\r\n]*$`). Big files can be broken down into chunks during snapshotting if needed. Should be the same as for the 3rd party repository.
- [`compress`](#spec.userConfig.s3_migration.compress-property){: name='spec.userConfig.s3_migration.compress-property'} (boolean). when set to true metadata files are stored in compressed format.
- [`endpoint`](#spec.userConfig.s3_migration.endpoint-property){: name='spec.userConfig.s3_migration.endpoint-property'} (string, Pattern: `^[^\r\n]*$`). The S3 service endpoint to connect to. If you are using an S3-compatible service then you should set this to the service’s endpoint.
- [`include_aliases`](#spec.userConfig.s3_migration.include_aliases-property){: name='spec.userConfig.s3_migration.include_aliases-property'} (boolean). Whether to restore aliases alongside their associated indexes. Default is true.
- [`readonly`](#spec.userConfig.s3_migration.readonly-property){: name='spec.userConfig.s3_migration.readonly-property'} (boolean). Whether the repository is read-only.
- [`restore_global_state`](#spec.userConfig.s3_migration.restore_global_state-property){: name='spec.userConfig.s3_migration.restore_global_state-property'} (boolean). If true, restore the cluster state. Defaults to false.
- [`server_side_encryption`](#spec.userConfig.s3_migration.server_side_encryption-property){: name='spec.userConfig.s3_migration.server_side_encryption-property'} (boolean). When set to true files are encrypted on server side.

### saml {: #spec.userConfig.saml }

_Appears on [`spec.userConfig`](#spec.userConfig)._

OpenSearch SAML configuration.

**Required**

- [`enabled`](#spec.userConfig.saml.enabled-property){: name='spec.userConfig.saml.enabled-property'} (boolean). Enables or disables SAML-based authentication for OpenSearch. When enabled, users can authenticate using SAML with an Identity Provider.
- [`idp_entity_id`](#spec.userConfig.saml.idp_entity_id-property){: name='spec.userConfig.saml.idp_entity_id-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). The unique identifier for the Identity Provider (IdP) entity that is used for SAML authentication. This value is typically provided by the IdP.
- [`idp_metadata_url`](#spec.userConfig.saml.idp_metadata_url-property){: name='spec.userConfig.saml.idp_metadata_url-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 2048). The URL of the SAML metadata for the Identity Provider (IdP). This is used to configure SAML-based authentication with the IdP.
- [`sp_entity_id`](#spec.userConfig.saml.sp_entity_id-property){: name='spec.userConfig.saml.sp_entity_id-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 1024). The unique identifier for the Service Provider (SP) entity that is used for SAML authentication. This value is typically provided by the SP.

**Optional**

- [`idp_pemtrustedcas_content`](#spec.userConfig.saml.idp_pemtrustedcas_content-property){: name='spec.userConfig.saml.idp_pemtrustedcas_content-property'} (string, MaxLength: 16384). This parameter specifies the PEM-encoded root certificate authority (CA) content for the SAML identity provider (IdP) server verification. The root CA content is used to verify the SSL/TLS certificate presented by the server.
- [`roles_key`](#spec.userConfig.saml.roles_key-property){: name='spec.userConfig.saml.roles_key-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 256). Optional. Specifies the attribute in the SAML response where role information is stored, if available. Role attributes are not required for SAML authentication, but can be included in SAML assertions by most Identity Providers (IdPs) to determine user access levels or permissions.
- [`subject_key`](#spec.userConfig.saml.subject_key-property){: name='spec.userConfig.saml.subject_key-property'} (string, Pattern: `^[^\r\n]*$`, MinLength: 1, MaxLength: 256). Optional. Specifies the attribute in the SAML response where the subject identifier is stored. If not configured, the NameID attribute is used by default.
