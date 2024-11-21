---
title: "KafkaConnect"
---

## Usage example

??? example 
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: KafkaConnect
    metadata:
      name: my-kafka-connect
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: my-aiven-project
      cloudName: google-europe-west1
      plan: business-4
    
      userConfig:
        kafka_connect:
          consumer_isolation_level: read_committed
        public_access:
          kafka_connect: true
    ```

!!! info
	To create this resource, a `Secret` containing Aiven token must be [created](/aiven-operator/authentication.html) first.

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `KafkaConnect`:

```shell
kubectl get kafkaconnects my-kafka-connect
```

The output is similar to the following:
```shell
Name                Project             Region                 Plan          State      
my-kafka-connect    my-aiven-project    google-europe-west1    business-4    RUNNING    
```

## KafkaConnect {: #KafkaConnect }

KafkaConnect is the Schema for the kafkaconnects API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `KafkaConnect`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaConnectSpec defines the desired state of KafkaConnect. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`KafkaConnect`](#KafkaConnect)._

KafkaConnectSpec defines the desired state of KafkaConnect.

**Required**

- [`plan`](#spec.plan-property){: name='spec.plan-property'} (string, MaxLength: 128). Subscription plan.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, MaxLength: 256). Cloud the service runs in.
- [`maintenanceWindowDow`](#spec.maintenanceWindowDow-property){: name='spec.maintenanceWindowDow-property'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
- [`maintenanceWindowTime`](#spec.maintenanceWindowTime-property){: name='spec.maintenanceWindowTime-property'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
- [`projectVPCRef`](#spec.projectVPCRef-property){: name='spec.projectVPCRef-property'} (object). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See below for [nested schema](#spec.projectVPCRef).
- [`projectVpcId`](#spec.projectVpcId-property){: name='spec.projectVpcId-property'} (string, MaxLength: 36). Identifier of the VPC the service should be in, if any.
- [`serviceIntegrations`](#spec.serviceIntegrations-property){: name='spec.serviceIntegrations-property'} (array of objects, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See below for [nested schema](#spec.serviceIntegrations).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize services.
- [`technicalEmails`](#spec.technicalEmails-property){: name='spec.technicalEmails-property'} (array of objects, MaxItems: 10). Defines the email addresses that will receive alerts about upcoming maintenance updates or warnings about service instability. See below for [nested schema](#spec.technicalEmails).
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services.
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). KafkaConnect specific user configuration options. See below for [nested schema](#spec.userConfig).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

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

KafkaConnect specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Deprecated. Additional Cloud Regions for Backup Replication.
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`kafka_connect`](#spec.userConfig.kafka_connect-property){: name='spec.userConfig.kafka_connect-property'} (object). Kafka Connect configuration values. See below for [nested schema](#spec.userConfig.kafka_connect).
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`secret_providers`](#spec.userConfig.secret_providers-property){: name='spec.userConfig.secret_providers-property'} (array of objects). Configure external secret providers in order to reference external secrets in connector configuration. Currently Hashicorp Vault (provider: vault, auth_method: token) and AWS Secrets Manager (provider: aws, auth_method: credentials) are supported. Secrets can be referenced in connector config with ${<provider_name>:<secret_path>:<key_name>}. See below for [nested schema](#spec.userConfig.secret_providers).
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### kafka_connect {: #spec.userConfig.kafka_connect }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Kafka Connect configuration values.

**Optional**

- [`connector_client_config_override_policy`](#spec.userConfig.kafka_connect.connector_client_config_override_policy-property){: name='spec.userConfig.kafka_connect.connector_client_config_override_policy-property'} (string, Enum: `All`, `None`). Defines what client configurations can be overridden by the connector. Default is None.
- [`consumer_auto_offset_reset`](#spec.userConfig.kafka_connect.consumer_auto_offset_reset-property){: name='spec.userConfig.kafka_connect.consumer_auto_offset_reset-property'} (string, Enum: `earliest`, `latest`). What to do when there is no initial offset in Kafka or if the current offset does not exist any more on the server. Default is earliest.
- [`consumer_fetch_max_bytes`](#spec.userConfig.kafka_connect.consumer_fetch_max_bytes-property){: name='spec.userConfig.kafka_connect.consumer_fetch_max_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). Records are fetched in batches by the consumer, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that the consumer can make progress. As such, this is not a absolute maximum.
- [`consumer_isolation_level`](#spec.userConfig.kafka_connect.consumer_isolation_level-property){: name='spec.userConfig.kafka_connect.consumer_isolation_level-property'} (string, Enum: `read_committed`, `read_uncommitted`). Transaction read isolation level. read_uncommitted is the default, but read_committed can be used if consume-exactly-once behavior is desired.
- [`consumer_max_partition_fetch_bytes`](#spec.userConfig.kafka_connect.consumer_max_partition_fetch_bytes-property){: name='spec.userConfig.kafka_connect.consumer_max_partition_fetch_bytes-property'} (integer, Minimum: 1048576, Maximum: 104857600). Records are fetched in batches by the consumer.If the first record batch in the first non-empty partition of the fetch is larger than this limit, the batch will still be returned to ensure that the consumer can make progress.
- [`consumer_max_poll_interval_ms`](#spec.userConfig.kafka_connect.consumer_max_poll_interval_ms-property){: name='spec.userConfig.kafka_connect.consumer_max_poll_interval_ms-property'} (integer, Minimum: 1, Maximum: 2147483647). The maximum delay in milliseconds between invocations of poll() when using consumer group management (defaults to 300000).
- [`consumer_max_poll_records`](#spec.userConfig.kafka_connect.consumer_max_poll_records-property){: name='spec.userConfig.kafka_connect.consumer_max_poll_records-property'} (integer, Minimum: 1, Maximum: 10000). The maximum number of records returned in a single call to poll() (defaults to 500).
- [`offset_flush_interval_ms`](#spec.userConfig.kafka_connect.offset_flush_interval_ms-property){: name='spec.userConfig.kafka_connect.offset_flush_interval_ms-property'} (integer, Minimum: 1, Maximum: 100000000). The interval at which to try committing offsets for tasks (defaults to 60000).
- [`offset_flush_timeout_ms`](#spec.userConfig.kafka_connect.offset_flush_timeout_ms-property){: name='spec.userConfig.kafka_connect.offset_flush_timeout_ms-property'} (integer, Minimum: 1, Maximum: 2147483647). Maximum number of milliseconds to wait for records to flush and partition offset data to be committed to offset storage before cancelling the process and restoring the offset data to be committed in a future attempt (defaults to 5000).
- [`producer_batch_size`](#spec.userConfig.kafka_connect.producer_batch_size-property){: name='spec.userConfig.kafka_connect.producer_batch_size-property'} (integer, Minimum: 0, Maximum: 5242880). This setting gives the upper bound of the batch size to be sent. If there are fewer than this many bytes accumulated for this partition, the producer will `linger` for the linger.ms time waiting for more records to show up. A batch size of zero will disable batching entirely (defaults to 16384).
- [`producer_buffer_memory`](#spec.userConfig.kafka_connect.producer_buffer_memory-property){: name='spec.userConfig.kafka_connect.producer_buffer_memory-property'} (integer, Minimum: 5242880, Maximum: 134217728). The total bytes of memory the producer can use to buffer records waiting to be sent to the broker (defaults to 33554432).
- [`producer_compression_type`](#spec.userConfig.kafka_connect.producer_compression_type-property){: name='spec.userConfig.kafka_connect.producer_compression_type-property'} (string, Enum: `gzip`, `lz4`, `none`, `snappy`, `zstd`). Specify the default compression type for producers. This configuration accepts the standard compression codecs (`gzip`, `snappy`, `lz4`, `zstd`). It additionally accepts `none` which is the default and equivalent to no compression.
- [`producer_linger_ms`](#spec.userConfig.kafka_connect.producer_linger_ms-property){: name='spec.userConfig.kafka_connect.producer_linger_ms-property'} (integer, Minimum: 0, Maximum: 5000). This setting gives the upper bound on the delay for batching: once there is batch.size worth of records for a partition it will be sent immediately regardless of this setting, however if there are fewer than this many bytes accumulated for this partition the producer will `linger` for the specified time waiting for more records to show up. Defaults to 0.
- [`producer_max_request_size`](#spec.userConfig.kafka_connect.producer_max_request_size-property){: name='spec.userConfig.kafka_connect.producer_max_request_size-property'} (integer, Minimum: 131072, Maximum: 67108864). This setting will limit the number of record batches the producer will send in a single request to avoid sending huge requests.
- [`scheduled_rebalance_max_delay_ms`](#spec.userConfig.kafka_connect.scheduled_rebalance_max_delay_ms-property){: name='spec.userConfig.kafka_connect.scheduled_rebalance_max_delay_ms-property'} (integer, Minimum: 0, Maximum: 600000). The maximum delay that is scheduled in order to wait for the return of one or more departed workers before rebalancing and reassigning their connectors and tasks to the group. During this period the connectors and tasks of the departed workers remain unassigned. Defaults to 5 minutes.
- [`session_timeout_ms`](#spec.userConfig.kafka_connect.session_timeout_ms-property){: name='spec.userConfig.kafka_connect.session_timeout_ms-property'} (integer, Minimum: 1, Maximum: 2147483647). The timeout in milliseconds used to detect failures when using Kafka’s group management facilities (defaults to 10000).

### private_access {: #spec.userConfig.private_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from private networks.

**Optional**

- [`kafka_connect`](#spec.userConfig.private_access.kafka_connect-property){: name='spec.userConfig.private_access.kafka_connect-property'} (boolean). Allow clients to connect to kafka_connect with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service components through Privatelink.

**Optional**

- [`jolokia`](#spec.userConfig.privatelink_access.jolokia-property){: name='spec.userConfig.privatelink_access.jolokia-property'} (boolean). Enable jolokia.
- [`kafka_connect`](#spec.userConfig.privatelink_access.kafka_connect-property){: name='spec.userConfig.privatelink_access.kafka_connect-property'} (boolean). Enable kafka_connect.
- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Optional**

- [`kafka_connect`](#spec.userConfig.public_access.kafka_connect-property){: name='spec.userConfig.public_access.kafka_connect-property'} (boolean). Allow clients to connect to kafka_connect from the public internet for service nodes that are in a project VPC or another type of private network.
- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.

### secret_providers {: #spec.userConfig.secret_providers }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Configure external secret providers in order to reference external secrets in connector configuration. Currently Hashicorp Vault and AWS Secrets Manager are supported.

**Required**

- [`name`](#spec.userConfig.secret_providers.name-property){: name='spec.userConfig.secret_providers.name-property'} (string). Name of the secret provider. Used to reference secrets in connector config.

**Optional**

- [`aws`](#spec.userConfig.secret_providers.aws-property){: name='spec.userConfig.secret_providers.aws-property'} (object). AWS secret provider configuration. See below for [nested schema](#spec.userConfig.secret_providers.aws).
- [`vault`](#spec.userConfig.secret_providers.vault-property){: name='spec.userConfig.secret_providers.vault-property'} (object). Vault secret provider configuration. See below for [nested schema](#spec.userConfig.secret_providers.vault).

#### aws {: #spec.userConfig.secret_providers.aws }

_Appears on [`spec.userConfig.secret_providers`](#spec.userConfig.secret_providers)._

AWS secret provider configuration.

**Required**

- [`auth_method`](#spec.userConfig.secret_providers.aws.auth_method-property){: name='spec.userConfig.secret_providers.aws.auth_method-property'} (string, Enum: `credentials`). Auth method of the vault secret provider.
- [`region`](#spec.userConfig.secret_providers.aws.region-property){: name='spec.userConfig.secret_providers.aws.region-property'} (string, MaxLength: 64). Region used to lookup secrets with AWS SecretManager.

**Optional**

- [`access_key`](#spec.userConfig.secret_providers.aws.access_key-property){: name='spec.userConfig.secret_providers.aws.access_key-property'} (string, MaxLength: 128). Access key used to authenticate with aws.
- [`secret_key`](#spec.userConfig.secret_providers.aws.secret_key-property){: name='spec.userConfig.secret_providers.aws.secret_key-property'} (string, MaxLength: 128). Secret key used to authenticate with aws.

#### vault {: #spec.userConfig.secret_providers.vault }

_Appears on [`spec.userConfig.secret_providers`](#spec.userConfig.secret_providers)._

Vault secret provider configuration.

**Required**

- [`address`](#spec.userConfig.secret_providers.vault.address-property){: name='spec.userConfig.secret_providers.vault.address-property'} (string, MinLength: 1, MaxLength: 65536). Address of the Vault server.
- [`auth_method`](#spec.userConfig.secret_providers.vault.auth_method-property){: name='spec.userConfig.secret_providers.vault.auth_method-property'} (string, Enum: `token`). Auth method of the vault secret provider.

**Optional**

- [`engine_version`](#spec.userConfig.secret_providers.vault.engine_version-property){: name='spec.userConfig.secret_providers.vault.engine_version-property'} (integer, Enum: `1`, `2`). KV Secrets Engine version of the Vault server instance.
- [`prefix_path_depth`](#spec.userConfig.secret_providers.vault.prefix_path_depth-property){: name='spec.userConfig.secret_providers.vault.prefix_path_depth-property'} (integer). Prefix path depth of the secrets Engine. Default is 1. If the secrets engine path has more than one segment it has to be increased to the number of segments.
- [`token`](#spec.userConfig.secret_providers.vault.token-property){: name='spec.userConfig.secret_providers.vault.token-property'} (string, MaxLength: 256). Token used to authenticate with vault and auth method `token`.

