---
title: "ServiceIntegration"
---

## Usage example

```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: my-service-integration
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: aiven-project-name

  integrationType: kafka_logs
  sourceServiceName: my-source-service-name
  destinationServiceName: my-destination-service-name

  kafkaLogs:
    kafka_topic: my-kafka-topic
```

## ServiceIntegration {: #ServiceIntegration }

ServiceIntegration is the Schema for the serviceintegrations API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ServiceIntegration`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ServiceIntegrationSpec defines the desired state of ServiceIntegration. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ServiceIntegration`](#ServiceIntegration)._

ServiceIntegrationSpec defines the desired state of ServiceIntegration.

**Required**

- [`integrationType`](#spec.integrationType-property){: name='spec.integrationType-property'} (string, Enum: `alertmanager`, `autoscaler`, `caching`, `cassandra_cross_service_cluster`, `clickhouse_kafka`, `clickhouse_postgresql`, `dashboard`, `datadog`, `datasource`, `external_aws_cloudwatch_logs`, `external_aws_cloudwatch_metrics`, `external_elasticsearch_logs`, `external_google_cloud_logging`, `external_opensearch_logs`, `flink`, `flink_external_kafka`, `internal_connectivity`, `jolokia`, `kafka_connect`, `kafka_logs`, `kafka_mirrormaker`, `logs`, `m3aggregator`, `m3coordinator`, `metrics`, `opensearch_cross_cluster_replication`, `opensearch_cross_cluster_search`, `prometheus`, `read_replica`, `rsyslog`, `schema_registry_proxy`, `stresstester`, `thanosquery`, `thanosstore`, `vmalert`, Immutable). Type of the service integration accepted by Aiven API. Some values may not be supported by the operator.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, MaxLength: 63). Project the integration belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`clickhouseKafka`](#spec.clickhouseKafka-property){: name='spec.clickhouseKafka-property'} (object). Clickhouse Kafka configuration values. See below for [nested schema](#spec.clickhouseKafka).
- [`clickhousePostgresql`](#spec.clickhousePostgresql-property){: name='spec.clickhousePostgresql-property'} (object). Clickhouse PostgreSQL configuration values. See below for [nested schema](#spec.clickhousePostgresql).
- [`datadog`](#spec.datadog-property){: name='spec.datadog-property'} (object). Datadog specific user configuration options. See below for [nested schema](#spec.datadog).
- [`destinationEndpointId`](#spec.destinationEndpointId-property){: name='spec.destinationEndpointId-property'} (string, Immutable, MaxLength: 36). Destination endpoint for the integration (if any).
- [`destinationProjectName`](#spec.destinationProjectName-property){: name='spec.destinationProjectName-property'} (string, Immutable, MaxLength: 63). Destination project for the integration (if any).
- [`destinationServiceName`](#spec.destinationServiceName-property){: name='spec.destinationServiceName-property'} (string, Immutable, MaxLength: 64). Destination service for the integration (if any).
- [`externalAWSCloudwatchMetrics`](#spec.externalAWSCloudwatchMetrics-property){: name='spec.externalAWSCloudwatchMetrics-property'} (object). External AWS CloudWatch Metrics integration Logs configuration values. See below for [nested schema](#spec.externalAWSCloudwatchMetrics).
- [`kafkaConnect`](#spec.kafkaConnect-property){: name='spec.kafkaConnect-property'} (object). Kafka Connect service configuration values. See below for [nested schema](#spec.kafkaConnect).
- [`kafkaLogs`](#spec.kafkaLogs-property){: name='spec.kafkaLogs-property'} (object). Kafka logs configuration values. See below for [nested schema](#spec.kafkaLogs).
- [`kafkaMirrormaker`](#spec.kafkaMirrormaker-property){: name='spec.kafkaMirrormaker-property'} (object). Kafka MirrorMaker configuration values. See below for [nested schema](#spec.kafkaMirrormaker).
- [`logs`](#spec.logs-property){: name='spec.logs-property'} (object). Logs configuration values. See below for [nested schema](#spec.logs).
- [`metrics`](#spec.metrics-property){: name='spec.metrics-property'} (object). Metrics configuration values. See below for [nested schema](#spec.metrics).
- [`sourceEndpointID`](#spec.sourceEndpointID-property){: name='spec.sourceEndpointID-property'} (string, Immutable, MaxLength: 36). Source endpoint for the integration (if any).
- [`sourceProjectName`](#spec.sourceProjectName-property){: name='spec.sourceProjectName-property'} (string, Immutable, MaxLength: 63). Source project for the integration (if any).
- [`sourceServiceName`](#spec.sourceServiceName-property){: name='spec.sourceServiceName-property'} (string, Immutable, MaxLength: 64). Source service for the integration (if any).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

## clickhouseKafka {: #spec.clickhouseKafka }

_Appears on [`spec`](#spec)._

Clickhouse Kafka configuration values.

**Required**

- [`tables`](#spec.clickhouseKafka.tables-property){: name='spec.clickhouseKafka.tables-property'} (array of objects, MaxItems: 100). Tables to create. See below for [nested schema](#spec.clickhouseKafka.tables).

### tables {: #spec.clickhouseKafka.tables }

_Appears on [`spec.clickhouseKafka`](#spec.clickhouseKafka)._

Tables to create.

**Required**

- [`columns`](#spec.clickhouseKafka.tables.columns-property){: name='spec.clickhouseKafka.tables.columns-property'} (array of objects, MaxItems: 100). Table columns. See below for [nested schema](#spec.clickhouseKafka.tables.columns).
- [`data_format`](#spec.clickhouseKafka.tables.data_format-property){: name='spec.clickhouseKafka.tables.data_format-property'} (string, Enum: `Avro`, `CSV`, `JSONAsString`, `JSONCompactEachRow`, `JSONCompactStringsEachRow`, `JSONEachRow`, `JSONStringsEachRow`, `MsgPack`, `TSKV`, `TSV`, `TabSeparated`, `RawBLOB`). Message data format.
- [`group_name`](#spec.clickhouseKafka.tables.group_name-property){: name='spec.clickhouseKafka.tables.group_name-property'} (string, MinLength: 1, MaxLength: 249). Kafka consumers group.
- [`name`](#spec.clickhouseKafka.tables.name-property){: name='spec.clickhouseKafka.tables.name-property'} (string, MinLength: 1, MaxLength: 40). Name of the table.
- [`topics`](#spec.clickhouseKafka.tables.topics-property){: name='spec.clickhouseKafka.tables.topics-property'} (array of objects, MaxItems: 100). Kafka topics. See below for [nested schema](#spec.clickhouseKafka.tables.topics).

#### columns {: #spec.clickhouseKafka.tables.columns }

_Appears on [`spec.clickhouseKafka.tables`](#spec.clickhouseKafka.tables)._

Table columns.

**Required**

- [`name`](#spec.clickhouseKafka.tables.columns.name-property){: name='spec.clickhouseKafka.tables.columns.name-property'} (string, MinLength: 1, MaxLength: 40). Column name.
- [`type`](#spec.clickhouseKafka.tables.columns.type-property){: name='spec.clickhouseKafka.tables.columns.type-property'} (string, MinLength: 1, MaxLength: 1000). Column type.

#### topics {: #spec.clickhouseKafka.tables.topics }

_Appears on [`spec.clickhouseKafka.tables`](#spec.clickhouseKafka.tables)._

Kafka topics.

**Required**

- [`name`](#spec.clickhouseKafka.tables.topics.name-property){: name='spec.clickhouseKafka.tables.topics.name-property'} (string, MinLength: 1, MaxLength: 249). Name of the topic.

## clickhousePostgresql {: #spec.clickhousePostgresql }

_Appears on [`spec`](#spec)._

Clickhouse PostgreSQL configuration values.

**Required**

- [`databases`](#spec.clickhousePostgresql.databases-property){: name='spec.clickhousePostgresql.databases-property'} (array of objects, MaxItems: 10). Databases to expose. See below for [nested schema](#spec.clickhousePostgresql.databases).

### databases {: #spec.clickhousePostgresql.databases }

_Appears on [`spec.clickhousePostgresql`](#spec.clickhousePostgresql)._

Databases to expose.

**Optional**

- [`database`](#spec.clickhousePostgresql.databases.database-property){: name='spec.clickhousePostgresql.databases.database-property'} (string, MinLength: 1, MaxLength: 63). PostgreSQL database to expose.
- [`schema`](#spec.clickhousePostgresql.databases.schema-property){: name='spec.clickhousePostgresql.databases.schema-property'} (string, MinLength: 1, MaxLength: 63). PostgreSQL schema to expose.

## datadog {: #spec.datadog }

_Appears on [`spec`](#spec)._

Datadog specific user configuration options.

**Optional**

- [`datadog_dbm_enabled`](#spec.datadog.datadog_dbm_enabled-property){: name='spec.datadog.datadog_dbm_enabled-property'} (boolean). Enable Datadog Database Monitoring.
- [`datadog_tags`](#spec.datadog.datadog_tags-property){: name='spec.datadog.datadog_tags-property'} (array of objects, MaxItems: 32). Custom tags provided by user. See below for [nested schema](#spec.datadog.datadog_tags).
- [`exclude_consumer_groups`](#spec.datadog.exclude_consumer_groups-property){: name='spec.datadog.exclude_consumer_groups-property'} (array of strings, MaxItems: 1024). List of custom metrics.
- [`exclude_topics`](#spec.datadog.exclude_topics-property){: name='spec.datadog.exclude_topics-property'} (array of strings, MaxItems: 1024). List of topics to exclude.
- [`include_consumer_groups`](#spec.datadog.include_consumer_groups-property){: name='spec.datadog.include_consumer_groups-property'} (array of strings, MaxItems: 1024). List of custom metrics.
- [`include_topics`](#spec.datadog.include_topics-property){: name='spec.datadog.include_topics-property'} (array of strings, MaxItems: 1024). List of topics to include.
- [`kafka_custom_metrics`](#spec.datadog.kafka_custom_metrics-property){: name='spec.datadog.kafka_custom_metrics-property'} (array of strings, MaxItems: 1024). List of custom metrics.
- [`max_jmx_metrics`](#spec.datadog.max_jmx_metrics-property){: name='spec.datadog.max_jmx_metrics-property'} (integer, Minimum: 10, Maximum: 100000). Maximum number of JMX metrics to send.
- [`opensearch`](#spec.datadog.opensearch-property){: name='spec.datadog.opensearch-property'} (object). Datadog Opensearch Options. See below for [nested schema](#spec.datadog.opensearch).

### datadog_tags {: #spec.datadog.datadog_tags }

_Appears on [`spec.datadog`](#spec.datadog)._

Custom tags provided by user.

**Required**

- [`tag`](#spec.datadog.datadog_tags.tag-property){: name='spec.datadog.datadog_tags.tag-property'} (string, MinLength: 1, MaxLength: 200). Tag format and usage are described here: https://docs.datadoghq.com/getting_started/tagging. Tags with prefix `aiven-` are reserved for Aiven.

**Optional**

- [`comment`](#spec.datadog.datadog_tags.comment-property){: name='spec.datadog.datadog_tags.comment-property'} (string, MaxLength: 1024). Optional tag explanation.

### opensearch {: #spec.datadog.opensearch }

_Appears on [`spec.datadog`](#spec.datadog)._

Datadog Opensearch Options.

**Optional**

- [`index_stats_enabled`](#spec.datadog.opensearch.index_stats_enabled-property){: name='spec.datadog.opensearch.index_stats_enabled-property'} (boolean). Enable Datadog Opensearch Index Monitoring.
- [`pending_task_stats_enabled`](#spec.datadog.opensearch.pending_task_stats_enabled-property){: name='spec.datadog.opensearch.pending_task_stats_enabled-property'} (boolean). Enable Datadog Opensearch Pending Task Monitoring.
- [`pshard_stats_enabled`](#spec.datadog.opensearch.pshard_stats_enabled-property){: name='spec.datadog.opensearch.pshard_stats_enabled-property'} (boolean). Enable Datadog Opensearch Primary Shard Monitoring.

## externalAWSCloudwatchMetrics {: #spec.externalAWSCloudwatchMetrics }

_Appears on [`spec`](#spec)._

External AWS CloudWatch Metrics integration Logs configuration values.

**Optional**

- [`dropped_metrics`](#spec.externalAWSCloudwatchMetrics.dropped_metrics-property){: name='spec.externalAWSCloudwatchMetrics.dropped_metrics-property'} (array of objects, MaxItems: 1024). Metrics to not send to AWS CloudWatch (takes precedence over extra_metrics). See below for [nested schema](#spec.externalAWSCloudwatchMetrics.dropped_metrics).
- [`extra_metrics`](#spec.externalAWSCloudwatchMetrics.extra_metrics-property){: name='spec.externalAWSCloudwatchMetrics.extra_metrics-property'} (array of objects, MaxItems: 1024). Metrics to allow through to AWS CloudWatch (in addition to default metrics). See below for [nested schema](#spec.externalAWSCloudwatchMetrics.extra_metrics).

### dropped_metrics {: #spec.externalAWSCloudwatchMetrics.dropped_metrics }

_Appears on [`spec.externalAWSCloudwatchMetrics`](#spec.externalAWSCloudwatchMetrics)._

Metrics to not send to AWS CloudWatch (takes precedence over extra_metrics).

**Required**

- [`field`](#spec.externalAWSCloudwatchMetrics.dropped_metrics.field-property){: name='spec.externalAWSCloudwatchMetrics.dropped_metrics.field-property'} (string, MaxLength: 1000). Identifier of a value in the metric.
- [`metric`](#spec.externalAWSCloudwatchMetrics.dropped_metrics.metric-property){: name='spec.externalAWSCloudwatchMetrics.dropped_metrics.metric-property'} (string, MaxLength: 1000). Identifier of the metric.

### extra_metrics {: #spec.externalAWSCloudwatchMetrics.extra_metrics }

_Appears on [`spec.externalAWSCloudwatchMetrics`](#spec.externalAWSCloudwatchMetrics)._

Metrics to allow through to AWS CloudWatch (in addition to default metrics).

**Required**

- [`field`](#spec.externalAWSCloudwatchMetrics.extra_metrics.field-property){: name='spec.externalAWSCloudwatchMetrics.extra_metrics.field-property'} (string, MaxLength: 1000). Identifier of a value in the metric.
- [`metric`](#spec.externalAWSCloudwatchMetrics.extra_metrics.metric-property){: name='spec.externalAWSCloudwatchMetrics.extra_metrics.metric-property'} (string, MaxLength: 1000). Identifier of the metric.

## kafkaConnect {: #spec.kafkaConnect }

_Appears on [`spec`](#spec)._

Kafka Connect service configuration values.

**Required**

- [`kafka_connect`](#spec.kafkaConnect.kafka_connect-property){: name='spec.kafkaConnect.kafka_connect-property'} (object). Kafka Connect service configuration values. See below for [nested schema](#spec.kafkaConnect.kafka_connect).

### kafka_connect {: #spec.kafkaConnect.kafka_connect }

_Appears on [`spec.kafkaConnect`](#spec.kafkaConnect)._

Kafka Connect service configuration values.

**Optional**

- [`config_storage_topic`](#spec.kafkaConnect.kafka_connect.config_storage_topic-property){: name='spec.kafkaConnect.kafka_connect.config_storage_topic-property'} (string, MaxLength: 249). The name of the topic where connector and task configuration data are stored.This must be the same for all workers with the same group_id.
- [`group_id`](#spec.kafkaConnect.kafka_connect.group_id-property){: name='spec.kafkaConnect.kafka_connect.group_id-property'} (string, MaxLength: 249). A unique string that identifies the Connect cluster group this worker belongs to.
- [`offset_storage_topic`](#spec.kafkaConnect.kafka_connect.offset_storage_topic-property){: name='spec.kafkaConnect.kafka_connect.offset_storage_topic-property'} (string, MaxLength: 249). The name of the topic where connector and task configuration offsets are stored.This must be the same for all workers with the same group_id.
- [`status_storage_topic`](#spec.kafkaConnect.kafka_connect.status_storage_topic-property){: name='spec.kafkaConnect.kafka_connect.status_storage_topic-property'} (string, MaxLength: 249). The name of the topic where connector and task configuration status updates are stored.This must be the same for all workers with the same group_id.

## kafkaLogs {: #spec.kafkaLogs }

_Appears on [`spec`](#spec)._

Kafka logs configuration values.

**Required**

- [`kafka_topic`](#spec.kafkaLogs.kafka_topic-property){: name='spec.kafkaLogs.kafka_topic-property'} (string, MinLength: 1, MaxLength: 249). Topic name.

## kafkaMirrormaker {: #spec.kafkaMirrormaker }

_Appears on [`spec`](#spec)._

Kafka MirrorMaker configuration values.

**Optional**

- [`cluster_alias`](#spec.kafkaMirrormaker.cluster_alias-property){: name='spec.kafkaMirrormaker.cluster_alias-property'} (string, Pattern: `^[a-zA-Z0-9_.-]+$`, MaxLength: 128). The alias under which the Kafka cluster is known to MirrorMaker. Can contain the following symbols: ASCII alphanumerics, `.`, `_`, and `-`.
- [`kafka_mirrormaker`](#spec.kafkaMirrormaker.kafka_mirrormaker-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker-property'} (object). Kafka MirrorMaker configuration values. See below for [nested schema](#spec.kafkaMirrormaker.kafka_mirrormaker).

### kafka_mirrormaker {: #spec.kafkaMirrormaker.kafka_mirrormaker }

_Appears on [`spec.kafkaMirrormaker`](#spec.kafkaMirrormaker)._

Kafka MirrorMaker configuration values.

**Optional**

- [`consumer_fetch_min_bytes`](#spec.kafkaMirrormaker.kafka_mirrormaker.consumer_fetch_min_bytes-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.consumer_fetch_min_bytes-property'} (integer, Minimum: 1, Maximum: 5242880). The minimum amount of data the server should return for a fetch request.
- [`producer_batch_size`](#spec.kafkaMirrormaker.kafka_mirrormaker.producer_batch_size-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.producer_batch_size-property'} (integer, Minimum: 0, Maximum: 5242880). The batch size in bytes producer will attempt to collect before publishing to broker.
- [`producer_buffer_memory`](#spec.kafkaMirrormaker.kafka_mirrormaker.producer_buffer_memory-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.producer_buffer_memory-property'} (integer, Minimum: 5242880, Maximum: 134217728). The amount of bytes producer can use for buffering data before publishing to broker.
- [`producer_linger_ms`](#spec.kafkaMirrormaker.kafka_mirrormaker.producer_linger_ms-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.producer_linger_ms-property'} (integer, Minimum: 0, Maximum: 5000). The linger time (ms) for waiting new data to arrive for publishing.
- [`producer_max_request_size`](#spec.kafkaMirrormaker.kafka_mirrormaker.producer_max_request_size-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.producer_max_request_size-property'} (integer, Minimum: 0, Maximum: 67108864). The maximum request size in bytes.

## logs {: #spec.logs }

_Appears on [`spec`](#spec)._

Logs configuration values.

**Optional**

- [`elasticsearch_index_days_max`](#spec.logs.elasticsearch_index_days_max-property){: name='spec.logs.elasticsearch_index_days_max-property'} (integer, Minimum: 1, Maximum: 10000). Elasticsearch index retention limit.
- [`elasticsearch_index_prefix`](#spec.logs.elasticsearch_index_prefix-property){: name='spec.logs.elasticsearch_index_prefix-property'} (string, MinLength: 1, MaxLength: 1024). Elasticsearch index prefix.

## metrics {: #spec.metrics }

_Appears on [`spec`](#spec)._

Metrics configuration values.

**Optional**

- [`database`](#spec.metrics.database-property){: name='spec.metrics.database-property'} (string, Pattern: `^[_A-Za-z0-9][-_A-Za-z0-9]{0,39}$`, MaxLength: 40). Name of the database where to store metric datapoints. Only affects PostgreSQL destinations. Defaults to `metrics`. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service.
- [`retention_days`](#spec.metrics.retention_days-property){: name='spec.metrics.retention_days-property'} (integer, Minimum: 0, Maximum: 10000). Number of days to keep old metrics. Only affects PostgreSQL destinations. Set to 0 for no automatic cleanup. Defaults to 30 days.
- [`ro_username`](#spec.metrics.ro_username-property){: name='spec.metrics.ro_username-property'} (string, Pattern: `^[_A-Za-z0-9][-._A-Za-z0-9]{0,39}$`, MaxLength: 40). Name of a user that can be used to read metrics. This will be used for Grafana integration (if enabled) to prevent Grafana users from making undesired changes. Only affects PostgreSQL destinations. Defaults to `metrics_reader`. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service.
- [`source_mysql`](#spec.metrics.source_mysql-property){: name='spec.metrics.source_mysql-property'} (object). Configuration options for metrics where source service is MySQL. See below for [nested schema](#spec.metrics.source_mysql).
- [`username`](#spec.metrics.username-property){: name='spec.metrics.username-property'} (string, Pattern: `^[_A-Za-z0-9][-._A-Za-z0-9]{0,39}$`, MaxLength: 40). Name of the user used to write metrics. Only affects PostgreSQL destinations. Defaults to `metrics_writer`. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service.

### source_mysql {: #spec.metrics.source_mysql }

_Appears on [`spec.metrics`](#spec.metrics)._

Configuration options for metrics where source service is MySQL.

**Required**

- [`telegraf`](#spec.metrics.source_mysql.telegraf-property){: name='spec.metrics.source_mysql.telegraf-property'} (object). Configuration options for Telegraf MySQL input plugin. See below for [nested schema](#spec.metrics.source_mysql.telegraf).

#### telegraf {: #spec.metrics.source_mysql.telegraf }

_Appears on [`spec.metrics.source_mysql`](#spec.metrics.source_mysql)._

Configuration options for Telegraf MySQL input plugin.

**Optional**

- [`gather_event_waits`](#spec.metrics.source_mysql.telegraf.gather_event_waits-property){: name='spec.metrics.source_mysql.telegraf.gather_event_waits-property'} (boolean). Gather metrics from PERFORMANCE_SCHEMA.EVENT_WAITS.
- [`gather_file_events_stats`](#spec.metrics.source_mysql.telegraf.gather_file_events_stats-property){: name='spec.metrics.source_mysql.telegraf.gather_file_events_stats-property'} (boolean). gather metrics from PERFORMANCE_SCHEMA.FILE_SUMMARY_BY_EVENT_NAME.
- [`gather_index_io_waits`](#spec.metrics.source_mysql.telegraf.gather_index_io_waits-property){: name='spec.metrics.source_mysql.telegraf.gather_index_io_waits-property'} (boolean). Gather metrics from PERFORMANCE_SCHEMA.TABLE_IO_WAITS_SUMMARY_BY_INDEX_USAGE.
- [`gather_info_schema_auto_inc`](#spec.metrics.source_mysql.telegraf.gather_info_schema_auto_inc-property){: name='spec.metrics.source_mysql.telegraf.gather_info_schema_auto_inc-property'} (boolean). Gather auto_increment columns and max values from information schema.
- [`gather_innodb_metrics`](#spec.metrics.source_mysql.telegraf.gather_innodb_metrics-property){: name='spec.metrics.source_mysql.telegraf.gather_innodb_metrics-property'} (boolean). Gather metrics from INFORMATION_SCHEMA.INNODB_METRICS.
- [`gather_perf_events_statements`](#spec.metrics.source_mysql.telegraf.gather_perf_events_statements-property){: name='spec.metrics.source_mysql.telegraf.gather_perf_events_statements-property'} (boolean). Gather metrics from PERFORMANCE_SCHEMA.EVENTS_STATEMENTS_SUMMARY_BY_DIGEST.
- [`gather_process_list`](#spec.metrics.source_mysql.telegraf.gather_process_list-property){: name='spec.metrics.source_mysql.telegraf.gather_process_list-property'} (boolean). Gather thread state counts from INFORMATION_SCHEMA.PROCESSLIST.
- [`gather_slave_status`](#spec.metrics.source_mysql.telegraf.gather_slave_status-property){: name='spec.metrics.source_mysql.telegraf.gather_slave_status-property'} (boolean). Gather metrics from SHOW SLAVE STATUS command output.
- [`gather_table_io_waits`](#spec.metrics.source_mysql.telegraf.gather_table_io_waits-property){: name='spec.metrics.source_mysql.telegraf.gather_table_io_waits-property'} (boolean). Gather metrics from PERFORMANCE_SCHEMA.TABLE_IO_WAITS_SUMMARY_BY_TABLE.
- [`gather_table_lock_waits`](#spec.metrics.source_mysql.telegraf.gather_table_lock_waits-property){: name='spec.metrics.source_mysql.telegraf.gather_table_lock_waits-property'} (boolean). Gather metrics from PERFORMANCE_SCHEMA.TABLE_LOCK_WAITS.
- [`gather_table_schema`](#spec.metrics.source_mysql.telegraf.gather_table_schema-property){: name='spec.metrics.source_mysql.telegraf.gather_table_schema-property'} (boolean). Gather metrics from INFORMATION_SCHEMA.TABLES.
- [`perf_events_statements_digest_text_limit`](#spec.metrics.source_mysql.telegraf.perf_events_statements_digest_text_limit-property){: name='spec.metrics.source_mysql.telegraf.perf_events_statements_digest_text_limit-property'} (integer, Minimum: 1, Maximum: 2048). Truncates digest text from perf_events_statements into this many characters.
- [`perf_events_statements_limit`](#spec.metrics.source_mysql.telegraf.perf_events_statements_limit-property){: name='spec.metrics.source_mysql.telegraf.perf_events_statements_limit-property'} (integer, Minimum: 1, Maximum: 4000). Limits metrics from perf_events_statements.
- [`perf_events_statements_time_limit`](#spec.metrics.source_mysql.telegraf.perf_events_statements_time_limit-property){: name='spec.metrics.source_mysql.telegraf.perf_events_statements_time_limit-property'} (integer, Minimum: 1, Maximum: 2592000). Only include perf_events_statements whose last seen is less than this many seconds.

