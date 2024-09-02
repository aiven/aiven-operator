---
title: "Kafka"
---

## Usage example

??? example 
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: Kafka
    metadata:
      name: my-kafka
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      connInfoSecretTarget:
        name: kafka-secret
        prefix: MY_SECRET_PREFIX_
        annotations:
          foo: bar
        labels:
          baz: egg
    
      project: my-aiven-project
      cloudName: google-europe-west1
      plan: startup-2
    
      maintenanceWindowDow: friday
      maintenanceWindowTime: 23:00:00
    ```

!!! info
	To create this resource, a `Secret` containing Aiven token must be [created](/aiven-operator/authentication.html) first.

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `Kafka`:

```shell
kubectl get kafkas my-kafka
```

The output is similar to the following:
```shell
Name        Project             Region                 Plan         State      
my-kafka    my-aiven-project    google-europe-west1    startup-2    RUNNING    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret kafka-secret
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret kafka-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"KAFKA_HOST": "<secret>",
	"KAFKA_PORT": "<secret>",
	"KAFKA_USERNAME": "<secret>",
	"KAFKA_PASSWORD": "<secret>",
	"KAFKA_ACCESS_CERT": "<secret>",
	"KAFKA_ACCESS_KEY": "<secret>",
	"KAFKA_SASL_HOST": "<secret>",
	"KAFKA_SASL_PORT": "<secret>",
	"KAFKA_SCHEMA_REGISTRY_HOST": "<secret>",
	"KAFKA_SCHEMA_REGISTRY_PORT": "<secret>",
	"KAFKA_CONNECT_HOST": "<secret>",
	"KAFKA_CONNECT_PORT": "<secret>",
	"KAFKA_REST_HOST": "<secret>",
	"KAFKA_REST_PORT": "<secret>",
	"KAFKA_CA_CERT": "<secret>",
}
```

## Kafka {: #Kafka }

Kafka is the Schema for the kafkas API.

!!! Info "Exposes secret keys"

    `KAFKA_HOST`, `KAFKA_PORT`, `KAFKA_USERNAME`, `KAFKA_PASSWORD`, `KAFKA_ACCESS_CERT`, `KAFKA_ACCESS_KEY`, `KAFKA_SASL_HOST`, `KAFKA_SASL_PORT`, `KAFKA_SCHEMA_REGISTRY_HOST`, `KAFKA_SCHEMA_REGISTRY_PORT`, `KAFKA_CONNECT_HOST`, `KAFKA_CONNECT_PORT`, `KAFKA_REST_HOST`, `KAFKA_REST_PORT`, `KAFKA_CA_CERT`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `Kafka`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaSpec defines the desired state of Kafka. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`Kafka`](#Kafka)._

KafkaSpec defines the desired state of Kafka.

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
- [`karapace`](#spec.karapace-property){: name='spec.karapace-property'} (boolean). Switch the service to use Karapace for schema registry and REST proxy.
- [`maintenanceWindowDow`](#spec.maintenanceWindowDow-property){: name='spec.maintenanceWindowDow-property'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
- [`maintenanceWindowTime`](#spec.maintenanceWindowTime-property){: name='spec.maintenanceWindowTime-property'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
- [`projectVPCRef`](#spec.projectVPCRef-property){: name='spec.projectVPCRef-property'} (object). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See below for [nested schema](#spec.projectVPCRef).
- [`projectVpcId`](#spec.projectVpcId-property){: name='spec.projectVpcId-property'} (string, MaxLength: 36). Identifier of the VPC the service should be in, if any.
- [`serviceIntegrations`](#spec.serviceIntegrations-property){: name='spec.serviceIntegrations-property'} (array of objects, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See below for [nested schema](#spec.serviceIntegrations).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize services.
- [`technicalEmails`](#spec.technicalEmails-property){: name='spec.technicalEmails-property'} (array of objects, MaxItems: 10). Defines the email addresses that will receive alerts about upcoming maintenance updates or warnings about service instability. See below for [nested schema](#spec.technicalEmails).
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services.
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). Kafka specific user configuration options. See below for [nested schema](#spec.userConfig).

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

Kafka specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Deprecated. Additional Cloud Regions for Backup Replication.
- [`aiven_kafka_topic_messages`](#spec.userConfig.aiven_kafka_topic_messages-property){: name='spec.userConfig.aiven_kafka_topic_messages-property'} (boolean). Allow access to read Kafka topic messages in the Aiven Console and REST API.
- [`custom_domain`](#spec.userConfig.custom_domain-property){: name='spec.userConfig.custom_domain-property'} (string, MaxLength: 255). Serve the web frontend using a custom CNAME pointing to the Aiven DNS name.
- [`follower_fetching`](#spec.userConfig.follower_fetching-property){: name='spec.userConfig.follower_fetching-property'} (object). Enable follower fetching. See below for [nested schema](#spec.userConfig.follower_fetching).
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`kafka`](#spec.userConfig.kafka-property){: name='spec.userConfig.kafka-property'} (object). Kafka broker configuration values. See below for [nested schema](#spec.userConfig.kafka).
- [`kafka_authentication_methods`](#spec.userConfig.kafka_authentication_methods-property){: name='spec.userConfig.kafka_authentication_methods-property'} (object). Kafka authentication methods. See below for [nested schema](#spec.userConfig.kafka_authentication_methods).
- [`kafka_connect`](#spec.userConfig.kafka_connect-property){: name='spec.userConfig.kafka_connect-property'} (boolean). Enable Kafka Connect service.
- [`kafka_connect_config`](#spec.userConfig.kafka_connect_config-property){: name='spec.userConfig.kafka_connect_config-property'} (object). Kafka Connect configuration values. See below for [nested schema](#spec.userConfig.kafka_connect_config).
- [`kafka_connect_secret_providers`](#spec.userConfig.kafka_connect_secret_providers-property){: name='spec.userConfig.kafka_connect_secret_providers-property'} (array of objects). Configure external secret providers in order to reference external secrets in connector configuration. Currently Hashicorp Vault (provider: vault, auth_method: token) and AWS Secrets Manager (provider: aws, auth_method: credentials) are supported. Secrets can be referenced in connector config with ${<provider_name>:<secret_path>:<key_name>}. See below for [nested schema](#spec.userConfig.kafka_connect_secret_providers).
- [`kafka_rest`](#spec.userConfig.kafka_rest-property){: name='spec.userConfig.kafka_rest-property'} (boolean). Enable Kafka-REST service.
- [`kafka_rest_authorization`](#spec.userConfig.kafka_rest_authorization-property){: name='spec.userConfig.kafka_rest_authorization-property'} (boolean). Enable authorization in Kafka-REST service.
- [`kafka_rest_config`](#spec.userConfig.kafka_rest_config-property){: name='spec.userConfig.kafka_rest_config-property'} (object). Kafka REST configuration. See below for [nested schema](#spec.userConfig.kafka_rest_config).
- [`kafka_sasl_mechanisms`](#spec.userConfig.kafka_sasl_mechanisms-property){: name='spec.userConfig.kafka_sasl_mechanisms-property'} (object). Kafka SASL mechanisms. See below for [nested schema](#spec.userConfig.kafka_sasl_mechanisms).
- [`kafka_version`](#spec.userConfig.kafka_version-property){: name='spec.userConfig.kafka_version-property'} (string, Enum: `3.4`, `3.5`, `3.6`, `3.7`). Kafka major version.
- [`letsencrypt_sasl_privatelink`](#spec.userConfig.letsencrypt_sasl_privatelink-property){: name='spec.userConfig.letsencrypt_sasl_privatelink-property'} (boolean). Use Letsencrypt CA for Kafka SASL via Privatelink.
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`schema_registry`](#spec.userConfig.schema_registry-property){: name='spec.userConfig.schema_registry-property'} (boolean). Enable Schema-Registry service.
- [`schema_registry_config`](#spec.userConfig.schema_registry_config-property){: name='spec.userConfig.schema_registry_config-property'} (object). Schema Registry configuration. See below for [nested schema](#spec.userConfig.schema_registry_config).
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.
- [`tiered_storage`](#spec.userConfig.tiered_storage-property){: name='spec.userConfig.tiered_storage-property'} (object). Tiered storage configuration. See below for [nested schema](#spec.userConfig.tiered_storage).

### follower_fetching {: #spec.userConfig.follower_fetching }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Enable follower fetching.

**Required**

- [`enabled`](#spec.userConfig.follower_fetching.enabled-property){: name='spec.userConfig.follower_fetching.enabled-property'} (boolean). Whether to enable the follower fetching functionality.

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### kafka {: #spec.userConfig.kafka }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Kafka broker configuration values.

**Optional**

- [`auto_create_topics_enable`](#spec.userConfig.kafka.auto_create_topics_enable-property){: name='spec.userConfig.kafka.auto_create_topics_enable-property'} (boolean). Enable auto-creation of topics. (Default: true).
- [`compression_type`](#spec.userConfig.kafka.compression_type-property){: name='spec.userConfig.kafka.compression_type-property'} (string, Enum: `gzip`, `snappy`, `lz4`, `zstd`, `uncompressed`, `producer`). Specify the final compression type for a given topic. This configuration accepts the standard compression codecs (`gzip`, `snappy`, `lz4`, `zstd`). It additionally accepts `uncompressed` which is equivalent to no compression; and `producer` which means retain the original compression codec set by the producer.(Default: producer).
- [`connections_max_idle_ms`](#spec.userConfig.kafka.connections_max_idle_ms-property){: name='spec.userConfig.kafka.connections_max_idle_ms-property'} (integer, Minimum: 1000, Maximum: 3600000). Idle connections timeout: the server socket processor threads close the connections that idle for longer than this. (Default: 600000 ms (10 minutes)).
- [`default_replication_factor`](#spec.userConfig.kafka.default_replication_factor-property){: name='spec.userConfig.kafka.default_replication_factor-property'} (integer, Minimum: 1, Maximum: 10). Replication factor for auto-created topics (Default: 3).
- [`group_initial_rebalance_delay_ms`](#spec.userConfig.kafka.group_initial_rebalance_delay_ms-property){: name='spec.userConfig.kafka.group_initial_rebalance_delay_ms-property'} (integer, Minimum: 0, Maximum: 300000). The amount of time, in milliseconds, the group coordinator will wait for more consumers to join a new group before performing the first rebalance. A longer delay means potentially fewer rebalances, but increases the time until processing begins. The default value for this is 3 seconds. During development and testing it might be desirable to set this to 0 in order to not delay test execution time. (Default: 3000 ms (3 seconds)).
- [`group_max_session_timeout_ms`](#spec.userConfig.kafka.group_max_session_timeout_ms-property){: name='spec.userConfig.kafka.group_max_session_timeout_ms-property'} (integer, Minimum: 0, Maximum: 1800000). The maximum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures. Default: 1800000 ms (30 minutes).
- [`group_min_session_timeout_ms`](#spec.userConfig.kafka.group_min_session_timeout_ms-property){: name='spec.userConfig.kafka.group_min_session_timeout_ms-property'} (integer, Minimum: 0, Maximum: 60000). The minimum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures. (Default: 6000 ms (6 seconds)).
- [`log_cleaner_delete_retention_ms`](#spec.userConfig.kafka.log_cleaner_delete_retention_ms-property){: name='spec.userConfig.kafka.log_cleaner_delete_retention_ms-property'} (integer, Minimum: 0, Maximum: 315569260000). How long are delete records retained? (Default: 86400000 (1 day)).
- [`log_cleaner_max_compaction_lag_ms`](#spec.userConfig.kafka.log_cleaner_max_compaction_lag_ms-property){: name='spec.userConfig.kafka.log_cleaner_max_compaction_lag_ms-property'} (integer, Minimum: 30000). The maximum amount of time message will remain uncompacted. Only applicable for logs that are being compacted. (Default: 9223372036854775807 ms (Long.MAX_VALUE)).
- [`log_cleaner_min_cleanable_ratio`](#spec.userConfig.kafka.log_cleaner_min_cleanable_ratio-property){: name='spec.userConfig.kafka.log_cleaner_min_cleanable_ratio-property'} (number, Minimum: 0.2, Maximum: 0.9). Controls log compactor frequency. Larger value means more frequent compactions but also more space wasted for logs. Consider setting log.cleaner.max.compaction.lag.ms to enforce compactions sooner, instead of setting a very high value for this option. (Default: 0.5).
- [`log_cleaner_min_compaction_lag_ms`](#spec.userConfig.kafka.log_cleaner_min_compaction_lag_ms-property){: name='spec.userConfig.kafka.log_cleaner_min_compaction_lag_ms-property'} (integer, Minimum: 0). The minimum time a message will remain uncompacted in the log. Only applicable for logs that are being compacted. (Default: 0 ms).
- [`log_cleanup_policy`](#spec.userConfig.kafka.log_cleanup_policy-property){: name='spec.userConfig.kafka.log_cleanup_policy-property'} (string, Enum: `delete`, `compact`, `compact,delete`). The default cleanup policy for segments beyond the retention window (Default: delete).
- [`log_flush_interval_messages`](#spec.userConfig.kafka.log_flush_interval_messages-property){: name='spec.userConfig.kafka.log_flush_interval_messages-property'} (integer, Minimum: 1). The number of messages accumulated on a log partition before messages are flushed to disk (Default: 9223372036854775807 (Long.MAX_VALUE)).
- [`log_flush_interval_ms`](#spec.userConfig.kafka.log_flush_interval_ms-property){: name='spec.userConfig.kafka.log_flush_interval_ms-property'} (integer, Minimum: 0). The maximum time in ms that a message in any topic is kept in memory (page-cache) before flushed to disk. If not set, the value in log.flush.scheduler.interval.ms is used (Default: null).
- [`log_index_interval_bytes`](#spec.userConfig.kafka.log_index_interval_bytes-property){: name='spec.userConfig.kafka.log_index_interval_bytes-property'} (integer, Minimum: 0, Maximum: 104857600). The interval with which Kafka adds an entry to the offset index (Default: 4096 bytes (4 kibibytes)).
- [`log_index_size_max_bytes`](#spec.userConfig.kafka.log_index_size_max_bytes-property){: name='spec.userConfig.kafka.log_index_size_max_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). The maximum size in bytes of the offset index (Default: 10485760 (10 mebibytes)).
- [`log_local_retention_bytes`](#spec.userConfig.kafka.log_local_retention_bytes-property){: name='spec.userConfig.kafka.log_local_retention_bytes-property'} (integer, Minimum: -2). The maximum size of local log segments that can grow for a partition before it gets eligible for deletion. If set to -2, the value of log.retention.bytes is used. The effective value should always be less than or equal to log.retention.bytes value. (Default: -2).
- [`log_local_retention_ms`](#spec.userConfig.kafka.log_local_retention_ms-property){: name='spec.userConfig.kafka.log_local_retention_ms-property'} (integer, Minimum: -2). The number of milliseconds to keep the local log segments before it gets eligible for deletion. If set to -2, the value of log.retention.ms is used. The effective value should always be less than or equal to log.retention.ms value. (Default: -2).
- [`log_message_downconversion_enable`](#spec.userConfig.kafka.log_message_downconversion_enable-property){: name='spec.userConfig.kafka.log_message_downconversion_enable-property'} (boolean). This configuration controls whether down-conversion of message formats is enabled to satisfy consume requests. (Default: true).
- [`log_message_timestamp_difference_max_ms`](#spec.userConfig.kafka.log_message_timestamp_difference_max_ms-property){: name='spec.userConfig.kafka.log_message_timestamp_difference_max_ms-property'} (integer, Minimum: 0). The maximum difference allowed between the timestamp when a broker receives a message and the timestamp specified in the message (Default: 9223372036854775807 (Long.MAX_VALUE)).
- [`log_message_timestamp_type`](#spec.userConfig.kafka.log_message_timestamp_type-property){: name='spec.userConfig.kafka.log_message_timestamp_type-property'} (string, Enum: `CreateTime`, `LogAppendTime`). Define whether the timestamp in the message is message create time or log append time. (Default: CreateTime).
- [`log_preallocate`](#spec.userConfig.kafka.log_preallocate-property){: name='spec.userConfig.kafka.log_preallocate-property'} (boolean). Should pre allocate file when create new segment? (Default: false).
- [`log_retention_bytes`](#spec.userConfig.kafka.log_retention_bytes-property){: name='spec.userConfig.kafka.log_retention_bytes-property'} (integer, Minimum: -1). The maximum size of the log before deleting messages (Default: -1).
- [`log_retention_hours`](#spec.userConfig.kafka.log_retention_hours-property){: name='spec.userConfig.kafka.log_retention_hours-property'} (integer, Minimum: -1, Maximum: 2147483647). The number of hours to keep a log file before deleting it (Default: 168 hours (1 week)).
- [`log_retention_ms`](#spec.userConfig.kafka.log_retention_ms-property){: name='spec.userConfig.kafka.log_retention_ms-property'} (integer, Minimum: -1). The number of milliseconds to keep a log file before deleting it (in milliseconds), If not set, the value in log.retention.minutes is used. If set to -1, no time limit is applied. (Default: null, log.retention.hours applies).
- [`log_roll_jitter_ms`](#spec.userConfig.kafka.log_roll_jitter_ms-property){: name='spec.userConfig.kafka.log_roll_jitter_ms-property'} (integer, Minimum: 0). The maximum jitter to subtract from logRollTimeMillis (in milliseconds). If not set, the value in log.roll.jitter.hours is used (Default: null).
- [`log_roll_ms`](#spec.userConfig.kafka.log_roll_ms-property){: name='spec.userConfig.kafka.log_roll_ms-property'} (integer, Minimum: 1). The maximum time before a new log segment is rolled out (in milliseconds). (Default: null, log.roll.hours applies (Default: 168, 7 days)).
- [`log_segment_bytes`](#spec.userConfig.kafka.log_segment_bytes-property){: name='spec.userConfig.kafka.log_segment_bytes-property'} (integer, Minimum: 10485760, Maximum: 1073741824). The maximum size of a single log file (Default: 1073741824 bytes (1 gibibyte)).
- [`log_segment_delete_delay_ms`](#spec.userConfig.kafka.log_segment_delete_delay_ms-property){: name='spec.userConfig.kafka.log_segment_delete_delay_ms-property'} (integer, Minimum: 0, Maximum: 3600000). The amount of time to wait before deleting a file from the filesystem (Default: 60000 ms (1 minute)).
- [`max_connections_per_ip`](#spec.userConfig.kafka.max_connections_per_ip-property){: name='spec.userConfig.kafka.max_connections_per_ip-property'} (integer, Minimum: 256, Maximum: 2147483647). The maximum number of connections allowed from each ip address (Default: 2147483647).
- [`max_incremental_fetch_session_cache_slots`](#spec.userConfig.kafka.max_incremental_fetch_session_cache_slots-property){: name='spec.userConfig.kafka.max_incremental_fetch_session_cache_slots-property'} (integer, Minimum: 1000, Maximum: 10000). The maximum number of incremental fetch sessions that the broker will maintain. (Default: 1000).
- [`message_max_bytes`](#spec.userConfig.kafka.message_max_bytes-property){: name='spec.userConfig.kafka.message_max_bytes-property'} (integer, Minimum: 0, Maximum: 100001200). The maximum size of message that the server can receive. (Default: 1048588 bytes (1 mebibyte + 12 bytes)).
- [`min_insync_replicas`](#spec.userConfig.kafka.min_insync_replicas-property){: name='spec.userConfig.kafka.min_insync_replicas-property'} (integer, Minimum: 1, Maximum: 7). When a producer sets acks to `all` (or `-1`), min.insync.replicas specifies the minimum number of replicas that must acknowledge a write for the write to be considered successful. (Default: 1).
- [`num_partitions`](#spec.userConfig.kafka.num_partitions-property){: name='spec.userConfig.kafka.num_partitions-property'} (integer, Minimum: 1, Maximum: 1000). Number of partitions for auto-created topics (Default: 1).
- [`offsets_retention_minutes`](#spec.userConfig.kafka.offsets_retention_minutes-property){: name='spec.userConfig.kafka.offsets_retention_minutes-property'} (integer, Minimum: 1, Maximum: 2147483647). Log retention window in minutes for offsets topic (Default: 10080 minutes (7 days)).
- [`producer_purgatory_purge_interval_requests`](#spec.userConfig.kafka.producer_purgatory_purge_interval_requests-property){: name='spec.userConfig.kafka.producer_purgatory_purge_interval_requests-property'} (integer, Minimum: 10, Maximum: 10000). The purge interval (in number of requests) of the producer request purgatory (Default: 1000).
- [`replica_fetch_max_bytes`](#spec.userConfig.kafka.replica_fetch_max_bytes-property){: name='spec.userConfig.kafka.replica_fetch_max_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). The number of bytes of messages to attempt to fetch for each partition . This is not an absolute maximum, if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made. (Default: 1048576 bytes (1 mebibytes)).
- [`replica_fetch_response_max_bytes`](#spec.userConfig.kafka.replica_fetch_response_max_bytes-property){: name='spec.userConfig.kafka.replica_fetch_response_max_bytes-property'} (integer, Minimum: 10485760, Maximum: 1048576000). Maximum bytes expected for the entire fetch response. Records are fetched in batches, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made. As such, this is not an absolute maximum. (Default: 10485760 bytes (10 mebibytes)).
- [`sasl_oauthbearer_expected_audience`](#spec.userConfig.kafka.sasl_oauthbearer_expected_audience-property){: name='spec.userConfig.kafka.sasl_oauthbearer_expected_audience-property'} (string, Pattern: `^[^\r\n]*$`, MaxLength: 128). The (optional) comma-delimited setting for the broker to use to verify that the JWT was issued for one of the expected audiences. (Default: null).
- [`sasl_oauthbearer_expected_issuer`](#spec.userConfig.kafka.sasl_oauthbearer_expected_issuer-property){: name='spec.userConfig.kafka.sasl_oauthbearer_expected_issuer-property'} (string, Pattern: `^[^\r\n]*$`, MaxLength: 128). Optional setting for the broker to use to verify that the JWT was created by the expected issuer.(Default: null).
- [`sasl_oauthbearer_jwks_endpoint_url`](#spec.userConfig.kafka.sasl_oauthbearer_jwks_endpoint_url-property){: name='spec.userConfig.kafka.sasl_oauthbearer_jwks_endpoint_url-property'} (string, MaxLength: 2048). OIDC JWKS endpoint URL. By setting this the SASL SSL OAuth2/OIDC authentication is enabled. See also other options for SASL OAuth2/OIDC. (Default: null).
- [`sasl_oauthbearer_sub_claim_name`](#spec.userConfig.kafka.sasl_oauthbearer_sub_claim_name-property){: name='spec.userConfig.kafka.sasl_oauthbearer_sub_claim_name-property'} (string, Pattern: `^[^\r\n]*\S[^\r\n]*$`, MaxLength: 128). Name of the scope from which to extract the subject claim from the JWT.(Default: sub).
- [`socket_request_max_bytes`](#spec.userConfig.kafka.socket_request_max_bytes-property){: name='spec.userConfig.kafka.socket_request_max_bytes-property'} (integer, Minimum: 10485760, Maximum: 209715200). The maximum number of bytes in a socket request (Default: 104857600 bytes).
- [`transaction_partition_verification_enable`](#spec.userConfig.kafka.transaction_partition_verification_enable-property){: name='spec.userConfig.kafka.transaction_partition_verification_enable-property'} (boolean). Enable verification that checks that the partition has been added to the transaction before writing transactional records to the partition. (Default: true).
- [`transaction_remove_expired_transaction_cleanup_interval_ms`](#spec.userConfig.kafka.transaction_remove_expired_transaction_cleanup_interval_ms-property){: name='spec.userConfig.kafka.transaction_remove_expired_transaction_cleanup_interval_ms-property'} (integer, Minimum: 600000, Maximum: 3600000). The interval at which to remove transactions that have expired due to transactional.id.expiration.ms passing (Default: 3600000 ms (1 hour)).
- [`transaction_state_log_segment_bytes`](#spec.userConfig.kafka.transaction_state_log_segment_bytes-property){: name='spec.userConfig.kafka.transaction_state_log_segment_bytes-property'} (integer, Minimum: 1048576, Maximum: 2147483647). The transaction topic segment bytes should be kept relatively small in order to facilitate faster log compaction and cache loads (Default: 104857600 bytes (100 mebibytes)).

### kafka_authentication_methods {: #spec.userConfig.kafka_authentication_methods }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Kafka authentication methods.

**Optional**

- [`certificate`](#spec.userConfig.kafka_authentication_methods.certificate-property){: name='spec.userConfig.kafka_authentication_methods.certificate-property'} (boolean). Enable certificate/SSL authentication.
- [`sasl`](#spec.userConfig.kafka_authentication_methods.sasl-property){: name='spec.userConfig.kafka_authentication_methods.sasl-property'} (boolean). Enable SASL authentication.

### kafka_connect_config {: #spec.userConfig.kafka_connect_config }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Kafka Connect configuration values.

**Optional**

- [`connector_client_config_override_policy`](#spec.userConfig.kafka_connect_config.connector_client_config_override_policy-property){: name='spec.userConfig.kafka_connect_config.connector_client_config_override_policy-property'} (string, Enum: `None`, `All`). Defines what client configurations can be overridden by the connector. Default is None.
- [`consumer_auto_offset_reset`](#spec.userConfig.kafka_connect_config.consumer_auto_offset_reset-property){: name='spec.userConfig.kafka_connect_config.consumer_auto_offset_reset-property'} (string, Enum: `earliest`, `latest`). What to do when there is no initial offset in Kafka or if the current offset does not exist any more on the server. Default is earliest.
- [`consumer_fetch_max_bytes`](#spec.userConfig.kafka_connect_config.consumer_fetch_max_bytes-property){: name='spec.userConfig.kafka_connect_config.consumer_fetch_max_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). Records are fetched in batches by the consumer, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that the consumer can make progress. As such, this is not a absolute maximum.
- [`consumer_isolation_level`](#spec.userConfig.kafka_connect_config.consumer_isolation_level-property){: name='spec.userConfig.kafka_connect_config.consumer_isolation_level-property'} (string, Enum: `read_uncommitted`, `read_committed`). Transaction read isolation level. read_uncommitted is the default, but read_committed can be used if consume-exactly-once behavior is desired.
- [`consumer_max_partition_fetch_bytes`](#spec.userConfig.kafka_connect_config.consumer_max_partition_fetch_bytes-property){: name='spec.userConfig.kafka_connect_config.consumer_max_partition_fetch_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). Records are fetched in batches by the consumer.If the first record batch in the first non-empty partition of the fetch is larger than this limit, the batch will still be returned to ensure that the consumer can make progress.
- [`consumer_max_poll_interval_ms`](#spec.userConfig.kafka_connect_config.consumer_max_poll_interval_ms-property){: name='spec.userConfig.kafka_connect_config.consumer_max_poll_interval_ms-property'} (integer, Minimum: 1, Maximum: 2147483647). The maximum delay in milliseconds between invocations of poll() when using consumer group management (defaults to 300000).
- [`consumer_max_poll_records`](#spec.userConfig.kafka_connect_config.consumer_max_poll_records-property){: name='spec.userConfig.kafka_connect_config.consumer_max_poll_records-property'} (integer, Minimum: 1, Maximum: 10000). The maximum number of records returned in a single call to poll() (defaults to 500).
- [`offset_flush_interval_ms`](#spec.userConfig.kafka_connect_config.offset_flush_interval_ms-property){: name='spec.userConfig.kafka_connect_config.offset_flush_interval_ms-property'} (integer, Minimum: 1, Maximum: 100000000). The interval at which to try committing offsets for tasks (defaults to 60000).
- [`offset_flush_timeout_ms`](#spec.userConfig.kafka_connect_config.offset_flush_timeout_ms-property){: name='spec.userConfig.kafka_connect_config.offset_flush_timeout_ms-property'} (integer, Minimum: 1, Maximum: 2147483647). Maximum number of milliseconds to wait for records to flush and partition offset data to be committed to offset storage before cancelling the process and restoring the offset data to be committed in a future attempt (defaults to 5000).
- [`producer_batch_size`](#spec.userConfig.kafka_connect_config.producer_batch_size-property){: name='spec.userConfig.kafka_connect_config.producer_batch_size-property'} (integer, Minimum: 0, Maximum: 5242880). This setting gives the upper bound of the batch size to be sent. If there are fewer than this many bytes accumulated for this partition, the producer will `linger` for the linger.ms time waiting for more records to show up. A batch size of zero will disable batching entirely (defaults to 16384).
- [`producer_buffer_memory`](#spec.userConfig.kafka_connect_config.producer_buffer_memory-property){: name='spec.userConfig.kafka_connect_config.producer_buffer_memory-property'} (integer, Minimum: 5242880, Maximum: 134217728). The total bytes of memory the producer can use to buffer records waiting to be sent to the broker (defaults to 33554432).
- [`producer_compression_type`](#spec.userConfig.kafka_connect_config.producer_compression_type-property){: name='spec.userConfig.kafka_connect_config.producer_compression_type-property'} (string, Enum: `gzip`, `snappy`, `lz4`, `zstd`, `none`). Specify the default compression type for producers. This configuration accepts the standard compression codecs (`gzip`, `snappy`, `lz4`, `zstd`). It additionally accepts `none` which is the default and equivalent to no compression.
- [`producer_linger_ms`](#spec.userConfig.kafka_connect_config.producer_linger_ms-property){: name='spec.userConfig.kafka_connect_config.producer_linger_ms-property'} (integer, Minimum: 0, Maximum: 5000). This setting gives the upper bound on the delay for batching: once there is batch.size worth of records for a partition it will be sent immediately regardless of this setting, however if there are fewer than this many bytes accumulated for this partition the producer will `linger` for the specified time waiting for more records to show up. Defaults to 0.
- [`producer_max_request_size`](#spec.userConfig.kafka_connect_config.producer_max_request_size-property){: name='spec.userConfig.kafka_connect_config.producer_max_request_size-property'} (integer, Minimum: 131072, Maximum: 67108864). This setting will limit the number of record batches the producer will send in a single request to avoid sending huge requests.
- [`scheduled_rebalance_max_delay_ms`](#spec.userConfig.kafka_connect_config.scheduled_rebalance_max_delay_ms-property){: name='spec.userConfig.kafka_connect_config.scheduled_rebalance_max_delay_ms-property'} (integer, Minimum: 0, Maximum: 600000). The maximum delay that is scheduled in order to wait for the return of one or more departed workers before rebalancing and reassigning their connectors and tasks to the group. During this period the connectors and tasks of the departed workers remain unassigned. Defaults to 5 minutes.
- [`session_timeout_ms`](#spec.userConfig.kafka_connect_config.session_timeout_ms-property){: name='spec.userConfig.kafka_connect_config.session_timeout_ms-property'} (integer, Minimum: 1, Maximum: 2147483647). The timeout in milliseconds used to detect failures when using Kafka’s group management facilities (defaults to 10000).

### kafka_connect_secret_providers {: #spec.userConfig.kafka_connect_secret_providers }

_Appears on [`spec.userConfig`](#spec.userConfig)._

SecretProvider.

**Required**

- [`name`](#spec.userConfig.kafka_connect_secret_providers.name-property){: name='spec.userConfig.kafka_connect_secret_providers.name-property'} (string). Name of the secret provider. Used to reference secrets in connector config.

**Optional**

- [`aws`](#spec.userConfig.kafka_connect_secret_providers.aws-property){: name='spec.userConfig.kafka_connect_secret_providers.aws-property'} (object). AWS config for Secret Provider. See below for [nested schema](#spec.userConfig.kafka_connect_secret_providers.aws).
- [`vault`](#spec.userConfig.kafka_connect_secret_providers.vault-property){: name='spec.userConfig.kafka_connect_secret_providers.vault-property'} (object). Vault Config for Secret Provider. See below for [nested schema](#spec.userConfig.kafka_connect_secret_providers.vault).

#### aws {: #spec.userConfig.kafka_connect_secret_providers.aws }

_Appears on [`spec.userConfig.kafka_connect_secret_providers`](#spec.userConfig.kafka_connect_secret_providers)._

AWS config for Secret Provider.

**Required**

- [`auth_method`](#spec.userConfig.kafka_connect_secret_providers.aws.auth_method-property){: name='spec.userConfig.kafka_connect_secret_providers.aws.auth_method-property'} (string, Enum: `credentials`). Auth method of the vault secret provider.
- [`region`](#spec.userConfig.kafka_connect_secret_providers.aws.region-property){: name='spec.userConfig.kafka_connect_secret_providers.aws.region-property'} (string, MaxLength: 64). Region used to lookup secrets with AWS SecretManager.

**Optional**

- [`access_key`](#spec.userConfig.kafka_connect_secret_providers.aws.access_key-property){: name='spec.userConfig.kafka_connect_secret_providers.aws.access_key-property'} (string, MaxLength: 128). Access key used to authenticate with aws.
- [`secret_key`](#spec.userConfig.kafka_connect_secret_providers.aws.secret_key-property){: name='spec.userConfig.kafka_connect_secret_providers.aws.secret_key-property'} (string, MaxLength: 128). Secret key used to authenticate with aws.

#### vault {: #spec.userConfig.kafka_connect_secret_providers.vault }

_Appears on [`spec.userConfig.kafka_connect_secret_providers`](#spec.userConfig.kafka_connect_secret_providers)._

Vault Config for Secret Provider.

**Required**

- [`address`](#spec.userConfig.kafka_connect_secret_providers.vault.address-property){: name='spec.userConfig.kafka_connect_secret_providers.vault.address-property'} (string, MinLength: 1, MaxLength: 65536). Address of the Vault server.
- [`auth_method`](#spec.userConfig.kafka_connect_secret_providers.vault.auth_method-property){: name='spec.userConfig.kafka_connect_secret_providers.vault.auth_method-property'} (string, Enum: `token`). Auth method of the vault secret provider.

**Optional**

- [`engine_version`](#spec.userConfig.kafka_connect_secret_providers.vault.engine_version-property){: name='spec.userConfig.kafka_connect_secret_providers.vault.engine_version-property'} (integer, Enum: `1`, `2`). KV Secrets Engine version of the Vault server instance.
- [`prefix_path_depth`](#spec.userConfig.kafka_connect_secret_providers.vault.prefix_path_depth-property){: name='spec.userConfig.kafka_connect_secret_providers.vault.prefix_path_depth-property'} (integer). Prefix path depth of the secrets Engine. Default is 1. If the secrets engine path has more than one segment it has to be increased to the number of segments.
- [`token`](#spec.userConfig.kafka_connect_secret_providers.vault.token-property){: name='spec.userConfig.kafka_connect_secret_providers.vault.token-property'} (string, MaxLength: 256). Token used to authenticate with vault and auth method `token`.

### kafka_rest_config {: #spec.userConfig.kafka_rest_config }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Kafka REST configuration.

**Optional**

- [`consumer_enable_auto_commit`](#spec.userConfig.kafka_rest_config.consumer_enable_auto_commit-property){: name='spec.userConfig.kafka_rest_config.consumer_enable_auto_commit-property'} (boolean). If true the consumer's offset will be periodically committed to Kafka in the background.
- [`consumer_request_max_bytes`](#spec.userConfig.kafka_rest_config.consumer_request_max_bytes-property){: name='spec.userConfig.kafka_rest_config.consumer_request_max_bytes-property'} (integer, Minimum: 0, Maximum: 671088640). Maximum number of bytes in unencoded message keys and values by a single request.
- [`consumer_request_timeout_ms`](#spec.userConfig.kafka_rest_config.consumer_request_timeout_ms-property){: name='spec.userConfig.kafka_rest_config.consumer_request_timeout_ms-property'} (integer, Enum: `1000`, `15000`, `30000`, Minimum: 1000, Maximum: 30000). The maximum total time to wait for messages for a request if the maximum number of messages has not yet been reached.
- [`name_strategy`](#spec.userConfig.kafka_rest_config.name_strategy-property){: name='spec.userConfig.kafka_rest_config.name_strategy-property'} (string, Enum: `topic_name`, `record_name`, `topic_record_name`). Name strategy to use when selecting subject for storing schemas.
- [`name_strategy_validation`](#spec.userConfig.kafka_rest_config.name_strategy_validation-property){: name='spec.userConfig.kafka_rest_config.name_strategy_validation-property'} (boolean). If true, validate that given schema is registered under expected subject name by the used name strategy when producing messages.
- [`producer_acks`](#spec.userConfig.kafka_rest_config.producer_acks-property){: name='spec.userConfig.kafka_rest_config.producer_acks-property'} (string, Enum: `all`, `-1`, `0`, `1`). The number of acknowledgments the producer requires the leader to have received before considering a request complete. If set to `all` or `-1`, the leader will wait for the full set of in-sync replicas to acknowledge the record.
- [`producer_compression_type`](#spec.userConfig.kafka_rest_config.producer_compression_type-property){: name='spec.userConfig.kafka_rest_config.producer_compression_type-property'} (string, Enum: `gzip`, `snappy`, `lz4`, `zstd`, `none`). Specify the default compression type for producers. This configuration accepts the standard compression codecs (`gzip`, `snappy`, `lz4`, `zstd`). It additionally accepts `none` which is the default and equivalent to no compression.
- [`producer_linger_ms`](#spec.userConfig.kafka_rest_config.producer_linger_ms-property){: name='spec.userConfig.kafka_rest_config.producer_linger_ms-property'} (integer, Minimum: 0, Maximum: 5000). Wait for up to the given delay to allow batching records together.
- [`producer_max_request_size`](#spec.userConfig.kafka_rest_config.producer_max_request_size-property){: name='spec.userConfig.kafka_rest_config.producer_max_request_size-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum size of a request in bytes. Note that Kafka broker can also cap the record batch size.
- [`simpleconsumer_pool_size_max`](#spec.userConfig.kafka_rest_config.simpleconsumer_pool_size_max-property){: name='spec.userConfig.kafka_rest_config.simpleconsumer_pool_size_max-property'} (integer, Minimum: 10, Maximum: 250). Maximum number of SimpleConsumers that can be instantiated per broker.

### kafka_sasl_mechanisms {: #spec.userConfig.kafka_sasl_mechanisms }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Kafka SASL mechanisms.

**Optional**

- [`plain`](#spec.userConfig.kafka_sasl_mechanisms.plain-property){: name='spec.userConfig.kafka_sasl_mechanisms.plain-property'} (boolean). Enable PLAIN mechanism.
- [`scram_sha_256`](#spec.userConfig.kafka_sasl_mechanisms.scram_sha_256-property){: name='spec.userConfig.kafka_sasl_mechanisms.scram_sha_256-property'} (boolean). Enable SCRAM-SHA-256 mechanism.
- [`scram_sha_512`](#spec.userConfig.kafka_sasl_mechanisms.scram_sha_512-property){: name='spec.userConfig.kafka_sasl_mechanisms.scram_sha_512-property'} (boolean). Enable SCRAM-SHA-512 mechanism.

### private_access {: #spec.userConfig.private_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from private networks.

**Optional**

- [`kafka`](#spec.userConfig.private_access.kafka-property){: name='spec.userConfig.private_access.kafka-property'} (boolean). Allow clients to connect to kafka with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`kafka_connect`](#spec.userConfig.private_access.kafka_connect-property){: name='spec.userConfig.private_access.kafka_connect-property'} (boolean). Allow clients to connect to kafka_connect with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`kafka_rest`](#spec.userConfig.private_access.kafka_rest-property){: name='spec.userConfig.private_access.kafka_rest-property'} (boolean). Allow clients to connect to kafka_rest with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`schema_registry`](#spec.userConfig.private_access.schema_registry-property){: name='spec.userConfig.private_access.schema_registry-property'} (boolean). Allow clients to connect to schema_registry with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service components through Privatelink.

**Optional**

- [`jolokia`](#spec.userConfig.privatelink_access.jolokia-property){: name='spec.userConfig.privatelink_access.jolokia-property'} (boolean). Enable jolokia.
- [`kafka`](#spec.userConfig.privatelink_access.kafka-property){: name='spec.userConfig.privatelink_access.kafka-property'} (boolean). Enable kafka.
- [`kafka_connect`](#spec.userConfig.privatelink_access.kafka_connect-property){: name='spec.userConfig.privatelink_access.kafka_connect-property'} (boolean). Enable kafka_connect.
- [`kafka_rest`](#spec.userConfig.privatelink_access.kafka_rest-property){: name='spec.userConfig.privatelink_access.kafka_rest-property'} (boolean). Enable kafka_rest.
- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.
- [`schema_registry`](#spec.userConfig.privatelink_access.schema_registry-property){: name='spec.userConfig.privatelink_access.schema_registry-property'} (boolean). Enable schema_registry.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Optional**

- [`kafka`](#spec.userConfig.public_access.kafka-property){: name='spec.userConfig.public_access.kafka-property'} (boolean). Allow clients to connect to kafka from the public internet for service nodes that are in a project VPC or another type of private network.
- [`kafka_connect`](#spec.userConfig.public_access.kafka_connect-property){: name='spec.userConfig.public_access.kafka_connect-property'} (boolean). Allow clients to connect to kafka_connect from the public internet for service nodes that are in a project VPC or another type of private network.
- [`kafka_rest`](#spec.userConfig.public_access.kafka_rest-property){: name='spec.userConfig.public_access.kafka_rest-property'} (boolean). Allow clients to connect to kafka_rest from the public internet for service nodes that are in a project VPC or another type of private network.
- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.
- [`schema_registry`](#spec.userConfig.public_access.schema_registry-property){: name='spec.userConfig.public_access.schema_registry-property'} (boolean). Allow clients to connect to schema_registry from the public internet for service nodes that are in a project VPC or another type of private network.

### schema_registry_config {: #spec.userConfig.schema_registry_config }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Schema Registry configuration.

**Optional**

- [`leader_eligibility`](#spec.userConfig.schema_registry_config.leader_eligibility-property){: name='spec.userConfig.schema_registry_config.leader_eligibility-property'} (boolean). If true, Karapace / Schema Registry on the service nodes can participate in leader election. It might be needed to disable this when the schemas topic is replicated to a secondary cluster and Karapace / Schema Registry there must not participate in leader election. Defaults to `true`.
- [`topic_name`](#spec.userConfig.schema_registry_config.topic_name-property){: name='spec.userConfig.schema_registry_config.topic_name-property'} (string, MinLength: 1, MaxLength: 249). The durable single partition topic that acts as the durable log for the data. This topic must be compacted to avoid losing data due to retention policy. Please note that changing this configuration in an existing Schema Registry / Karapace setup leads to previous schemas being inaccessible, data encoded with them potentially unreadable and schema ID sequence put out of order. It's only possible to do the switch while Schema Registry / Karapace is disabled. Defaults to `_schemas`.

### tiered_storage {: #spec.userConfig.tiered_storage }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Tiered storage configuration.

**Optional**

- [`enabled`](#spec.userConfig.tiered_storage.enabled-property){: name='spec.userConfig.tiered_storage.enabled-property'} (boolean). Whether to enable the tiered storage functionality.
- [`local_cache`](#spec.userConfig.tiered_storage.local_cache-property){: name='spec.userConfig.tiered_storage.local_cache-property'} (object). Deprecated. Local cache configuration. See below for [nested schema](#spec.userConfig.tiered_storage.local_cache).

#### local_cache {: #spec.userConfig.tiered_storage.local_cache }

_Appears on [`spec.userConfig.tiered_storage`](#spec.userConfig.tiered_storage)._

Deprecated. Local cache configuration.

**Required**

- [`size`](#spec.userConfig.tiered_storage.local_cache.size-property){: name='spec.userConfig.tiered_storage.local_cache.size-property'} (integer, Minimum: 1, Maximum: 107374182400). Deprecated. Local cache size in bytes.
