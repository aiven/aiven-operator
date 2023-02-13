---
title: "ServiceIntegration"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | ServiceIntegration |

ServiceIntegrationSpec defines the desired state of ServiceIntegration.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`datadog`](#datadog){: name='datadog'} (object). Datadog specific user configuration options. See [below for nested schema](#datadog).
- [`destinationEndpointId`](#destinationEndpointId){: name='destinationEndpointId'} (string). Destination endpoint for the integration (if any). 
- [`destinationServiceName`](#destinationServiceName){: name='destinationServiceName'} (string). Destination service for the integration (if any). 
- [`integrationType`](#integrationType){: name='integrationType'} (string, Enum: `datadog`, `kafka_logs`, `kafka_connect`, `metrics`, `dashboard`, `rsyslog`, `read_replica`, `schema_registry_proxy`, `signalfx`, `jolokia`, `internal_connectivity`, `external_google_cloud_logging`, `datasource`). Type of the service integration. 
- [`kafkaConnect`](#kafkaConnect){: name='kafkaConnect'} (object). Kafka Connect service configuration values. See [below for nested schema](#kafkaConnect).
- [`kafkaLogs`](#kafkaLogs){: name='kafkaLogs'} (object). Kafka logs configuration values. See [below for nested schema](#kafkaLogs).
- [`metrics`](#metrics){: name='metrics'} (object). Metrics configuration values. See [below for nested schema](#metrics).
- [`project`](#project){: name='project'} (string, MaxLength: 63). Project the integration belongs to. 
- [`sourceEndpointID`](#sourceEndpointID){: name='sourceEndpointID'} (string). Source endpoint for the integration (if any). 
- [`sourceServiceName`](#sourceServiceName){: name='sourceServiceName'} (string). Source service for the integration (if any). 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

## datadog {: #datadog }

Datadog specific user configuration options.

**Optional**

- [`exclude_consumer_groups`](#exclude_consumer_groups){: name='exclude_consumer_groups'} (array). Consumer groups to exclude. 
- [`exclude_topics`](#exclude_topics){: name='exclude_topics'} (array). List of topics to exclude. 
- [`include_consumer_groups`](#include_consumer_groups){: name='include_consumer_groups'} (array). Consumer groups to include. 
- [`include_topics`](#include_topics){: name='include_topics'} (array). Topics to include. 
- [`kafka_custom_metrics`](#kafka_custom_metrics){: name='kafka_custom_metrics'} (array). List of custom metrics. 

## kafkaConnect {: #kafkaConnect }

Kafka Connect service configuration values.

**Required**

- [`kafka_connect`](#kafka_connect){: name='kafka_connect'} (object).  See [below for nested schema](#kafka_connect).

### kafka_connect {: #kafka_connect }

**Optional**

- [`config_storage_topic`](#config_storage_topic){: name='config_storage_topic'} (string, MaxLength: 249). The name of the topic where connector and task configuration data are stored. This must be the same for all workers with the same group_id. 
- [`group_id`](#group_id){: name='group_id'} (string, MaxLength: 249). A unique string that identifies the Connect cluster group this worker belongs to. 
- [`offset_storage_topic`](#offset_storage_topic){: name='offset_storage_topic'} (string, MaxLength: 249). The name of the topic where connector and task configuration offsets are stored. This must be the same for all workers with the same group_id. 
- [`status_storage_topic`](#status_storage_topic){: name='status_storage_topic'} (string, MaxLength: 249). The name of the topic where connector and task configuration status updates are stored.This must be the same for all workers with the same group_id. 

## kafkaLogs {: #kafkaLogs }

Kafka logs configuration values.

**Required**

- [`kafka_topic`](#kafka_topic){: name='kafka_topic'} (string, MinLength: 1, MaxLength: 63). Topic name. 

## metrics {: #metrics }

Metrics configuration values.

**Optional**

- [`database`](#database){: name='database'} (string, MaxLength: 40). Name of the database where to store metric datapoints. Only affects PostgreSQL destinations. 
- [`retention_days`](#retention_days){: name='retention_days'} (integer). Number of days to keep old metrics. Only affects PostgreSQL destinations. Set to 0 for no automatic cleanup. Defaults to 30 days. 
- [`ro_username`](#ro_username){: name='ro_username'} (string, MaxLength: 40). Name of a user that can be used to read metrics. This will be used for Grafana integration (if enabled) to prevent Grafana users from making undesired changes. Only affects PostgreSQL destinations. Defaults to 'metrics_reader'. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service. 
- [`username`](#username){: name='username'} (string, MaxLength: 40). Name of the user used to write metrics. Only affects PostgreSQL destinations. Defaults to 'metrics_writer'. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service. 

