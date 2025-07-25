# Changelog


## v0.31.0 - 2025-07-25

- Upgraded HPA from deprecated `autoscaling/v2beta1` to stable `autoscaling/v2` API
- Added `ServiceUser` field `connInfoSecretSource`: Allows reading passwords from existing secrets for credential management. Supports setting passwords for new users and existing users
- Change `AlloyDBOmni` field `userConfig.pg.max_wal_senders`: maximum ~~`64`~~ → `256`
- Add `Kafka` field `userConfig.single_zone.availability_zone`, type `string`: The availability zone
  to use for the service. This is only used when enabled is set to true
- Change `PostgreSQL` field `userConfig.pg.max_wal_senders`: maximum ~~`64`~~ → `256`
- Add `ClickhouseUser` field `connInfoSecretSource`:  Allows reading passwords from existing secrets for credential management. Supports setting passwords for new users and existing users

## v0.30.0 - 2025-07-03

- Added `powered` field (default: `true`) to control service power state. When `false`, the service is powered off.
  Note: Kafka services without backups will lose topic data on power off. See field description for more information.
- Completely replace the old go client with the new one, which is generated from the OpenAPI spec
- Change `PostgreSQL` field `userConfig.pg_version`: enum remove `12`
- Add `KafkaTopic` field `config.inkless_enable`, type `boolean`: Indicates whether inkless should be enabled
- Add `KafkaTopic` field `config.unclean_leader_election_enable`, type `boolean`: Indicates whether to
  enable replicas not in the ISR set to be elected as leader as a last resort, even though doing so
  may result in data loss
- Refactor `KafkaTopic`: replace HTTP client with code-generated one to improve maintainability and type safety
- Add kind: `KafkaNativeACL`. Creates and manages Kafka-native access control lists (ACLs) for an Aiven for Apache Kafka® service.
- Add key `OPENSEARCH_URI` to `OpenSearch` service secrets: Contains the OpenSearch service URI.
- Change `KafkaSchema` fields `schemaType` and `subjectName` to be immutable since these fields cannot be modified after creation in the Kafka Schema Registry API
- Improve `KafkaSchema` controller: optimize polling and add better error handling
- Improve `KafkaTopic`: better handle API 5xx errors.
- Improve `KafkaConnector`: better handle API 404 and 5xx errors.
- Fix webhooks `containerPort` configuration not being properly applied in deployment template
- Change `AlloyDBOmni`, `Cassandra`, `Clickhouse`, `Flink`, `Grafana`, `KafkaConnect`, `Kafka`, `MySQL`, `OpenSearch`, `PostgreSQL`, `Redis`, `Valkey` field `userConfig.ip_filter`: maxItems ~~`2048`~~ → `8000`
- Add `Clickhouse` field `userConfig.enable_ipv6`, type `boolean`: Register AAAA DNS records for the
  service, and allow IPv6 packets to service ports
- Add `OpenSearch` field `userConfig.opensearch.cluster.filecache.remote_data_ratio`, type `number`:
  Defines a limit of how much total remote data can be referenced as a ratio of the size of the disk
  reserved for the file cache
- Add `OpenSearch` field `userConfig.opensearch.cluster.remote_store`, type `object`: no description
- Add `OpenSearch` field `userConfig.opensearch.enable_snapshot_api`, type `boolean`: Enable/Disable
  snapshot API for custom repositories, this requires security management to be enabled
- Add `OpenSearch` field `userConfig.opensearch.node.search.cache.size`, type `string`: Defines a limit
  of how much total remote data can be referenced as a ratio of the size of the disk reserved for
  the file cache
- Add `OpenSearch` field `userConfig.opensearch.remote_store`, type `object`: no description

## v0.29.0 - 2025-04-29

- Added retry logic to the `ServiceIntegration` controller
- Made `ConnectionPool` username field optional, allowing connection pools to use the credentials of the connecting client instead of a fixed service user
- Add `Kafka` field `userConfig.kafka_rest_config.consumer_idle_disconnect_timeout`, type `integer`:
  Specifies the maximum duration (in seconds) a client can remain idle before it is deleted
- Change `ServiceIntegration` field `clickhouseKafka.tables`: maxItems ~~`100`~~ → `400`
- Add `Valkey` field `userConfig.enable_ipv6`, type `boolean`: Register AAAA DNS records for the service,
  and allow IPv6 packets to service ports
- Add `Valkey` field `userConfig.valkey_active_expire_effort`, type `integer`: Valkey reclaims expired
  keys both when accessed and in the background
- Add `OpenSearch` field `userConfig.azure_migration.readonly`, type `boolean`: Whether the repository
  is read-only
- Add `OpenSearch` field `userConfig.gcs_migration.readonly`, type `boolean`: Whether the repository
  is read-only
- Add `OpenSearch` field `userConfig.opensearch.disk_watermarks`, type `object`: Watermark settings
- Add `OpenSearch` field `userConfig.s3_migration.readonly`, type `boolean`: Whether the repository is
  read-only
- Add `AlloyDBOmni` field `userConfig.pgaudit`, type `object`: System-wide settings for the pgaudit extension
- Add `Clickhouse` field `userConfig.backup_hour`, type `integer`: The hour of day (in UTC) when backup
  for the service is started
- Add `Clickhouse` field `userConfig.backup_minute`, type `integer`: The minute of an hour when backup
  for the service is started
- Add `Kafka` field `userConfig.kafka_connect_plugin_versions`, type `array`: The plugin selected by the user
- Change `Kafka` field `userConfig.kafka_version`: enum add `3.9`
- Add `OpenSearch` field `userConfig.opensearch.enable_searchable_snapshots`, type `boolean`: Enable
  searchable snapshots
- Change `PostgreSQL` field `userConfig.pgaudit.log_level`: enum add `debug1`, `debug2`, `debug3`, `debug4`,
  `debug5`, `info`, `log`, `notice`

## v0.28.0 - 2025-02-17

- Add kind: `AlloyDBOmni`
- Deprecate `Redis`: use `Valkey` instead. Please follow [these](https://aiven.io/docs/products/caching/howto/upgrade-aiven-for-caching-to-valkey#upgrade-service) instructions to upgrade your service to Valkey
- Deprecate `Cassandra`, see Aiven platform [end-of-life](https://aiven.io/docs/platform/reference/end-of-life) policy.
- Change `Cassandra` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Change `Clickhouse` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Change `Flink` field `userConfig.custom_code`: immutable `true`
- Change `Flink` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Add `Grafana` field `userConfig.dashboard_scenes_enabled`, type `boolean`: Enable use of the Grafana
  Scenes Library as the dashboard engine. i.e
- Change `Grafana` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Add `KafkaConnect` field `userConfig.plugin_versions`, type `array`: The plugin selected by the user
- Change `KafkaConnect` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Change `Kafka` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Change `MySQL` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Add `OpenSearch` field `userConfig.opensearch.cluster.search.request.slowlog`, type `object`
- Add `OpenSearch` field `userConfig.opensearch.enable_remote_backed_storage`, type `boolean`: Enable
  remote-backed storage
- Change `OpenSearch` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Change `PostgreSQL` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Change `Redis` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Change `ServiceIntegration` field `logs.elasticsearch_index_prefix`: pattern `^[a-z0-9][a-z0-9-_.]+$`
- Change `Valkey` field `userConfig.ip_filter`: maxItems ~~`1024`~~ → `2048`
- Add `Valkey` field `userConfig.frequent_snapshots`, type `boolean`: When enabled, Valkey will create
  frequent local RDB snapshots
- Change `OpenSearch` field `userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.allowed_tries`:
  maximum ~~`2147483647`~~ → `32767`
- Change `OpenSearch` field `userConfig.opensearch.auth_failure_listeners.ip_rate_limiting`: deprecated
- Add `Database` field `databaseName` type `string`: DatabaseName is the name of the database to be created.

## v0.27.0 - 2025-01-16

- Add `ServiceIntegrationEndpoint` field `datadog.extra_tags_prefix`, type `string`: Extra tags prefix.
  Defaults to aiven
- Change `Flink` field `userConfig.flink_version`: enum add `1.20`
- Add `OpenSearch` field `userConfig.opensearch_dashboards.multiple_data_source_enabled`, type `boolean`:
  Enable or disable multiple data sources in OpenSearch Dashboards
- Change `OpenSearch` field `userConfig.opensearch_dashboards.max_old_space_size`: maximum ~~`2048`~~
  → `4096`
- Change `PostgreSQL` field `userConfig.pg_version`: enum add `17`
- Add `PostgreSQL` field `userConfig.pg.password_encryption`, type `string`: Chooses the algorithm for
  encrypting passwords
- Add `OpenSearch` field `userConfig.opensearch.cluster.routing.allocation.balance.prefer_primary`, type
  `boolean`: When set to true, OpenSearch attempts to evenly distribute the primary shards between
  the cluster nodes
- Add `OpenSearch` field `userConfig.opensearch.segrep`, type `object`: Segment Replication Backpressure
  Settings
- Add `Flink` field `userConfig.custom_code`, type `boolean`: Enable to upload Custom JARs for Flink
  applications
- Add kind: `Valkey`

## v0.26.0 - 2024-11-21

- Add kind: `Flink`
- Add `Clickhouse` field `userConfig.recovery_basebackup_name`, type `string`: Name of the basebackup
  to restore in forked service
- Add `Grafana` field `userConfig.auth_generic_oauth.use_refresh_token`, type `boolean`: Set to true
  to use refresh token and check access token expiration
- Add `Kafka` field `userConfig.schema_registry_config.retriable_errors_silenced`, type `boolean`: If
  enabled, kafka errors which can be retried or custom errors specified for the service will not be
  raised, instead, a warning log is emitted
- Add `Kafka` field `userConfig.schema_registry_config.schema_reader_strict_mode`, type `boolean`: If
  enabled, causes the Karapace schema-registry service to shutdown when there are invalid schema records
  in the `_schemas` topic
- Add `Kafka` field `userConfig.single_zone`, type `object`: Single-zone configuration
- Change `Kafka` field `userConfig.kafka_version`: enum remove `3.5`, `3.6`
- Add `MySQL` field `userConfig.mysql.log_output`, type `string`: The slow log output destination when
  slow_query_log is ON
- Add `OpenSearch` field `userConfig.azure_migration.indices`, type `string`: A comma-delimited list
  of indices to restore from the snapshot. Multi-index syntax is supported
- Add `OpenSearch` field `userConfig.gcs_migration.indices`, type `string`: A comma-delimited list of
  indices to restore from the snapshot. Multi-index syntax is supported
- Add `OpenSearch` field `userConfig.s3_migration.indices`, type `string`: A comma-delimited list of
  indices to restore from the snapshot. Multi-index syntax is supported
- Change `PostgreSQL` field `userConfig.additional_backup_regions`: deprecated
- Add `OpenSearch` field `userConfig.azure_migration.restore_global_state`, type `boolean`: If true,
  restore the cluster state. Defaults to false
- Add `OpenSearch` field `userConfig.gcs_migration.restore_global_state`, type `boolean`: If true, restore
  the cluster state. Defaults to false
- Add `OpenSearch` field `userConfig.opensearch.search_backpressure`, type `object`: Search Backpressure
  Settings
- Add `OpenSearch` field `userConfig.opensearch.shard_indexing_pressure`, type `object`: Shard indexing
  back pressure settings
- Add `OpenSearch` field `userConfig.s3_migration.restore_global_state`, type `boolean`: If true, restore
  the cluster state. Defaults to false
- Change `Redis` field `userConfig.redis_timeout`: maximum ~~`31536000`~~ → `2073600`
- Add `OpenSearch` field `userConfig.azure_migration.include_aliases`, type `boolean`: Whether to restore
  aliases alongside their associated indexes. Default is true
- Add `OpenSearch` field `userConfig.gcs_migration.include_aliases`, type `boolean`: Whether to restore
  aliases alongside their associated indexes. Default is true
- Add `OpenSearch` field `userConfig.s3_migration.include_aliases`, type `boolean`: Whether to restore
  aliases alongside their associated indexes. Default is true
- Add `ServiceIntegration` field `autoscaler`, type `object`: Autoscaler specific user configuration options
- Add `ServiceIntegrationEndpoint` field `autoscaler`, type `object`: Autoscaler configuration values
- Change `Grafana` field `userConfig.alerting_enabled`: deprecated
- Change `OpenSearch` field `userConfig.opensearch.auth_failure_listeners.internal_authentication_backend_limiting.allowed_tries`:
  minimum ~~`0`~~ → `1`
- Change `OpenSearch` field `userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.block_expiry_seconds`:
  minimum ~~`1`~~ → `0`
- Change `OpenSearch` field `userConfig.opensearch.auth_failure_listeners.ip_rate_limiting.time_window_seconds`:
  minimum ~~`1`~~ → `0`
- Change `Cassandra` field `userConfig.cassandra_version`: enum remove `4`
- Change `PostgreSQL` field `userConfig.pg_version`: enum remove `12`
- Add `OpenSearch` field `userConfig.opensearch.search.insights.top_queries`, type `object`

## v0.25.0 - 2024-09-19

- Fix `KafkaTopic`: fails to create a topic with the replication factor set more than running Kafka nodes
- Fix `ServiceIntegration`: sends empty source and destination projects
- Fix `KafkaSchema`: poll resource availability
- Add `KafkaSchema` field `schemaType`, type `string`: Schema type
- Add `Kafka` field `userConfig.follower_fetching`, type `object`: Enable follower fetching
- Add `Kafka` field `userConfig.kafka_sasl_mechanisms`, type `object`: Kafka SASL mechanisms
- Change `Kafka` field `userConfig.kafka.sasl_oauthbearer_sub_claim_name`: pattern ~~`^[^\r\n]*$`~~ →
  `^[^\r\n]*\S[^\r\n]*$`
- Add `MySQL` field `userConfig.migration.ignore_roles`, type `string`: Comma-separated list of database
  roles, which should be ignored during migration (supported by PostgreSQL only at the moment)
- Add `PostgreSQL` field `userConfig.migration.ignore_roles`, type `string`: Comma-separated list of
  database roles, which should be ignored during migration (supported by PostgreSQL only at the moment)
- Add `PostgreSQL` field `userConfig.pgbouncer.max_prepared_statements`, type `integer`: PgBouncer tracks
  protocol-level named prepared statements related commands sent by the client in transaction and
  statement pooling modes when max_prepared_statements is set to a non-zero value
- Add `Redis` field `userConfig.migration.ignore_roles`, type `string`: Comma-separated list of database
  roles, which should be ignored during migration (supported by PostgreSQL only at the moment)
- Add `Redis` field `userConfig.backup_hour`, type `integer`: The hour of day (in UTC) when backup for
  the service is started
- Add `Redis` field `userConfig.backup_minute`, type `integer`: The minute of an hour when backup for
  the service is started
- Add `Grafana` field `userConfig.wal`, type `boolean`: Setting to enable/disable Write-Ahead Logging.
  The default value is false (disabled)
- Add `OpenSearch` field `userConfig.azure_migration`, type `object`: Azure migration settings
- Add `OpenSearch` field `userConfig.gcs_migration`, type `object`: Google Cloud Storage migration settings
- Add `OpenSearch` field `userConfig.index_rollup`, type `object`: Index rollup settings
- Add `OpenSearch` field `userConfig.s3_migration`, type `object`: AWS S3 / AWS S3 compatible migration settings
- Change `OpenSearch` field `userConfig.openid.connect_url`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.opensearch.script_max_compilations_rate`: pattern `^[^\r\n]*$`
- Change `OpenSearch` field `userConfig.saml.idp_metadata_url`: pattern `^[^\r\n]*$`

## v0.24.0 - 2024-07-16

- Fix `PostgreSQL`: wait for a valid backup to create read replica
- Fix `ClickhouseGrant`: grant privileges for an unknown table (Clickhouse can do that)
- Fix `ClickhouseGrant`: track the state to revoke only known privileges
- Add `Cassandra` field `userConfig.cassandra.read_request_timeout_in_ms`, type `integer`: How long the
  coordinator waits for read operations to complete before timing it out
- Add `Cassandra` field `userConfig.cassandra.write_request_timeout_in_ms`, type `integer`: How long
  the coordinator waits for write requests to complete with at least one node in the local datacenter
- Add `OpenSearch` field `userConfig.opensearch.knn_memory_circuit_breaker_enabled`, type `boolean`:
  Enable or disable KNN memory circuit breaker. Defaults to true
- Add `OpenSearch` field `userConfig.opensearch.knn_memory_circuit_breaker_limit`, type `integer`: Maximum
  amount of memory that can be used for KNN index. Defaults to 50% of the JVM heap size
- Change `PostgreSQL` field `userConfig.pg.log_line_prefix`: enum ~~`['%m [%p] %q[user=%u,db=%d,app=%a]
', '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h ', 'pid=%p,user=%u,db=%d,app=%a,client=%h ']`~~
  → `['%m [%p] %q[user=%u,db=%d,app=%a] ', '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h ',
'pid=%p,user=%u,db=%d,app=%a,client=%h ', 'pid=%p,user=%u,db=%d,app=%a,client=%h,txid=%x,qid=%Q ']`

## v0.23.0 - 2024-07-12

- Ignore `http.StatusBadRequest` on `ClickhouseGrant` deletion
- Retry conflict error when k8s object saved to the storage
- Fix `ClickhouseGrant` invalid remote and local privileges comparison
- Fix `ClickhouseGrant`: doesn't escape role name to grant
- Fix `ClickhouseUser`: password was reset due to an incorrect processing cycle

## v0.22.0 - 2024-07-02

- Ignore `ClickhouseRole` deletion error (missing database)
- Ignore `ClickhouseGrant` deletion errors (missing database, service, role)
- Do not block service operations in `REBALANCING` state

## v0.21.0 - 2024-06-25

- Add kind: `ClickhouseGrant`
- Add `KafkaConnect` field `userConfig.secret_providers`, type `array`: Configure external secret providers
  in order to reference external secrets in connector configuration
- Add `Kafka` field `userConfig.kafka_connect_secret_providers`, type `array`: Configure external secret
  providers in order to reference external secrets in connector configuration
- Add `Kafka` field `userConfig.letsencrypt_sasl_privatelink`, type `boolean`: Use Letsencrypt CA for
  Kafka SASL via Privatelink
- Add `ServiceIntegration` field `datadog.mirrormaker_custom_metrics`, type `array`: List of custom metrics
- Add `ServiceIntegration` field `kafkaMirrormaker.kafka_mirrormaker.consumer_auto_offset_reset`, type
  `string`: Set where consumer starts to consume data
- Add `ServiceIntegration` field `kafkaMirrormaker.kafka_mirrormaker.consumer_max_poll_records`, type
  `integer`: Set consumer max.poll.records. The default is 500
- Change `PostgreSQL` field `userConfig.pgaudit`: deprecated
- Breaking change `ServiceIntegrationEndpoint` field `externalPostgresql.ssl_mode`: enum ~~`[allow, disable, prefer,
require, verify-ca, verify-full]`~~ → `[require, verify-ca, verify-full]`

## v0.20.0 - 2024-06-05

- Add kind: `ServiceIntegrationEndpoint`
- Add `ServiceIntegration` `flink_external_postgresql` type
- Add `ServiceIntegration` field `datadog.datadog_pgbouncer_enabled`, type `boolean`: Enable Datadog
  PgBouncer Metric Tracking
- Fix `ServiceIntegration` deletion when instance has no id set
- Fix service types `disk_space` field validation
- Fix resources `project`, `serviceName` fields validation
- Fix `ConnectionPool` doesn't check service user precondition
- Remove `CA_CERT` secret key for `Grafana`, `OpenSearch`, `Redis`, and `Clickhouse`. Can't be used with these service types
  ddog-gov.com, us3.datadoghq.com, us5.datadoghq.com]`
- Change `ServiceIntegrationEndpoint` field `externalKafka.ssl_endpoint_identification_algorithm`: enum
  ~~`[, https]`~~ → `[https]`
- Remove `ClickhouseUser` webhook. Doesn't do any validation or mutation
- Change `Kafka` field `userConfig.kafka_version`: enum ~~`[3.4, 3.5, 3.6]`~~ → `[3.4, 3.5, 3.6, 3.7]`
- Change `ServiceIntegrationEndpoint` field `datadog.site`: enum ~~`[datadoghq.com, datadoghq.eu, ddog-gov.com,
us3.datadoghq.com, us5.datadoghq.com]`~~ → `[ap1.datadoghq.com, datadoghq.com, datadoghq.eu,
- Move immutable fields validation from webhooks to CRD validation rules

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
