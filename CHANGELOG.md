# Changelog

## [MAJOR.MINOR.PATCH] - YYYY-MM-DD

- Add kind: `ServiceIntegrationEndpoint`
- Fix `ServiceIntegration` deletion when instance has no id set
- Change `Kafka` field `userConfig.kafka_version`: enum ~~`[3.4, 3.5, 3.6]`~~ → `[3.4, 3.5, 3.6, 3.7]`
- Add `ServiceIntegration` `flink_external_postgresql` type
- Remove `REDIS_CA_CERT` secret key. Can't be used with the service type

## v0.19.0 - 2024-04-18

- Add kind: `ClickhouseRole`
- Unified User-Agent format with the Terraform Provider
- Unify cluster role permissions
- Add missing role permissions to `KafkaACL`

## v0.18.1 - 2024-04-02

- Add `KafkaSchemaRegistryACL` kind
- Add `ClickhouseDatabase` kind
- Fix secret creation for kinds with no secrets
- Include the Kubernetes version in the Go client's user agent
- Replace `Database` kind validations and default values with CRD validation rules
- Perform upgrade tasks to check if PG service can be upgraded before updating the service
- Expose project CA certificate to service secrets: `REDIS_CA_CERT`, `MYSQL_CA_CERT`, etc.
- Add `KafkaTopic` field `config.local_retention_bytes`, type `integer`: local.retention.bytes value
- Add `KafkaTopic` field `config.local_retention_ms`, type `integer`: local.retention.ms value
- Add `KafkaTopic` field `config.remote_storage_enable`, type `boolean`: remote_storage_enable
- Change `Cassandra` field `userConfig.cassandra_version`: pattern `^[0-9]+(\.[0-9]+)?$`
- Change `Cassandra` field `userConfig.project_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `Cassandra` field `userConfig.service_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `Cassandra` field `userConfig.service_to_join_with`: pattern `^[a-z][-a-z0-9]{0,63}$`
- Change `Clickhouse` field `userConfig.project_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `Clickhouse` field `userConfig.service_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `Grafana` field `userConfig.project_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `Grafana` field `userConfig.service_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `Kafka` field `userConfig.kafka.sasl_oauthbearer_expected_audience`: pattern `^[^\r\n]*$`
- Change `Kafka` field `userConfig.kafka.sasl_oauthbearer_expected_issuer`: pattern `^[^\r\n]*$`
- Change `Kafka` field `userConfig.kafka.sasl_oauthbearer_sub_claim_name`: pattern `^[^\r\n]*$`
- Change `MySQL` field `userConfig.mysql.default_time_zone`: pattern `^([-+][\d:]*|[\w/]*)$`
- Change `MySQL` field `userConfig.project_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `MySQL` field `userConfig.service_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `OpenSearch` field `userConfig.openid.client_id`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.openid.client_secret`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.openid.header`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.openid.jwt_header`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.openid.jwt_url_parameter`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.openid.roles_key`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.openid.scope`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.openid.subject_key`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.project_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `OpenSearch` field `userConfig.saml.idp_entity_id`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.saml.roles_key`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.saml.sp_entity_id`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.saml.subject_key`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.service_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `PostgreSQL` field `userConfig.pg.timezone`: pattern `^[\w/]*$`
- Change `PostgreSQL` field `userConfig.pg_service_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `PostgreSQL` field `userConfig.project_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `PostgreSQL` field `userConfig.service_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `Redis` field `userConfig.project_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Change `Redis` field `userConfig.service_to_fork_from`: pattern `^[a-z][-a-z0-9]{0,63}$|^$`
- Add `OpenSearch` field `userConfig.opensearch.plugins_alerting_filter_by_backend_roles`, type `boolean`:
  Enable or disable filtering of alerting by backend roles. Requires Security plugin
- Change `Redis` field `userConfig.redis_notify_keyspace_events`: pattern ~~`^[KEg\$lshzxeA]*$`~~ →
  `^[KEg\$lshzxentdmA]*$`
- Add `PostgreSQL` field `userConfig.pgaudit`, type `object`: System-wide settings for the pgaudit extension
- Add `ServiceIntegration` field `datadog.opensearch.cluster_stats_enabled`, type `boolean`: Enable Datadog
  Opensearch Cluster Monitoring

## v0.17.0 - 2024-02-01

- Bump k8s deps to 1.26.13
- Add `OpenSearch` field `userConfig.opensearch.enable_security_audit`, type `boolean`: Enable/Disable
  security audit
- Add `Kafka` field `userConfig.kafka_rest_config.name_strategy`, type `string`: Name strategy to use
  when selecting subject for storing schemas
- Add `Redis` field `userConfig.redis_version`, type `string`: Redis major version
- Add `Grafana` field `userConfig.auth_github.auto_login`, type `boolean`: Allow users to bypass the
  login screen and automatically log in
- Add `Grafana` field `userConfig.auth_github.skip_org_role_sync`, type `boolean`: Stop automatically
  syncing user roles
- Change `Clickhouse` field `userConfig.additional_backup_regions`: deprecated
- Change `Grafana` field `userConfig.additional_backup_regions`: deprecated
- Change `KafkaConnect` field `userConfig.additional_backup_regions`: deprecated
- Change `Kafka` field `userConfig.additional_backup_regions`: deprecated
- Change `OpenSearch` field `userConfig.additional_backup_regions`: deprecated
- Change `Redis` field `userConfig.additional_backup_regions`: deprecated
- Change `Cassandra` field `userConfig.cassandra_version`: enum ~~`[3, 4, 4.1]`~~ → `[4, 4.1]`
- Change `Kafka` field `userConfig.kafka_version`: enum ~~`[3.1, 3.3, 3.4, 3.5, 3.6]`~~ → `[3.4, 3.5, 3.6]`
- Change `PostgreSQL` field `userConfig.pg_version`: enum ~~`[11, 12, 13, 14, 15, 16]`~~ → `[12, 13,
14, 15, 16]`
- Add `Cassandra` field `technicalEmails`, type `array`: Defines the email addresses that will receive
  alerts about upcoming maintenance updates or warnings about service instability
- Add `Clickhouse` field `technicalEmails`, type `array`: Defines the email addresses that will receive
  alerts about upcoming maintenance updates or warnings about service instability
- Add `Grafana` field `technicalEmails`, type `array`: Defines the email addresses that will receive
  alerts about upcoming maintenance updates or warnings about service instability
- Add `KafkaConnect` field `technicalEmails`, type `array`: Defines the email addresses that will receive
  alerts about upcoming maintenance updates or warnings about service instability
- Add `Kafka` field `technicalEmails`, type `array`: Defines the email addresses that will receive alerts
  about upcoming maintenance updates or warnings about service instability
- Add `MySQL` field `technicalEmails`, type `array`: Defines the email addresses that will receive alerts
  about upcoming maintenance updates or warnings about service instability
- Add `OpenSearch` field `technicalEmails`, type `array`: Defines the email addresses that will receive
  alerts about upcoming maintenance updates or warnings about service instability
- Add `PostgreSQL` field `technicalEmails`, type `array`: Defines the email addresses that will receive
  alerts about upcoming maintenance updates or warnings about service instability
- Add `Redis` field `technicalEmails`, type `array`: Defines the email addresses that will receive alerts
  about upcoming maintenance updates or warnings about service instability
- Add `Cassandra` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `Clickhouse` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `ClickhouseUser` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `ConnectionPool` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `Grafana` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `Kafka` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `MySQL` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `OpenSearch` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `PostgreSQL` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `Project` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `Redis` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false
- Add `ServiceUser` field `connInfoSecretTargetDisabled`, type `boolean`: When true, the secret containing
  connection information will not be created, defaults to false

## v0.16.1 - 2023-12-15

- Check VPC for running services before deletion. Prevents VPC from hanging in the DELETING state
- Expose `KAFKA_SCHEMA_REGISTRY_URI` and `KAFKA_REST_URI` to `Kafka` secret
- Expose `CONNECTIONPOOL_NAME` in `ConnectionPool` secret
- Fix `CONNECTIONPOOL_PORT` exposes service port instead of pool port
- Fix `SERVICEUSER_PORT` when `sasl` is the only authentication method
- Change `PostgreSQL` field `userConfig.pg_qualstats.enabled`: deprecated
- Change `PostgreSQL` field `userConfig.pg_qualstats.min_err_estimate_num`: deprecated
- Change `PostgreSQL` field `userConfig.pg_qualstats.min_err_estimate_ratio`: deprecated
- Change `PostgreSQL` field `userConfig.pg_qualstats.track_constants`: deprecated
- Change `PostgreSQL` field `userConfig.pg_qualstats.track_pg_catalog`: deprecated

## v0.16.0 - 2023-12-07

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
- Remove `ip_filter` backward compatibility
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

- add Redis CRD

improvements:

- watch CRDs to reconcile token secrets

fixes:

- fix RBACs of KafkaACL CRD

## v0.1.1 - 2021-09-13

improvements:

- update helm installation docs

fixes:

- fix typo in a kafka-connector kuttl test

## v0.1.0 - 2021-09-10

features:

- initial release
