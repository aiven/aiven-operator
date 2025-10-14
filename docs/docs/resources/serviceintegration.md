---
title: "ServiceIntegration"
---

## Usage examples

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

	
=== "autoscaler"

    ```yaml linenums="1"
    apiVersion: aiven.io/v1alpha1
    kind: ServiceIntegration
    metadata:
      name: my-service-integration
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      integrationType: autoscaler
      sourceServiceName: my-pg
      # Look up autoscaler integration endpoint ID via Console
      destinationEndpointId: my-destination-endpoint-id
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: PostgreSQL
    metadata:
      name: my-pg
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: startup-4
    ```

	
=== "clickhouse_postgresql"

    ```yaml linenums="1"
    apiVersion: aiven.io/v1alpha1
    kind: ServiceIntegration
    metadata:
      name: my-service-integration
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      integrationType: clickhouse_postgresql
      sourceServiceName: my-pg
      destinationServiceName: my-clickhouse
    
      clickhousePostgresql:
        databases:
          - database: defaultdb
            schema: public
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: Clickhouse
    metadata:
      name: my-clickhouse
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: startup-16
      maintenanceWindowDow: friday
      maintenanceWindowTime: 23:00:00
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: PostgreSQL
    metadata:
      name: my-pg
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: startup-4
      maintenanceWindowDow: friday
      maintenanceWindowTime: 23:00:00
    ```

	
=== "datadog"

    ```yaml linenums="1"
    apiVersion: aiven.io/v1alpha1
    kind: ServiceIntegration
    metadata:
      name: my-service-integration
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      integrationType: datadog
      sourceServiceName: my-pg
      destinationEndpointId: destination-endpoint-id
    
      datadog:
        datadog_dbm_enabled: True
        datadog_tags:
          - tag: env
            comment: test
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: PostgreSQL
    metadata:
      name: my-pg
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: startup-4
    ```

	
=== "kafka_connect"

    ```yaml linenums="1"
    apiVersion: aiven.io/v1alpha1
    kind: ServiceIntegration
    metadata:
      name: my-service-integration
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      integrationType: kafka_connect
      sourceServiceName: my-kafka
      destinationServiceName: my-kafka-connect
    
      kafkaConnect:
        kafka_connect:
          group_id: connect
          status_storage_topic: __connect_status
          offset_storage_topic: __connect_offsets
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: Kafka
    metadata:
      name: my-kafka
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: business-4
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: KafkaConnect
    metadata:
      name: my-kafka-connect
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: business-4
    
      userConfig:
        kafka_connect:
          consumer_isolation_level: read_committed
        public_access:
          kafka_connect: true
    ```

	
=== "kafka_logs"

    ```yaml linenums="1"
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
      sourceServiceName: my-kafka
      destinationServiceName: my-kafka
    
      kafkaLogs:
        kafka_topic: my-kafka-topic
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: Kafka
    metadata:
      name: my-kafka
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: business-4
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: KafkaTopic
    metadata:
      name: my-kafka-topic
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      serviceName: my-kafka
      replication: 2
      partitions: 1
    ```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `ServiceIntegration`:

```shell
kubectl get serviceintegrations my-service-integration
```

The output is similar to the following:
```shell
Name                      Project               Type          Source Service Name    Destination Endpoint ID       
my-service-integration    aiven-project-name    autoscaler    my-pg                  my-destination-endpoint-id    
```

---

## ServiceIntegration {: #ServiceIntegration }

ServiceIntegration is the Schema for the serviceintegrations API.

!!! info "Adoption of existing integrations"

    If a ServiceIntegration resource is created with configuration matching an existing Aiven integration (created outside the operator), the operator will adopt the existing integration.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ServiceIntegration`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ServiceIntegrationSpec defines the desired state of ServiceIntegration. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ServiceIntegration`](#ServiceIntegration)._

ServiceIntegrationSpec defines the desired state of ServiceIntegration.

**Required**

- [`integrationType`](#spec.integrationType-property){: name='spec.integrationType-property'} (string, Enum: `alertmanager`, `autoscaler`, `caching`, `cassandra_cross_service_cluster`, `clickhouse_kafka`, `clickhouse_postgresql`, `dashboard`, `datadog`, `datasource`, `external_aws_cloudwatch_logs`, `external_aws_cloudwatch_metrics`, `external_elasticsearch_logs`, `external_google_cloud_logging`, `external_opensearch_logs`, `flink`, `flink_external_kafka`, `flink_external_postgresql`, `internal_connectivity`, `jolokia`, `kafka_connect`, `kafka_logs`, `kafka_mirrormaker`, `logs`, `m3aggregator`, `m3coordinator`, `metrics`, `opensearch_cross_cluster_replication`, `opensearch_cross_cluster_search`, `prometheus`, `read_replica`, `rsyslog`, `schema_registry_proxy`, `stresstester`, `thanosquery`, `thanosstore`, `vmalert`, Immutable). Type of the service integration accepted by Aiven API. Some values may not be supported by the operator.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`autoscaler`](#spec.autoscaler-property){: name='spec.autoscaler-property'} (object). Autoscaler specific user configuration options.
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

- [`tables`](#spec.clickhouseKafka.tables-property){: name='spec.clickhouseKafka.tables-property'} (array of objects, MaxItems: 400). Array of table configurations that define how Kafka topics are mapped to ClickHouse tables. Each table configuration specifies the table structure, associated Kafka topics, and read/write settings. See below for [nested schema](#spec.clickhouseKafka.tables).

### tables {: #spec.clickhouseKafka.tables }

_Appears on [`spec.clickhouseKafka`](#spec.clickhouseKafka)._

Table to create.

**Required**

- [`columns`](#spec.clickhouseKafka.tables.columns-property){: name='spec.clickhouseKafka.tables.columns-property'} (array of objects, MaxItems: 100). Array of column definitions that specify the structure of the ClickHouse table. Each column maps to a field in the Kafka messages. See below for [nested schema](#spec.clickhouseKafka.tables.columns).
- [`data_format`](#spec.clickhouseKafka.tables.data_format-property){: name='spec.clickhouseKafka.tables.data_format-property'} (string, Enum: `Avro`, `AvroConfluent`, `CSV`, `JSONAsString`, `JSONCompactEachRow`, `JSONCompactStringsEachRow`, `JSONEachRow`, `JSONStringsEachRow`, `MsgPack`, `Parquet`, `RawBLOB`, `TSKV`, `TSV`, `TabSeparated`). The format of the messages in the Kafka topics. Determines how ClickHouse parses and serializes the data (e.g., JSON, CSV, Avro).
- [`group_name`](#spec.clickhouseKafka.tables.group_name-property){: name='spec.clickhouseKafka.tables.group_name-property'} (string, MinLength: 1, MaxLength: 249). The Kafka consumer group name. Multiple consumers with the same group name will share the workload and maintain offset positions.
- [`name`](#spec.clickhouseKafka.tables.name-property){: name='spec.clickhouseKafka.tables.name-property'} (string, MinLength: 1, MaxLength: 40). The name of the ClickHouse table to be created. This table can consume data from and write data to the specified Kafka topics.
- [`topics`](#spec.clickhouseKafka.tables.topics-property){: name='spec.clickhouseKafka.tables.topics-property'} (array of objects, MaxItems: 100). Array of Kafka topics that this table will read data from or write data to. Messages from all specified topics will be inserted into this table, and data inserted into this table will be published to the topics. See below for [nested schema](#spec.clickhouseKafka.tables.topics).

**Optional**

- [`auto_offset_reset`](#spec.clickhouseKafka.tables.auto_offset_reset-property){: name='spec.clickhouseKafka.tables.auto_offset_reset-property'} (string, Enum: `beginning`, `earliest`, `end`, `largest`, `latest`, `smallest`). Determines where to start reading from Kafka when no offset is stored or the stored offset is out of range. `earliest` starts from the beginning, `latest` starts from the end.
- [`date_time_input_format`](#spec.clickhouseKafka.tables.date_time_input_format-property){: name='spec.clickhouseKafka.tables.date_time_input_format-property'} (string, Enum: `basic`, `best_effort`, `best_effort_us`). Specifies how ClickHouse should parse DateTime values from text-based input formats. `basic` uses simple parsing, `best_effort` attempts more flexible parsing.
- [`handle_error_mode`](#spec.clickhouseKafka.tables.handle_error_mode-property){: name='spec.clickhouseKafka.tables.handle_error_mode-property'} (string, Enum: `default`, `stream`). Defines how ClickHouse should handle errors when processing Kafka messages. `default` stops on errors, `stream` continues processing and logs errors.
- [`max_block_size`](#spec.clickhouseKafka.tables.max_block_size-property){: name='spec.clickhouseKafka.tables.max_block_size-property'} (integer, Minimum: 0, Maximum: 1000000000). Maximum number of rows to collect before flushing data between Kafka and ClickHouse.
- [`max_rows_per_message`](#spec.clickhouseKafka.tables.max_rows_per_message-property){: name='spec.clickhouseKafka.tables.max_rows_per_message-property'} (integer, Minimum: 1, Maximum: 1000000000). Maximum number of rows that can be processed from a single Kafka message for row-based formats. Useful for controlling memory usage.
- [`num_consumers`](#spec.clickhouseKafka.tables.num_consumers-property){: name='spec.clickhouseKafka.tables.num_consumers-property'} (integer, Minimum: 1, Maximum: 10). Number of Kafka consumers to run per table per replica. Increasing this can improve throughput but may increase resource usage.
- [`poll_max_batch_size`](#spec.clickhouseKafka.tables.poll_max_batch_size-property){: name='spec.clickhouseKafka.tables.poll_max_batch_size-property'} (integer, Minimum: 0, Maximum: 1000000000). Maximum number of messages to fetch in a single Kafka poll operation for reading.
- [`poll_max_timeout_ms`](#spec.clickhouseKafka.tables.poll_max_timeout_ms-property){: name='spec.clickhouseKafka.tables.poll_max_timeout_ms-property'} (integer, Minimum: 0, Maximum: 30000). Timeout in milliseconds for a single poll from Kafka. Takes the value of the stream_flush_interval_ms server setting by default (500ms).
- [`producer_batch_num_messages`](#spec.clickhouseKafka.tables.producer_batch_num_messages-property){: name='spec.clickhouseKafka.tables.producer_batch_num_messages-property'} (integer, Minimum: 1, Maximum: 1000000). The maximum number of messages in a batch sent to Kafka. If the number of messages exceeds this value, the batch is sent.
- [`producer_batch_size`](#spec.clickhouseKafka.tables.producer_batch_size-property){: name='spec.clickhouseKafka.tables.producer_batch_size-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum size in bytes of a batch of messages sent to Kafka. If the batch size is exceeded, the batch is sent.
- [`producer_compression_codec`](#spec.clickhouseKafka.tables.producer_compression_codec-property){: name='spec.clickhouseKafka.tables.producer_compression_codec-property'} (string, Enum: `gzip`, `lz4`, `none`, `snappy`, `zstd`). The compression codec to use when sending a batch of messages to Kafka.
- [`producer_compression_level`](#spec.clickhouseKafka.tables.producer_compression_level-property){: name='spec.clickhouseKafka.tables.producer_compression_level-property'} (integer, Minimum: -1, Maximum: 12). The compression level to use when sending a batch of messages to Kafka. Usable range is algorithm-dependent: [0-9] for gzip; [0-12] for lz4; only 0 for snappy; -1 = codec-dependent default compression level.
- [`producer_linger_ms`](#spec.clickhouseKafka.tables.producer_linger_ms-property){: name='spec.clickhouseKafka.tables.producer_linger_ms-property'} (integer, Minimum: 0, Maximum: 900000). The time in milliseconds to wait for additional messages before sending a batch. If the time is exceeded, the batch is sent.
- [`producer_queue_buffering_max_kbytes`](#spec.clickhouseKafka.tables.producer_queue_buffering_max_kbytes-property){: name='spec.clickhouseKafka.tables.producer_queue_buffering_max_kbytes-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum size of the buffer in kilobytes before sending.
- [`producer_queue_buffering_max_messages`](#spec.clickhouseKafka.tables.producer_queue_buffering_max_messages-property){: name='spec.clickhouseKafka.tables.producer_queue_buffering_max_messages-property'} (integer, Minimum: 0, Maximum: 2147483647). The maximum number of messages to buffer before sending.
- [`producer_request_required_acks`](#spec.clickhouseKafka.tables.producer_request_required_acks-property){: name='spec.clickhouseKafka.tables.producer_request_required_acks-property'} (integer, Minimum: -1, Maximum: 1000). The number of acknowledgements the leader broker must receive from ISR brokers before responding to the request: 0=Broker does not send any response/ack to client, -1 will block until message is committed by all in sync replicas (ISRs).
- [`skip_broken_messages`](#spec.clickhouseKafka.tables.skip_broken_messages-property){: name='spec.clickhouseKafka.tables.skip_broken_messages-property'} (integer, Minimum: 0, Maximum: 1000000000). Number of broken messages to skip before stopping processing when reading from Kafka. Useful for handling corrupted data without failing the entire integration.
- [`thread_per_consumer`](#spec.clickhouseKafka.tables.thread_per_consumer-property){: name='spec.clickhouseKafka.tables.thread_per_consumer-property'} (boolean). When enabled, each consumer runs in its own thread, providing better isolation and potentially better performance for high-throughput scenarios.

#### columns {: #spec.clickhouseKafka.tables.columns }

_Appears on [`spec.clickhouseKafka.tables`](#spec.clickhouseKafka.tables)._

Table column.

**Required**

- [`name`](#spec.clickhouseKafka.tables.columns.name-property){: name='spec.clickhouseKafka.tables.columns.name-property'} (string, MinLength: 1, MaxLength: 40). The name of the column in the ClickHouse table. This should match the field names in your Kafka message format.
- [`type`](#spec.clickhouseKafka.tables.columns.type-property){: name='spec.clickhouseKafka.tables.columns.type-property'} (string, MinLength: 1, MaxLength: 1000). The ClickHouse data type for this column. Must be a valid ClickHouse data type that can handle the data format.

#### topics {: #spec.clickhouseKafka.tables.topics }

_Appears on [`spec.clickhouseKafka.tables`](#spec.clickhouseKafka.tables)._

Kafka topic.

**Required**

- [`name`](#spec.clickhouseKafka.tables.topics.name-property){: name='spec.clickhouseKafka.tables.topics.name-property'} (string, MinLength: 1, MaxLength: 249). The name of the Kafka topic to read messages from or write messages to. The topic must exist in the Kafka cluster.

## clickhousePostgresql {: #spec.clickhousePostgresql }

_Appears on [`spec`](#spec)._

Clickhouse PostgreSQL configuration values.

**Required**

- [`databases`](#spec.clickhousePostgresql.databases-property){: name='spec.clickhousePostgresql.databases-property'} (array of objects, MaxItems: 10). Databases to expose. See below for [nested schema](#spec.clickhousePostgresql.databases).

### databases {: #spec.clickhousePostgresql.databases }

_Appears on [`spec.clickhousePostgresql`](#spec.clickhousePostgresql)._

Database to expose.

**Optional**

- [`database`](#spec.clickhousePostgresql.databases.database-property){: name='spec.clickhousePostgresql.databases.database-property'} (string, MinLength: 1, MaxLength: 63). PostgreSQL database to expose.
- [`schema`](#spec.clickhousePostgresql.databases.schema-property){: name='spec.clickhousePostgresql.databases.schema-property'} (string, MinLength: 1, MaxLength: 63). PostgreSQL schema to expose.

## datadog {: #spec.datadog }

_Appears on [`spec`](#spec)._

Datadog specific user configuration options.

**Optional**

- [`datadog_dbm_enabled`](#spec.datadog.datadog_dbm_enabled-property){: name='spec.datadog.datadog_dbm_enabled-property'} (boolean). Enable Datadog Database Monitoring.
- [`datadog_pgbouncer_enabled`](#spec.datadog.datadog_pgbouncer_enabled-property){: name='spec.datadog.datadog_pgbouncer_enabled-property'} (boolean). Enable Datadog PgBouncer Metric Tracking.
- [`datadog_tags`](#spec.datadog.datadog_tags-property){: name='spec.datadog.datadog_tags-property'} (array of objects, MaxItems: 32). Custom tags provided by user. See below for [nested schema](#spec.datadog.datadog_tags).
- [`exclude_consumer_groups`](#spec.datadog.exclude_consumer_groups-property){: name='spec.datadog.exclude_consumer_groups-property'} (array of strings, MaxItems: 1024). List of custom metrics.
- [`exclude_topics`](#spec.datadog.exclude_topics-property){: name='spec.datadog.exclude_topics-property'} (array of strings, MaxItems: 1024). List of topics to exclude.
- [`include_consumer_groups`](#spec.datadog.include_consumer_groups-property){: name='spec.datadog.include_consumer_groups-property'} (array of strings, MaxItems: 1024). List of custom metrics.
- [`include_topics`](#spec.datadog.include_topics-property){: name='spec.datadog.include_topics-property'} (array of strings, MaxItems: 1024). List of topics to include.
- [`kafka_custom_metrics`](#spec.datadog.kafka_custom_metrics-property){: name='spec.datadog.kafka_custom_metrics-property'} (array of strings, MaxItems: 1024). List of custom metrics.
- [`max_jmx_metrics`](#spec.datadog.max_jmx_metrics-property){: name='spec.datadog.max_jmx_metrics-property'} (integer, Minimum: 10, Maximum: 100000). Maximum number of JMX metrics to send.
- [`mirrormaker_custom_metrics`](#spec.datadog.mirrormaker_custom_metrics-property){: name='spec.datadog.mirrormaker_custom_metrics-property'} (array of strings, MaxItems: 1024). List of custom metrics.
- [`opensearch`](#spec.datadog.opensearch-property){: name='spec.datadog.opensearch-property'} (object). Datadog Opensearch Options. See below for [nested schema](#spec.datadog.opensearch).
- [`redis`](#spec.datadog.redis-property){: name='spec.datadog.redis-property'} (object). Datadog Redis Options. See below for [nested schema](#spec.datadog.redis).

### datadog_tags {: #spec.datadog.datadog_tags }

_Appears on [`spec.datadog`](#spec.datadog)._

Datadog tag defined by user.

**Required**

- [`tag`](#spec.datadog.datadog_tags.tag-property){: name='spec.datadog.datadog_tags.tag-property'} (string, MinLength: 1, MaxLength: 200). Tag format and usage are described here: https://docs.datadoghq.com/getting_started/tagging. Tags with prefix `aiven-` are reserved for Aiven.

**Optional**

- [`comment`](#spec.datadog.datadog_tags.comment-property){: name='spec.datadog.datadog_tags.comment-property'} (string, MaxLength: 1024). Optional tag explanation.

### opensearch {: #spec.datadog.opensearch }

_Appears on [`spec.datadog`](#spec.datadog)._

Datadog Opensearch Options.

**Optional**

- [`cluster_stats_enabled`](#spec.datadog.opensearch.cluster_stats_enabled-property){: name='spec.datadog.opensearch.cluster_stats_enabled-property'} (boolean). Enable Datadog Opensearch Cluster Monitoring.
- [`index_stats_enabled`](#spec.datadog.opensearch.index_stats_enabled-property){: name='spec.datadog.opensearch.index_stats_enabled-property'} (boolean). Enable Datadog Opensearch Index Monitoring.
- [`pending_task_stats_enabled`](#spec.datadog.opensearch.pending_task_stats_enabled-property){: name='spec.datadog.opensearch.pending_task_stats_enabled-property'} (boolean). Enable Datadog Opensearch Pending Task Monitoring.
- [`pshard_stats_enabled`](#spec.datadog.opensearch.pshard_stats_enabled-property){: name='spec.datadog.opensearch.pshard_stats_enabled-property'} (boolean). Enable Datadog Opensearch Primary Shard Monitoring.

### redis {: #spec.datadog.redis }

_Appears on [`spec.datadog`](#spec.datadog)._

Datadog Redis Options.

**Required**

- [`command_stats_enabled`](#spec.datadog.redis.command_stats_enabled-property){: name='spec.datadog.redis.command_stats_enabled-property'} (boolean). Enable command_stats option in the agent's configuration.

## externalAWSCloudwatchMetrics {: #spec.externalAWSCloudwatchMetrics }

_Appears on [`spec`](#spec)._

External AWS CloudWatch Metrics integration Logs configuration values.

**Optional**

- [`dropped_metrics`](#spec.externalAWSCloudwatchMetrics.dropped_metrics-property){: name='spec.externalAWSCloudwatchMetrics.dropped_metrics-property'} (array of objects, MaxItems: 1024). Metrics to not send to AWS CloudWatch (takes precedence over extra_metrics). See below for [nested schema](#spec.externalAWSCloudwatchMetrics.dropped_metrics).
- [`extra_metrics`](#spec.externalAWSCloudwatchMetrics.extra_metrics-property){: name='spec.externalAWSCloudwatchMetrics.extra_metrics-property'} (array of objects, MaxItems: 1024). Metrics to allow through to AWS CloudWatch (in addition to default metrics). See below for [nested schema](#spec.externalAWSCloudwatchMetrics.extra_metrics).

### dropped_metrics {: #spec.externalAWSCloudwatchMetrics.dropped_metrics }

_Appears on [`spec.externalAWSCloudwatchMetrics`](#spec.externalAWSCloudwatchMetrics)._

Metric name and subfield.

**Required**

- [`field`](#spec.externalAWSCloudwatchMetrics.dropped_metrics.field-property){: name='spec.externalAWSCloudwatchMetrics.dropped_metrics.field-property'} (string, MaxLength: 1000). Identifier of a value in the metric.
- [`metric`](#spec.externalAWSCloudwatchMetrics.dropped_metrics.metric-property){: name='spec.externalAWSCloudwatchMetrics.dropped_metrics.metric-property'} (string, MaxLength: 1000). Identifier of the metric.

### extra_metrics {: #spec.externalAWSCloudwatchMetrics.extra_metrics }

_Appears on [`spec.externalAWSCloudwatchMetrics`](#spec.externalAWSCloudwatchMetrics)._

Metric name and subfield.

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

**Optional**

- [`selected_log_fields`](#spec.kafkaLogs.selected_log_fields-property){: name='spec.kafkaLogs.selected_log_fields-property'} (array of strings, MaxItems: 5). The list of logging fields that will be sent to the integration logging service. The MESSAGE and timestamp fields are always sent.

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

- [`consumer_auto_offset_reset`](#spec.kafkaMirrormaker.kafka_mirrormaker.consumer_auto_offset_reset-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.consumer_auto_offset_reset-property'} (string, Enum: `earliest`, `latest`). Set where consumer starts to consume data. Value `earliest`: Start replication from the earliest offset. Value `latest`: Start replication from the latest offset. Default is `earliest`.
- [`consumer_fetch_min_bytes`](#spec.kafkaMirrormaker.kafka_mirrormaker.consumer_fetch_min_bytes-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.consumer_fetch_min_bytes-property'} (integer, Minimum: 1, Maximum: 5242880). The minimum amount of data the server should return for a fetch request.
- [`consumer_max_poll_records`](#spec.kafkaMirrormaker.kafka_mirrormaker.consumer_max_poll_records-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.consumer_max_poll_records-property'} (integer, Minimum: 100, Maximum: 20000). Set consumer max.poll.records. The default is 500.
- [`producer_batch_size`](#spec.kafkaMirrormaker.kafka_mirrormaker.producer_batch_size-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.producer_batch_size-property'} (integer, Minimum: 0, Maximum: 5242880). The batch size in bytes producer will attempt to collect before publishing to broker.
- [`producer_buffer_memory`](#spec.kafkaMirrormaker.kafka_mirrormaker.producer_buffer_memory-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.producer_buffer_memory-property'} (integer, Minimum: 5242880, Maximum: 134217728). The amount of bytes producer can use for buffering data before publishing to broker.
- [`producer_compression_type`](#spec.kafkaMirrormaker.kafka_mirrormaker.producer_compression_type-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.producer_compression_type-property'} (string, Enum: `gzip`, `lz4`, `none`, `snappy`, `zstd`). Specify the default compression type for producers. This configuration accepts the standard compression codecs (`gzip`, `snappy`, `lz4`, `zstd`). It additionally accepts `none` which is the default and equivalent to no compression.
- [`producer_linger_ms`](#spec.kafkaMirrormaker.kafka_mirrormaker.producer_linger_ms-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.producer_linger_ms-property'} (integer, Minimum: 0, Maximum: 5000). The linger time (ms) for waiting new data to arrive for publishing.
- [`producer_max_request_size`](#spec.kafkaMirrormaker.kafka_mirrormaker.producer_max_request_size-property){: name='spec.kafkaMirrormaker.kafka_mirrormaker.producer_max_request_size-property'} (integer, Minimum: 0, Maximum: 268435456). The maximum request size in bytes.

## logs {: #spec.logs }

_Appears on [`spec`](#spec)._

Logs configuration values.

**Optional**

- [`elasticsearch_index_days_max`](#spec.logs.elasticsearch_index_days_max-property){: name='spec.logs.elasticsearch_index_days_max-property'} (integer, Minimum: 1, Maximum: 10000). Elasticsearch index retention limit.
- [`elasticsearch_index_prefix`](#spec.logs.elasticsearch_index_prefix-property){: name='spec.logs.elasticsearch_index_prefix-property'} (string, Pattern: `^[a-z0-9][a-z0-9-_.]+$`, MinLength: 1, MaxLength: 1024). Elasticsearch index prefix.
- [`selected_log_fields`](#spec.logs.selected_log_fields-property){: name='spec.logs.selected_log_fields-property'} (array of strings, MaxItems: 5). The list of logging fields that will be sent to the integration logging service. The MESSAGE and timestamp fields are always sent.

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
