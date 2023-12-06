# Changelog

## [MAJOR.MINOR.PATCH] - YYYY-MM-DD

- Set conditions on errors: `Preconditions`, `CreateOrUpdate`, `Delete`. Thanks to @atarax
- Fix object updates lost when reconciler exits before the object is committed  
- Add `Kafka` field `userConfig.kafka.transaction_partition_verification_enable`, type `boolean`: Enable
  verification that checks that the partition has been added to the transaction before writing transactional
  records to the partition
- Add `Cassandra` field `userConfig.service_log`, type `boolean`: Store logs for the service so that
  they are available in the HTTP API and console
- Add `Clickhouse` field `userConfig.service_log`, type `boolean`: Store logs for the service so that
  they are available in the HTTP API and console
- Add `Grafana` field `userConfig.service_log`, type `boolean`: Store logs for the service so that they
  are available in the HTTP API and console
- Add `KafkaConnect` field `userConfig.service_log`, type `boolean`: Store logs for the service so that
  they are available in the HTTP API and console
- Add `Kafka` field `userConfig.kafka_rest_config.name_strategy_validation`, type `boolean`: If true,
  validate that given schema is registered under expected subject name by the used name strategy when
  producing messages
- Add `Kafka` field `userConfig.service_log`, type `boolean`: Store logs for the service so that they
  are available in the HTTP API and console
- Add `MySQL` field `userConfig.service_log`, type `boolean`: Store logs for the service so that they
  are available in the HTTP API and console
- Add `OpenSearch` field `userConfig.service_log`, type `boolean`: Store logs for the service so that
  they are available in the HTTP API and console
- Add `PostgreSQL` field `userConfig.pg_qualstats`, type `object`: System-wide settings for the pg_qualstats
  extension
- Add `PostgreSQL` field `userConfig.service_log`, type `boolean`: Store logs for the service so that
  they are available in the HTTP API and console
- Add `Redis` field `userConfig.service_log`, type `boolean`: Store logs for the service so that they
  are available in the HTTP API and console

## v0.15.0 - 2023-11-17

- Upgrade to Go 1.21
- Add option to orphan resources. Thanks to @atarax
- Fix `ServiceIntegration`: do not send empty user config to the API 
- Add a format for `string` type fields to the documentation
- Generate CRDs changelog
- Add `Clickhouse` field `userConfig.private_access.clickhouse_mysql`, type `boolean`: Allow clients
  to connect to clickhouse_mysql with a DNS name that always resolves to the service's private IP addresses
- Add `Clickhouse` field `userConfig.privatelink_access.clickhouse_mysql`, type `boolean`: Enable clickhouse_mysql
- Add `Clickhouse` field `userConfig.public_access.clickhouse_mysql`, type `boolean`: Allow clients to
  connect to clickhouse_mysql from the public internet for service nodes that are in a project VPC
  or another type of private network
- Add `Grafana` field `userConfig.unified_alerting_enabled`, type `boolean`: Enable or disable Grafana
  unified alerting functionality
- Add `Kafka` field `userConfig.aiven_kafka_topic_messages`, type `boolean`: Allow access to read Kafka
  topic messages in the Aiven Console and REST API
- Add `Kafka` field `userConfig.kafka.sasl_oauthbearer_expected_audience`, type `string`: The (optional)
  comma-delimited setting for the broker to use to verify that the JWT was issued for one of the
  expected audiences
- Add `Kafka` field `userConfig.kafka.sasl_oauthbearer_expected_issuer`, type `string`: Optional setting
  for the broker to use to verify that the JWT was created by the expected issuer
- Add `Kafka` field `userConfig.kafka.sasl_oauthbearer_jwks_endpoint_url`, type `string`: OIDC JWKS endpoint
  URL. By setting this the SASL SSL OAuth2/OIDC authentication is enabled
- Add `Kafka` field `userConfig.kafka.sasl_oauthbearer_sub_claim_name`, type `string`: Name of the scope
  from which to extract the subject claim from the JWT. Defaults to sub
- Change `Kafka` field `userConfig.kafka_version`: enum ~~`[3.1, 3.3, 3.4, 3.5]`~~ → `[3.1, 3.3, 3.4,
  3.5, 3.6]`
- Change `Kafka` field `userConfig.tiered_storage.local_cache.size`: deprecated
- Add `OpenSearch` field `userConfig.opensearch.indices_memory_max_index_buffer_size`, type `integer`:
  Absolute value. Default is unbound. Doesn't work without indices.memory.index_buffer_size
- Add `OpenSearch` field `userConfig.opensearch.indices_memory_min_index_buffer_size`, type `integer`:
  Absolute value. Default is 48mb. Doesn't work without indices.memory.index_buffer_size
- Change `OpenSearch` field `userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.authentication_backend`:
  enum `[internal]`
- Change `OpenSearch` field `userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.type`:
  enum `[username]`
- Change `OpenSearch` field `userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.type`: enum `[ip]`
- Change `OpenSearch` field `userConfig.opensearch.search_max_buckets`: maximum ~~`65536`~~ → `1000000`
- Change `ServiceIntegration` field `kafkaMirrormaker.kafka_mirrormaker.producer_max_request_size`: maximum
  ~~`67108864`~~ → `268435456`

## v0.14.0 - 2023-09-21

- Make `projectVpcId` and `projectVPCRef` mutable
- Fix panic on `nil` user config conversion
- Use aiven-go-client with context support
- Deprecate `Cassandra` kind option `additional_backup_regions`
- Add `Grafana` kind option `auto_login`
- Add `Kafka` kind properties `log_local_retention_bytes`, `log_local_retention_ms`
- Remove `Kafka` kind option `remote_log_storage_system_enable`
- Add `OpenSearch` kind option `auth_failure_listeners`
- Add `OpenSearch` kind [Index State Management](https://opensearch.org/docs/latest/im-plugin/ism/index/) options

## v0.13.0 - 2023-08-18

- Add [TieredStorage](https://github.com/Aiven-Open/tiered-storage-for-apache-kafka) support to `Kafka`
- Add `Kafka` version `3.5`
- Add `Kafka` spec property `scheduled_rebalance_max_delay_ms`
- Mark deprecated `Kafka` spec property `remote_log_storage_system_enable`
- Add `KafkaConnect` spec property `scheduled_rebalance_max_delay_ms`
- Add `OpenSearch` spec property `openid` 
- Use updated go client with enhanced retries

## v0.12.3 - 2023-07-13

- Expose `KAFKA_SCHEMA_REGISTRY_HOST` and `KAFKA_SCHEMA_REGISTRY_PORT` for `Kafka`
- Expose `KAFKA_CONNECT_HOST`, `KAFKA_CONNECT_PORT`, `KAFKA_REST_HOST` and `KAFKA_REST_PORT` for `Kafka`. Thanks to @Dariusch

## v0.12.2 - 2023-06-20

- Make conditions and state optional attributes of service status. Thanks to @mortenlj
- Remove deprecated `unclean_leader_election_enable` from `KafkaTopic` kind config
- Expose `KAFKA_SASL_PORT` for `Kafka` kind if `SASL` authentication method is enabled
- Add `redis` options to datadog `ServiceIntegration`
- Add `Cassandra` version `3`
- Add `Kafka` versions `3.1` and `3.4`
- Add `kafka_rest_config.producer_max_request_size` option
- Add `kafka_mirrormaker.producer_compression_type` option

## v0.12.0 - 2023-05-10

- Fix service tags create/update. Thanks to @mortenlj
- Add prefix name option for secrets. Thanks to @jordiclariana
- Add `clusterRole.create` option to Helm chart. Thanks to @ryaneorth
- Use kind name as default prefix for secrets to avoid collisions. Please migrate your applications before legacy names removed
- Fix secrets creation on openshift
- Add `OpenSearch.spec.userConfig.idp_pemtrustedcas_content` option.
  Specifies the PEM-encoded root certificate authority (CA) content for the SAML identity provider (IdP) server verification.


## v0.11.0 - 2023-04-25

- Add `ServiceIntegration` kind `SourceProjectName` and `DestinationProjectName` fields
- Add `ServiceIntegration` fields `MaxLength` validation
- Add `ServiceIntegration` validation: multiple user configs cannot be set
- Fix `ServiceIntegration`, should not require `destinationServiceName` or `sourceEndpointID` field
- Fix `ServiceIntegration`, add missing `external_aws_cloudwatch_metrics` type config serialization
- Update `ServiceIntegration` integration type list
- Add `annotations` and `labels` fields to `connInfoSecretTarget`
- Allow to disable capabilities check to install webhooks. Thanks to @amstee
- Set `OpenSearch.spec.userConfig.opensearch.search_max_buckets` maximum to `65536`

## v0.10.0 - 2023-04-17

- Mark service `plan` as a required field
- Add `minumim`, `maximum` validations for `number` type
- Move helm charts to the operator repository
- Add helm charts generator
- Remove `ip_filter` backward compatability
- Fix deletion errors omitted
- Add service integration `clickhouseKafka.tables.data_format-property` enum `RawBLOB` value
- Update OpenSearch `userConfig.opensearch.email_sender_username` validation pattern
- Add Kafka `log_cleaner_min_cleanable_ratio` minimum and maximum validation rules
- Remove Kafka version `3.2`, reached EOL
- Remove PostgreSQL version `10`, reached EOL
- Explicitly delete `ProjectVPC` by `ID` to avoid conflicts 
- Speed up `ProjectVPC` deletion by exiting on `DELETING` status
- Fix missing RBAC permissions to update finalizers for various controllers 
- Refactor `ClickhouseUser` controller
- Mark `ClickhouseUser.spec.project` and `ClickhouseUser.spec.serviceName` as immutable
- Remove deprecated service integration type `signalfx`
- Add build version to the Aiven client user-agent

## v0.9.0 - 2023-03-03

- `AuthSecretRef` fields marked as required
- Generate user configs for existing service integrations: `datadog`, `kafka_connect`, `kafka_logs`, `metrics`
- Add new service integrations: `clickhouse_postgresql`, `clickhouse_kafka`, `clickhouse_kafka`, `logs`, `external_aws_cloudwatch_metrics`
- Add `KafkaTopic.Spec.topicName` field. Unlike the `metadata.name`, supports additional characters and has a longer length.
  `KafkaTopic.Spec.topicName` replaces `metadata.name` in future releases and will be marked as required.
- Accept `false` value for `termination_protection` property
- Fix `min_cleanable_dirty_ratio`. Thanks to @TV2rd

## v0.8.0 - 2023-02-15

**Important:** This release brings breaking changes to the `userConfig` property.
After new charts are installed, update your existing instances manually using the `kubectl edit` command
according to the [API reference](https://aiven.github.io/aiven-operator/api-reference/).

**Note:** It is now recommended to disable webhooks for Kubernetes version 1.25 and higher,
as native [CRD validation rules](https://kubernetes.io/blog/2022/09/23/crd-validation-rules-beta/) are used.

- **Breaking change:** `ip_filter` field is now of `object` type
- **Breaking change:** Update user configs for following kinds: PostgreSQL, Kafka, KafkaConnect, Redis, Clickhouse, OpenSearch
- Add CRD validation rules for immutable fields
- Add user config field validations (enum, minimum, maximum, minLength, and others)
- Add `serviceIntegrations` on service types. Only the `read_replica` type is available.
- Add KafkaTopic `min_cleanable_dirty_ratio` config field support
- Add Clickhouse `spec.disk_space` property
- Use updated aiven-go-client with retries
- Add `linux/amd64` build. Thanks to @christoffer-eide

## v0.7.1 - 2023-01-24

- Add Cassandra Kind
- Add Grafana Kind
- Recreate Kafka ACL if modified. 
  Note: Modification of ACL created prior to v0.5.1 won't delete existing instance at Aiven.
  It must be deleted manually.
- Fix MySQL webhook

## v0.6.0 - 2023-01-16

- Remove `never` from choices of maintenance dow
- Add `development` flag to configure logger's behavior
- Add user config generator (see `make generate-user-configs`)
- Add `genericServiceHandler` to generalize service management 
- Add MySQL Kind

## v0.5.2 - 2022-12-09

- Fix deployment release manifest generation

## v0.5.1 - 2022-11-28

- Fix `KafkaACL` deletion

## v0.5.0 - 2022-11-27

- Add ability to link resources through the references
- Add `ProjectVPCRef` property to `Kafka`, `OpenSearch`, `Clickhouse` and `Redis` kinds
  to get `ProjectVPC` ID when resource is ready
- Improve `ProjectVPC` deletion, deletes by ID first if possible, then tries by name
- Fix `client.Object` storage update data loss

## v0.4.0 - 2022-08-04

- Upgrade to Go 1.18
- Add support for connection pull incoming user
- Fix typo on config/samples/kafka disk_space
- Add tags support for project and service resources
- Enable termination protection

## v0.2.0 - 2021-11-17

features:
* add Redis CRD

improvements:
* watch CRDs to reconcile token secrets

fixes:
* fix RBACs of KafkaACL CRD

## v0.1.1 - 2021-09-13

improvements:
* update helm installation docs

fixes:
* fix typo in a kafka-connector kuttl test

## v0.1.0 - 2021-09-10

features:
* initial release
