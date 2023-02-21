---
title: "ServiceIntegration"
---

## Schema {: #Schema }

ServiceIntegration is the Schema for the serviceintegrations API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Must be equal to `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Must be equal to `ServiceIntegration`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ServiceIntegrationSpec defines the desired state of ServiceIntegration. See below for [nested schema](#spec).

## spec {: #spec }

ServiceIntegrationSpec defines the desired state of ServiceIntegration.

**Required**

- [`integrationType`](#spec.integrationType-property){: name='spec.integrationType-property'} (string, Enum: `datadog`, `kafka_logs`, `kafka_connect`, `metrics`, `dashboard`, `rsyslog`, `read_replica`, `schema_registry_proxy`, `signalfx`, `jolokia`, `internal_connectivity`, `external_google_cloud_logging`, `datasource`). Type of the service integration.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, MaxLength: 63). Project the integration belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`datadog`](#spec.datadog-property){: name='spec.datadog-property'} (object). Datadog specific user configuration options. See below for [nested schema](#spec.datadog).
- [`destinationEndpointId`](#spec.destinationEndpointId-property){: name='spec.destinationEndpointId-property'} (string). Destination endpoint for the integration (if any).
- [`destinationServiceName`](#spec.destinationServiceName-property){: name='spec.destinationServiceName-property'} (string). Destination service for the integration (if any).
- [`kafkaConnect`](#spec.kafkaConnect-property){: name='spec.kafkaConnect-property'} (object). Kafka Connect service configuration values. See below for [nested schema](#spec.kafkaConnect).
- [`kafkaLogs`](#spec.kafkaLogs-property){: name='spec.kafkaLogs-property'} (object). Kafka logs configuration values. See below for [nested schema](#spec.kafkaLogs).
- [`metrics`](#spec.metrics-property){: name='spec.metrics-property'} (object). Metrics configuration values. See below for [nested schema](#spec.metrics).
- [`sourceEndpointID`](#spec.sourceEndpointID-property){: name='spec.sourceEndpointID-property'} (string). Source endpoint for the integration (if any).
- [`sourceServiceName`](#spec.sourceServiceName-property){: name='spec.sourceServiceName-property'} (string). Source service for the integration (if any).

## authSecretRef {: #spec.authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

## datadog {: #spec.datadog }

Datadog specific user configuration options.

**Optional**

- [`exclude_consumer_groups`](#spec.datadog.exclude_consumer_groups-property){: name='spec.datadog.exclude_consumer_groups-property'} (array of strings). Consumer groups to exclude.
- [`exclude_topics`](#spec.datadog.exclude_topics-property){: name='spec.datadog.exclude_topics-property'} (array of strings). List of topics to exclude.
- [`include_consumer_groups`](#spec.datadog.include_consumer_groups-property){: name='spec.datadog.include_consumer_groups-property'} (array of strings). Consumer groups to include.
- [`include_topics`](#spec.datadog.include_topics-property){: name='spec.datadog.include_topics-property'} (array of strings). Topics to include.
- [`kafka_custom_metrics`](#spec.datadog.kafka_custom_metrics-property){: name='spec.datadog.kafka_custom_metrics-property'} (array of strings). List of custom metrics.

## kafkaConnect {: #spec.kafkaConnect }

Kafka Connect service configuration values.

**Required**

- [`kafka_connect`](#spec.kafkaConnect.kafka_connect-property){: name='spec.kafkaConnect.kafka_connect-property'} (object).  See below for [nested schema](#spec.kafkaConnect.kafka_connect).

### kafka_connect {: #spec.kafkaConnect.kafka_connect }

**Optional**

- [`config_storage_topic`](#spec.kafkaConnect.kafka_connect.config_storage_topic-property){: name='spec.kafkaConnect.kafka_connect.config_storage_topic-property'} (string, MaxLength: 249). The name of the topic where connector and task configuration data are stored. This must be the same for all workers with the same group_id.
- [`group_id`](#spec.kafkaConnect.kafka_connect.group_id-property){: name='spec.kafkaConnect.kafka_connect.group_id-property'} (string, MaxLength: 249). A unique string that identifies the Connect cluster group this worker belongs to.
- [`offset_storage_topic`](#spec.kafkaConnect.kafka_connect.offset_storage_topic-property){: name='spec.kafkaConnect.kafka_connect.offset_storage_topic-property'} (string, MaxLength: 249). The name of the topic where connector and task configuration offsets are stored. This must be the same for all workers with the same group_id.
- [`status_storage_topic`](#spec.kafkaConnect.kafka_connect.status_storage_topic-property){: name='spec.kafkaConnect.kafka_connect.status_storage_topic-property'} (string, MaxLength: 249). The name of the topic where connector and task configuration status updates are stored.This must be the same for all workers with the same group_id.

## kafkaLogs {: #spec.kafkaLogs }

Kafka logs configuration values.

**Required**

- [`kafka_topic`](#spec.kafkaLogs.kafka_topic-property){: name='spec.kafkaLogs.kafka_topic-property'} (string, MinLength: 1, MaxLength: 63). Topic name.

## metrics {: #spec.metrics }

Metrics configuration values.

**Optional**

- [`database`](#spec.metrics.database-property){: name='spec.metrics.database-property'} (string, MaxLength: 40). Name of the database where to store metric datapoints. Only affects PostgreSQL destinations.
- [`retention_days`](#spec.metrics.retention_days-property){: name='spec.metrics.retention_days-property'} (integer). Number of days to keep old metrics. Only affects PostgreSQL destinations. Set to 0 for no automatic cleanup. Defaults to 30 days.
- [`ro_username`](#spec.metrics.ro_username-property){: name='spec.metrics.ro_username-property'} (string, MaxLength: 40). Name of a user that can be used to read metrics. This will be used for Grafana integration (if enabled) to prevent Grafana users from making undesired changes. Only affects PostgreSQL destinations. Defaults to 'metrics_reader'. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service.
- [`username`](#spec.metrics.username-property){: name='spec.metrics.username-property'} (string, MaxLength: 40). Name of the user used to write metrics. Only affects PostgreSQL destinations. Defaults to 'metrics_writer'. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service.

