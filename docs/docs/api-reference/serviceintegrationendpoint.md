---
title: "ServiceIntegrationEndpoint"
---

## Usage examples

??? example "external_postgresql"
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: ServiceIntegrationEndpoint
    metadata:
      name: my-service-integration-endpoint
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      endpointName: my-external-postgresql
      endpointType: external_postgresql
    
      externalPostgresql:
        username: username
        password: password
        host: example.example
        port: 5432
        ssl_mode: require
    ```

??? example "external_schema_registry"
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: ServiceIntegrationEndpoint
    metadata:
      name: my-service-integration-endpoint
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      endpointName: my-external-schema-registry
      endpointType: external_schema_registry
    
      externalSchemaRegistry:
        url: https://schema-registry.example.com:8081
        authentication: basic
        basic_auth_username: username
        basic_auth_password: password
    ```

!!! info
	To create this resource, a `Secret` containing Aiven token must be [created](/aiven-operator/authentication.html) first.

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `ServiceIntegrationEndpoint`:

```shell
kubectl get serviceintegrationendpoints my-service-integration-endpoint
```

The output is similar to the following:
```shell
Name                               Project               Endpoint Name             Endpoint Type          ID      
my-service-integration-endpoint    aiven-project-name    my-external-postgresql    external_postgresql    <id>    
```

## ServiceIntegrationEndpoint {: #ServiceIntegrationEndpoint }

ServiceIntegrationEndpoint is the Schema for the serviceintegrationendpoints API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ServiceIntegrationEndpoint`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ServiceIntegrationEndpointSpec defines the desired state of ServiceIntegrationEndpoint. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ServiceIntegrationEndpoint`](#ServiceIntegrationEndpoint)._

ServiceIntegrationEndpointSpec defines the desired state of ServiceIntegrationEndpoint.

**Required**

- [`endpointType`](#spec.endpointType-property){: name='spec.endpointType-property'} (string, Enum: `autoscaler`, `datadog`, `external_aws_cloudwatch_logs`, `external_aws_cloudwatch_metrics`, `external_aws_s3`, `external_clickhouse`, `external_elasticsearch_logs`, `external_google_cloud_bigquery`, `external_google_cloud_logging`, `external_kafka`, `external_mysql`, `external_opensearch_logs`, `external_postgresql`, `external_redis`, `external_schema_registry`, `external_sumologic_logs`, `jolokia`, `prometheus`, `rsyslog`, Immutable). Type of the service integration endpoint.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`autoscaler`](#spec.autoscaler-property){: name='spec.autoscaler-property'} (object). Autoscaler configuration values. See below for [nested schema](#spec.autoscaler).
- [`datadog`](#spec.datadog-property){: name='spec.datadog-property'} (object). Datadog configuration values. See below for [nested schema](#spec.datadog).
- [`endpointName`](#spec.endpointName-property){: name='spec.endpointName-property'} (string, Immutable, MaxLength: 36). Source endpoint for the integration (if any).
- [`externalAWSCloudwatchLogs`](#spec.externalAWSCloudwatchLogs-property){: name='spec.externalAWSCloudwatchLogs-property'} (object). ExternalAwsCloudwatchLogs configuration values. See below for [nested schema](#spec.externalAWSCloudwatchLogs).
- [`externalAWSCloudwatchMetrics`](#spec.externalAWSCloudwatchMetrics-property){: name='spec.externalAWSCloudwatchMetrics-property'} (object). ExternalAwsCloudwatchMetrics configuration values. See below for [nested schema](#spec.externalAWSCloudwatchMetrics).
- [`externalElasticsearchLogs`](#spec.externalElasticsearchLogs-property){: name='spec.externalElasticsearchLogs-property'} (object). ExternalElasticsearchLogs configuration values. See below for [nested schema](#spec.externalElasticsearchLogs).
- [`externalGoogleCloudBigquery`](#spec.externalGoogleCloudBigquery-property){: name='spec.externalGoogleCloudBigquery-property'} (object). ExternalGoogleCloudBigquery configuration values. See below for [nested schema](#spec.externalGoogleCloudBigquery).
- [`externalGoogleCloudLogging`](#spec.externalGoogleCloudLogging-property){: name='spec.externalGoogleCloudLogging-property'} (object). ExternalGoogleCloudLogging configuration values. See below for [nested schema](#spec.externalGoogleCloudLogging).
- [`externalKafka`](#spec.externalKafka-property){: name='spec.externalKafka-property'} (object). ExternalKafka configuration values. See below for [nested schema](#spec.externalKafka).
- [`externalOpensearchLogs`](#spec.externalOpensearchLogs-property){: name='spec.externalOpensearchLogs-property'} (object). ExternalOpensearchLogs configuration values. See below for [nested schema](#spec.externalOpensearchLogs).
- [`externalPostgresql`](#spec.externalPostgresql-property){: name='spec.externalPostgresql-property'} (object). ExternalPostgresql configuration values. See below for [nested schema](#spec.externalPostgresql).
- [`externalSchemaRegistry`](#spec.externalSchemaRegistry-property){: name='spec.externalSchemaRegistry-property'} (object). ExternalSchemaRegistry configuration values. See below for [nested schema](#spec.externalSchemaRegistry).
- [`jolokia`](#spec.jolokia-property){: name='spec.jolokia-property'} (object). Jolokia configuration values. See below for [nested schema](#spec.jolokia).
- [`prometheus`](#spec.prometheus-property){: name='spec.prometheus-property'} (object). Prometheus configuration values. See below for [nested schema](#spec.prometheus).
- [`rsyslog`](#spec.rsyslog-property){: name='spec.rsyslog-property'} (object). Rsyslog configuration values. See below for [nested schema](#spec.rsyslog).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## autoscaler {: #spec.autoscaler }

_Appears on [`spec`](#spec)._

Autoscaler configuration values.

**Required**

- [`autoscaling`](#spec.autoscaler.autoscaling-property){: name='spec.autoscaler.autoscaling-property'} (array of objects, MaxItems: 64). Configure autoscaling thresholds for a service. See below for [nested schema](#spec.autoscaler.autoscaling).

### autoscaling {: #spec.autoscaler.autoscaling }

_Appears on [`spec.autoscaler`](#spec.autoscaler)._

AutoscalingProperties.

**Required**

- [`cap_gb`](#spec.autoscaler.autoscaling.cap_gb-property){: name='spec.autoscaler.autoscaling.cap_gb-property'} (integer, Minimum: 50, Maximum: 10000). The maximum total disk size (in gb) to allow autoscaler to scale up to.
- [`type`](#spec.autoscaler.autoscaling.type-property){: name='spec.autoscaler.autoscaling.type-property'} (string, Enum: `autoscale_disk`). Type of autoscale event.

## datadog {: #spec.datadog }

_Appears on [`spec`](#spec)._

Datadog configuration values.

**Required**

- [`datadog_api_key`](#spec.datadog.datadog_api_key-property){: name='spec.datadog.datadog_api_key-property'} (string, Pattern: `^[A-Za-z0-9]{1,256}$`, MinLength: 1, MaxLength: 256). Datadog API key.

**Optional**

- [`datadog_tags`](#spec.datadog.datadog_tags-property){: name='spec.datadog.datadog_tags-property'} (array of objects, MaxItems: 32). Custom tags provided by user. See below for [nested schema](#spec.datadog.datadog_tags).
- [`disable_consumer_stats`](#spec.datadog.disable_consumer_stats-property){: name='spec.datadog.disable_consumer_stats-property'} (boolean). Disable consumer group metrics.
- [`kafka_consumer_check_instances`](#spec.datadog.kafka_consumer_check_instances-property){: name='spec.datadog.kafka_consumer_check_instances-property'} (integer, Minimum: 1, Maximum: 100). Number of separate instances to fetch kafka consumer statistics with.
- [`kafka_consumer_stats_timeout`](#spec.datadog.kafka_consumer_stats_timeout-property){: name='spec.datadog.kafka_consumer_stats_timeout-property'} (integer, Minimum: 2, Maximum: 300). Number of seconds that datadog will wait to get consumer statistics from brokers.
- [`max_partition_contexts`](#spec.datadog.max_partition_contexts-property){: name='spec.datadog.max_partition_contexts-property'} (integer, Minimum: 200, Maximum: 200000). Maximum number of partition contexts to send.
- [`site`](#spec.datadog.site-property){: name='spec.datadog.site-property'} (string, Enum: `datadoghq.com`, `datadoghq.eu`, `us3.datadoghq.com`, `us5.datadoghq.com`, `ddog-gov.com`, `ap1.datadoghq.com`). Datadog intake site. Defaults to datadoghq.com.

### datadog_tags {: #spec.datadog.datadog_tags }

_Appears on [`spec.datadog`](#spec.datadog)._

Datadog tag defined by user.

**Required**

- [`tag`](#spec.datadog.datadog_tags.tag-property){: name='spec.datadog.datadog_tags.tag-property'} (string, MinLength: 1, MaxLength: 200). Tag format and usage are described here: https://docs.datadoghq.com/getting_started/tagging. Tags with prefix `aiven-` are reserved for Aiven.

**Optional**

- [`comment`](#spec.datadog.datadog_tags.comment-property){: name='spec.datadog.datadog_tags.comment-property'} (string, MaxLength: 1024). Optional tag explanation.

## externalAWSCloudwatchLogs {: #spec.externalAWSCloudwatchLogs }

_Appears on [`spec`](#spec)._

ExternalAwsCloudwatchLogs configuration values.

**Required**

- [`access_key`](#spec.externalAWSCloudwatchLogs.access_key-property){: name='spec.externalAWSCloudwatchLogs.access_key-property'} (string, MaxLength: 4096). AWS access key. Required permissions are logs:CreateLogGroup, logs:CreateLogStream, logs:PutLogEvents and logs:DescribeLogStreams.
- [`region`](#spec.externalAWSCloudwatchLogs.region-property){: name='spec.externalAWSCloudwatchLogs.region-property'} (string, MaxLength: 32). AWS region.
- [`secret_key`](#spec.externalAWSCloudwatchLogs.secret_key-property){: name='spec.externalAWSCloudwatchLogs.secret_key-property'} (string, MaxLength: 4096). AWS secret key.

**Optional**

- [`log_group_name`](#spec.externalAWSCloudwatchLogs.log_group_name-property){: name='spec.externalAWSCloudwatchLogs.log_group_name-property'} (string, Pattern: `^[\.\-_/#A-Za-z0-9]+$`, MinLength: 1, MaxLength: 512). AWS CloudWatch log group name.

## externalAWSCloudwatchMetrics {: #spec.externalAWSCloudwatchMetrics }

_Appears on [`spec`](#spec)._

ExternalAwsCloudwatchMetrics configuration values.

**Required**

- [`access_key`](#spec.externalAWSCloudwatchMetrics.access_key-property){: name='spec.externalAWSCloudwatchMetrics.access_key-property'} (string, MaxLength: 4096). AWS access key. Required permissions are cloudwatch:PutMetricData.
- [`namespace`](#spec.externalAWSCloudwatchMetrics.namespace-property){: name='spec.externalAWSCloudwatchMetrics.namespace-property'} (string, MinLength: 1, MaxLength: 255). AWS CloudWatch Metrics Namespace.
- [`region`](#spec.externalAWSCloudwatchMetrics.region-property){: name='spec.externalAWSCloudwatchMetrics.region-property'} (string, MaxLength: 32). AWS region.
- [`secret_key`](#spec.externalAWSCloudwatchMetrics.secret_key-property){: name='spec.externalAWSCloudwatchMetrics.secret_key-property'} (string, MaxLength: 4096). AWS secret key.

## externalElasticsearchLogs {: #spec.externalElasticsearchLogs }

_Appears on [`spec`](#spec)._

ExternalElasticsearchLogs configuration values.

**Required**

- [`index_prefix`](#spec.externalElasticsearchLogs.index_prefix-property){: name='spec.externalElasticsearchLogs.index_prefix-property'} (string, Pattern: `^[a-z0-9][a-z0-9-_.]+$`, MinLength: 1, MaxLength: 1000). Elasticsearch index prefix.
- [`url`](#spec.externalElasticsearchLogs.url-property){: name='spec.externalElasticsearchLogs.url-property'} (string, MinLength: 12, MaxLength: 2048). Elasticsearch connection URL.

**Optional**

- [`ca`](#spec.externalElasticsearchLogs.ca-property){: name='spec.externalElasticsearchLogs.ca-property'} (string, MaxLength: 16384). PEM encoded CA certificate.
- [`index_days_max`](#spec.externalElasticsearchLogs.index_days_max-property){: name='spec.externalElasticsearchLogs.index_days_max-property'} (integer, Minimum: 1, Maximum: 10000). Maximum number of days of logs to keep.
- [`timeout`](#spec.externalElasticsearchLogs.timeout-property){: name='spec.externalElasticsearchLogs.timeout-property'} (number, Minimum: 10, Maximum: 120). Elasticsearch request timeout limit.

## externalGoogleCloudBigquery {: #spec.externalGoogleCloudBigquery }

_Appears on [`spec`](#spec)._

ExternalGoogleCloudBigquery configuration values.

**Required**

- [`project_id`](#spec.externalGoogleCloudBigquery.project_id-property){: name='spec.externalGoogleCloudBigquery.project_id-property'} (string, MinLength: 6, MaxLength: 30). GCP project id.
- [`service_account_credentials`](#spec.externalGoogleCloudBigquery.service_account_credentials-property){: name='spec.externalGoogleCloudBigquery.service_account_credentials-property'} (string, MaxLength: 4096). This is a JSON object with the fields documented in https://cloud.google.com/iam/docs/creating-managing-service-account-keys .

## externalGoogleCloudLogging {: #spec.externalGoogleCloudLogging }

_Appears on [`spec`](#spec)._

ExternalGoogleCloudLogging configuration values.

**Required**

- [`log_id`](#spec.externalGoogleCloudLogging.log_id-property){: name='spec.externalGoogleCloudLogging.log_id-property'} (string, MaxLength: 512). Google Cloud Logging log id.
- [`project_id`](#spec.externalGoogleCloudLogging.project_id-property){: name='spec.externalGoogleCloudLogging.project_id-property'} (string, MinLength: 6, MaxLength: 30). GCP project id.
- [`service_account_credentials`](#spec.externalGoogleCloudLogging.service_account_credentials-property){: name='spec.externalGoogleCloudLogging.service_account_credentials-property'} (string, MaxLength: 4096). This is a JSON object with the fields documented in https://cloud.google.com/iam/docs/creating-managing-service-account-keys .

## externalKafka {: #spec.externalKafka }

_Appears on [`spec`](#spec)._

ExternalKafka configuration values.

**Required**

- [`bootstrap_servers`](#spec.externalKafka.bootstrap_servers-property){: name='spec.externalKafka.bootstrap_servers-property'} (string, MinLength: 3, MaxLength: 256). Bootstrap servers.
- [`security_protocol`](#spec.externalKafka.security_protocol-property){: name='spec.externalKafka.security_protocol-property'} (string, Enum: `PLAINTEXT`, `SSL`, `SASL_PLAINTEXT`, `SASL_SSL`). Security protocol.

**Optional**

- [`sasl_mechanism`](#spec.externalKafka.sasl_mechanism-property){: name='spec.externalKafka.sasl_mechanism-property'} (string, Enum: `PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512`). SASL mechanism used for connections to the Kafka server.
- [`sasl_plain_password`](#spec.externalKafka.sasl_plain_password-property){: name='spec.externalKafka.sasl_plain_password-property'} (string, MinLength: 1, MaxLength: 256). Password for SASL PLAIN mechanism in the Kafka server.
- [`sasl_plain_username`](#spec.externalKafka.sasl_plain_username-property){: name='spec.externalKafka.sasl_plain_username-property'} (string, MinLength: 1, MaxLength: 256). Username for SASL PLAIN mechanism in the Kafka server.
- [`ssl_ca_cert`](#spec.externalKafka.ssl_ca_cert-property){: name='spec.externalKafka.ssl_ca_cert-property'} (string, MaxLength: 16384). PEM-encoded CA certificate.
- [`ssl_client_cert`](#spec.externalKafka.ssl_client_cert-property){: name='spec.externalKafka.ssl_client_cert-property'} (string, MaxLength: 16384). PEM-encoded client certificate.
- [`ssl_client_key`](#spec.externalKafka.ssl_client_key-property){: name='spec.externalKafka.ssl_client_key-property'} (string, MaxLength: 16384). PEM-encoded client key.
- [`ssl_endpoint_identification_algorithm`](#spec.externalKafka.ssl_endpoint_identification_algorithm-property){: name='spec.externalKafka.ssl_endpoint_identification_algorithm-property'} (string, Enum: `https`). The endpoint identification algorithm to validate server hostname using server certificate.

## externalOpensearchLogs {: #spec.externalOpensearchLogs }

_Appears on [`spec`](#spec)._

ExternalOpensearchLogs configuration values.

**Required**

- [`index_prefix`](#spec.externalOpensearchLogs.index_prefix-property){: name='spec.externalOpensearchLogs.index_prefix-property'} (string, Pattern: `^[a-z0-9][a-z0-9-_.]+$`, MinLength: 1, MaxLength: 1000). OpenSearch index prefix.
- [`url`](#spec.externalOpensearchLogs.url-property){: name='spec.externalOpensearchLogs.url-property'} (string, MinLength: 12, MaxLength: 2048). OpenSearch connection URL.

**Optional**

- [`ca`](#spec.externalOpensearchLogs.ca-property){: name='spec.externalOpensearchLogs.ca-property'} (string, MaxLength: 16384). PEM encoded CA certificate.
- [`index_days_max`](#spec.externalOpensearchLogs.index_days_max-property){: name='spec.externalOpensearchLogs.index_days_max-property'} (integer, Minimum: 1, Maximum: 10000). Maximum number of days of logs to keep.
- [`timeout`](#spec.externalOpensearchLogs.timeout-property){: name='spec.externalOpensearchLogs.timeout-property'} (number, Minimum: 10, Maximum: 120). OpenSearch request timeout limit.

## externalPostgresql {: #spec.externalPostgresql }

_Appears on [`spec`](#spec)._

ExternalPostgresql configuration values.

**Required**

- [`host`](#spec.externalPostgresql.host-property){: name='spec.externalPostgresql.host-property'} (string, MaxLength: 255). Hostname or IP address of the server.
- [`port`](#spec.externalPostgresql.port-property){: name='spec.externalPostgresql.port-property'} (integer, Minimum: 1, Maximum: 65535). Port number of the server.
- [`username`](#spec.externalPostgresql.username-property){: name='spec.externalPostgresql.username-property'} (string, MaxLength: 256). User name.

**Optional**

- [`default_database`](#spec.externalPostgresql.default_database-property){: name='spec.externalPostgresql.default_database-property'} (string, Pattern: `^[_A-Za-z0-9][-_A-Za-z0-9]{0,62}$`, MaxLength: 63). Default database.
- [`password`](#spec.externalPostgresql.password-property){: name='spec.externalPostgresql.password-property'} (string, MaxLength: 256). Password.
- [`ssl_client_certificate`](#spec.externalPostgresql.ssl_client_certificate-property){: name='spec.externalPostgresql.ssl_client_certificate-property'} (string, MaxLength: 16384). Client certificate.
- [`ssl_client_key`](#spec.externalPostgresql.ssl_client_key-property){: name='spec.externalPostgresql.ssl_client_key-property'} (string, MaxLength: 16384). Client key.
- [`ssl_mode`](#spec.externalPostgresql.ssl_mode-property){: name='spec.externalPostgresql.ssl_mode-property'} (string, Enum: `require`, `verify-ca`, `verify-full`). SSL mode to use for the connection.  Please note that Aiven requires TLS for all connections to external PostgreSQL services.
- [`ssl_root_cert`](#spec.externalPostgresql.ssl_root_cert-property){: name='spec.externalPostgresql.ssl_root_cert-property'} (string, MaxLength: 16384). SSL Root Cert.

## externalSchemaRegistry {: #spec.externalSchemaRegistry }

_Appears on [`spec`](#spec)._

ExternalSchemaRegistry configuration values.

**Required**

- [`authentication`](#spec.externalSchemaRegistry.authentication-property){: name='spec.externalSchemaRegistry.authentication-property'} (string, Enum: `none`, `basic`). Authentication method.
- [`url`](#spec.externalSchemaRegistry.url-property){: name='spec.externalSchemaRegistry.url-property'} (string, MaxLength: 2048). Schema Registry URL.

**Optional**

- [`basic_auth_password`](#spec.externalSchemaRegistry.basic_auth_password-property){: name='spec.externalSchemaRegistry.basic_auth_password-property'} (string, MaxLength: 256). Basic authentication password.
- [`basic_auth_username`](#spec.externalSchemaRegistry.basic_auth_username-property){: name='spec.externalSchemaRegistry.basic_auth_username-property'} (string, MaxLength: 256). Basic authentication user name.

## jolokia {: #spec.jolokia }

_Appears on [`spec`](#spec)._

Jolokia configuration values.

**Optional**

- [`basic_auth_password`](#spec.jolokia.basic_auth_password-property){: name='spec.jolokia.basic_auth_password-property'} (string, MinLength: 8, MaxLength: 64). Jolokia basic authentication password.
- [`basic_auth_username`](#spec.jolokia.basic_auth_username-property){: name='spec.jolokia.basic_auth_username-property'} (string, Pattern: `^[a-z0-9\-@_]{5,32}$`, MinLength: 5, MaxLength: 32). Jolokia basic authentication username.

## prometheus {: #spec.prometheus }

_Appears on [`spec`](#spec)._

Prometheus configuration values.

**Optional**

- [`basic_auth_password`](#spec.prometheus.basic_auth_password-property){: name='spec.prometheus.basic_auth_password-property'} (string, MinLength: 8, MaxLength: 64). Prometheus basic authentication password.
- [`basic_auth_username`](#spec.prometheus.basic_auth_username-property){: name='spec.prometheus.basic_auth_username-property'} (string, Pattern: `^[a-z0-9\-@_]{5,32}$`, MinLength: 5, MaxLength: 32). Prometheus basic authentication username.

## rsyslog {: #spec.rsyslog }

_Appears on [`spec`](#spec)._

Rsyslog configuration values.

**Required**

- [`format`](#spec.rsyslog.format-property){: name='spec.rsyslog.format-property'} (string, Enum: `rfc5424`, `rfc3164`, `custom`). Message format.
- [`port`](#spec.rsyslog.port-property){: name='spec.rsyslog.port-property'} (integer, Minimum: 1, Maximum: 65535). Rsyslog server port.
- [`server`](#spec.rsyslog.server-property){: name='spec.rsyslog.server-property'} (string, MinLength: 4, MaxLength: 255). Rsyslog server IP address or hostname.
- [`tls`](#spec.rsyslog.tls-property){: name='spec.rsyslog.tls-property'} (boolean). Require TLS.

**Optional**

- [`ca`](#spec.rsyslog.ca-property){: name='spec.rsyslog.ca-property'} (string, MaxLength: 16384). PEM encoded CA certificate.
- [`cert`](#spec.rsyslog.cert-property){: name='spec.rsyslog.cert-property'} (string, MaxLength: 16384). PEM encoded client certificate.
- [`key`](#spec.rsyslog.key-property){: name='spec.rsyslog.key-property'} (string, MaxLength: 16384). PEM encoded client key.
- [`logline`](#spec.rsyslog.logline-property){: name='spec.rsyslog.logline-property'} (string, Pattern: `^[ -~\t]+$`, MinLength: 1, MaxLength: 512). Custom syslog message format.
- [`max_message_size`](#spec.rsyslog.max_message_size-property){: name='spec.rsyslog.max_message_size-property'} (integer, Minimum: 2048, Maximum: 2147483647). Rsyslog max message size.
- [`sd`](#spec.rsyslog.sd-property){: name='spec.rsyslog.sd-property'} (string, MaxLength: 1024). Structured data block for log message.
