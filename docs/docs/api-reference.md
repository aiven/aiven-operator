---
title: "API Reference"
linkTitle: "API Reference"
weight: 90
---

## aiven.io/v1alpha1

### AuthSecretReference 

AuthSecretReference references a Secret containing an Aiven authentication token  

| Field | Description|
|---|---|
|`name` <br>string|N/A|
|`key` <br>string|N/A| 

### Cassandra 

Cassandra is the Schema for the cassandras API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[CassandraSpec](#cassandraspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 

### CassandraSpec 

CassandraSpec defines the desired state of Cassandra  

| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/cassandra.CassandraUserConfig|Cassandra specific user configuration options| 

### Clickhouse 

Clickhouse is the Schema for the clickhouses API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[ClickhouseSpec](#clickhousespec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 

### ClickhouseSpec 

ClickhouseSpec defines the desired state of Clickhouse  

| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/clickhouse.ClickhouseUserConfig|OpenSearch specific user configuration options| 

### ClickhouseUser 

ClickhouseUser is the Schema for the clickhouseusers API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[ClickhouseUserSpec](#clickhouseuserspec)|N/A|
|`status` <br>[ClickhouseUserStatus](#clickhouseuserstatus)|N/A| 

### ClickhouseUserSpec 

ClickhouseUserSpec defines the desired state of ClickhouseUser  

| Field | Description|
|---|---|
|`project` <br>string|Project to link the user to|
|`serviceName` <br>string|Service to link the user to|
|`authentication` <br>string|Authentication details|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### ClickhouseUserStatus 

ClickhouseUserStatus defines the observed state of ClickhouseUser  

| Field | Description|
|---|---|
|`uuid` <br>string|Clickhouse user UUID|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an ClickhouseUser state| 

### ConnInfoSecretTarget 

ConnInfoSecretTarget contains information secret name  

| Field | Description|
|---|---|
|`name` <br>string|Name of the Secret resource to be created| 

### ConnectionPool 

ConnectionPool is the Schema for the connectionpools API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[ConnectionPoolSpec](#connectionpoolspec)|N/A|
|`status` <br>[ConnectionPoolStatus](#connectionpoolstatus)|N/A| 

### ConnectionPoolSpec 

ConnectionPoolSpec defines the desired state of ConnectionPool  

| Field | Description|
|---|---|
|`project` <br>string|Target project.|
|`serviceName` <br>string|Service name.|
|`databaseName` <br>string|Name of the database the pool connects to|
|`username` <br>string|Name of the service user used to connect to the database|
|`poolSize` <br>int|Number of connections the pool may create towards the backend server|
|`poolMode` <br>string|Mode the pool operates in (session, transaction, statement)|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### ConnectionPoolStatus 

ConnectionPoolStatus defines the observed state of ConnectionPool  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an ConnectionPool state| 

### Database 

Database is the Schema for the databases API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[DatabaseSpec](#databasespec)|N/A|
|`status` <br>[DatabaseStatus](#databasestatus)|N/A| 

### DatabaseSpec 

DatabaseSpec defines the desired state of Database  

| Field | Description|
|---|---|
|`project` <br>string|Project to link the database to|
|`serviceName` <br>string|PostgreSQL service to link the database to|
|`lcCollate` <br>string|Default string sort order (LC_COLLATE) of the database. Default value: en_US.UTF-8|
|`lcCtype` <br>string|Default character classification (LC_CTYPE) of the database. Default value: en_US.UTF-8|
|`terminationProtection` <br>bool|It is a Kubernetes side deletion protections, which prevents the databasefrom being deleted by Kubernetes. It is recommended to enable this for any production databases containing critical data. |
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### DatabaseStatus 

DatabaseStatus defines the observed state of Database  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an Database state| 

### Grafana 

Grafana is the Schema for the grafanas API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[GrafanaSpec](#grafanaspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 

### GrafanaSpec 

GrafanaSpec defines the desired state of Grafana  

| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/grafana.GrafanaUserConfig|Cassandra specific user configuration options| 

### Kafka 

Kafka is the Schema for the kafkas API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[KafkaSpec](#kafkaspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 

### KafkaACL 

KafkaACL is the Schema for the kafkaacls API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[KafkaACLSpec](#kafkaaclspec)|N/A|
|`status` <br>[KafkaACLStatus](#kafkaaclstatus)|N/A| 

### KafkaACLSpec 

KafkaACLSpec defines the desired state of KafkaACL  

| Field | Description|
|---|---|
|`project` <br>string|Project to link the Kafka ACL to|
|`serviceName` <br>string|Service to link the Kafka ACL to|
|`permission` <br>string|Kafka permission to grant (admin, read, readwrite, write)|
|`topic` <br>string|Topic name pattern for the ACL entry|
|`username` <br>string|Username pattern for the ACL entry|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### KafkaACLStatus 

KafkaACLStatus defines the observed state of KafkaACL  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an KafkaACL state|
|`id` <br>string|Kafka ACL ID| 

### KafkaConnect 

KafkaConnect is the Schema for the kafkaconnects API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[KafkaConnectSpec](#kafkaconnectspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 

### KafkaConnectSpec 

KafkaConnectSpec defines the desired state of KafkaConnect  

| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`userConfig` <br>github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/kafka_connect.KafkaConnectUserConfig|PostgreSQL specific user configuration options| 

### KafkaConnector 

KafkaConnector is the Schema for the kafkaconnectors API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[KafkaConnectorSpec](#kafkaconnectorspec)|N/A|
|`status` <br>[KafkaConnectorStatus](#kafkaconnectorstatus)|N/A| 

### KafkaConnectorPluginStatus 

KafkaConnectorPluginStatus describes the observed state of a Kafka Connector Plugin  

| Field | Description|
|---|---|
|`author` <br>string|N/A|
|`class` <br>string|N/A|
|`docUrl` <br>string|N/A|
|`title` <br>string|N/A|
|`type` <br>string|N/A|
|`version` <br>string|N/A| 

### KafkaConnectorSpec 

KafkaConnectorSpec defines the desired state of KafkaConnector  

| Field | Description|
|---|---|
|`project` <br>string|Target project.|
|`serviceName` <br>string|Service name.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connectorClass` <br>string|The Java class of the connector.|
|`userConfig` <br>map[string]string|The connector specific configurationTo build config values from secret the template function `{{ fromSecret &#34;name&#34; &#34;key&#34; }}` is provided when interpreting the keys | 

### KafkaConnectorStatus 

KafkaConnectorStatus defines the observed state of KafkaConnector  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an kafka connector state|
|`state` <br>string|Connector state|
|`pluginStatus` <br>[KafkaConnectorPluginStatus](#kafkaconnectorpluginstatus)|PluginStatus contains metadata about the configured connector plugin|
|`tasksStatus` <br>[KafkaConnectorTasksStatus](#kafkaconnectortasksstatus)|TasksStatus contains metadata about the running tasks| 

### KafkaConnectorTasksStatus 

KafkaConnectorTasksStatus describes the observed state of the Kafka Connector Tasks  

| Field | Description|
|---|---|
|`total` <br>uint|N/A|
|`running` <br>uint|N/A|
|`failed` <br>uint|N/A|
|`paused` <br>uint|N/A|
|`unassigned` <br>uint|N/A|
|`unknown` <br>uint|N/A|
|`stackTrace` <br>string|N/A| 

### KafkaSchema 

KafkaSchema is the Schema for the kafkaschemas API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[KafkaSchemaSpec](#kafkaschemaspec)|N/A|
|`status` <br>[KafkaSchemaStatus](#kafkaschemastatus)|N/A| 

### KafkaSchemaSpec 

KafkaSchemaSpec defines the desired state of KafkaSchema  

| Field | Description|
|---|---|
|`project` <br>string|Project to link the Kafka Schema to|
|`serviceName` <br>string|Service to link the Kafka Schema to|
|`subjectName` <br>string|Kafka Schema Subject name|
|`schema` <br>string|Kafka Schema configuration should be a valid Avro Schema JSON format|
|`compatibilityLevel` <br>string|Kafka Schemas compatibility level|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### KafkaSchemaStatus 

KafkaSchemaStatus defines the observed state of KafkaSchema  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an KafkaSchema state|
|`version` <br>int|Kafka Schema configuration version| 

### KafkaSpec 

KafkaSpec defines the desired state of Kafka  

| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`karapace` <br>bool|Switch the service to use Karapace for schema registry and REST proxy|
|`userConfig` <br>github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/kafka.KafkaUserConfig|Kafka specific user configuration options| 

### KafkaTopic 

KafkaTopic is the Schema for the kafkatopics API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[KafkaTopicSpec](#kafkatopicspec)|N/A|
|`status` <br>[KafkaTopicStatus](#kafkatopicstatus)|N/A| 

### KafkaTopicConfig 

| Field | Description|
|---|---|
|`cleanup_policy` <br>string|cleanup.policy value|
|`compression_type` <br>string|compression.type value|
|`delete_retention_ms` <br>int64|delete.retention.ms value|
|`file_delete_delay_ms` <br>int64|file.delete.delay.ms value|
|`flush_messages` <br>int64|flush.messages value|
|`flush_ms` <br>int64|flush.ms value|
|`index_interval_bytes` <br>int64|index.interval.bytes value|
|`max_compaction_lag_ms` <br>int64|max.compaction.lag.ms value|
|`max_message_bytes` <br>int64|max.message.bytes value|
|`message_downconversion_enable` <br>bool|message.downconversion.enable value|
|`message_format_version` <br>string|message.format.version value|
|`message_timestamp_difference_max_ms` <br>int64|message.timestamp.difference.max.ms value|
|`message_timestamp_type` <br>string|message.timestamp.type value|
|`min_cleanable_dirty_ratio` <br>float64|min.cleanable.dirty.ratio value|
|`min_compaction_lag_ms` <br>int64|min.compaction.lag.ms value|
|`min_insync_replicas` <br>int64|min.insync.replicas value|
|`preallocate` <br>bool|preallocate value|
|`retention_bytes` <br>int64|retention.bytes value|
|`retention_ms` <br>int64|retention.ms value|
|`segment_bytes` <br>int64|segment.bytes value|
|`segment_index_bytes` <br>int64|segment.index.bytes value|
|`segment_jitter_ms` <br>int64|segment.jitter.ms value|
|`segment_ms` <br>int64|segment.ms value|
|`unclean_leader_election_enable` <br>bool|unclean.leader.election.enable value| 

### KafkaTopicSpec 

KafkaTopicSpec defines the desired state of KafkaTopic  

| Field | Description|
|---|---|
|`project` <br>string|Target project.|
|`serviceName` <br>string|Service name.|
|`partitions` <br>int|Number of partitions to create in the topic|
|`replication` <br>int|Replication factor for the topic|
|`tags` <br>[[]KafkaTopicTag](#kafkatopictag)|Kafka topic tags|
|`config` <br>[KafkaTopicConfig](#kafkatopicconfig)|Kafka topic configuration|
|`termination_protection` <br>bool|It is a Kubernetes side deletion protections, which prevents the kafka topicfrom being deleted by Kubernetes. It is recommended to enable this for any production databases containing critical data. |
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### KafkaTopicStatus 

KafkaTopicStatus defines the observed state of KafkaTopic  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an KafkaTopic state|
|`state` <br>string|State represents the state of the kafka topic| 

### KafkaTopicTag 

| Field | Description|
|---|---|
|`key` <br>string|N/A|
|`value` <br>string|N/A| 

### MySQL 

MySQL is the Schema for the mysqls API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[MySQLSpec](#mysqlspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 

### MySQLSpec 

MySQLSpec defines the desired state of MySQL  

| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/mysql.MysqlUserConfig|MySQL specific user configuration options| 

### OpenSearch 

OpenSearch is the Schema for the opensearches API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[OpenSearchSpec](#opensearchspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 

### OpenSearchSpec 

OpenSearchSpec defines the desired state of OpenSearch  

| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/opensearch.OpensearchUserConfig|OpenSearch specific user configuration options| 

### PostgreSQL 

PostgreSQL is the Schema for the postgresql API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[PostgreSQLSpec](#postgresqlspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 

### PostgreSQLSpec 

PostgreSQLSpec defines the desired state of postgres instance  

| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/pg.PgUserConfig|PostgreSQL specific user configuration options| 

### Project 

Project is the Schema for the projects API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[ProjectSpec](#projectspec)|N/A|
|`status` <br>[ProjectStatus](#projectstatus)|N/A| 

### ProjectSpec 

ProjectSpec defines the desired state of Project  

| Field | Description|
|---|---|
|`cardId` <br>string|Credit card ID; The ID may be either last 4 digits of the card or the actual ID|
|`accountId` <br>string|Account ID|
|`billingAddress` <br>string|Billing name and address of the project|
|`billingEmails` <br>[]string|Billing contact emails of the project|
|`billingCurrency` <br>string|Billing currency|
|`billingExtraText` <br>string|Extra text to be included in all project invoices, e.g. purchase order or cost center number|
|`billingGroupId` <br>string|BillingGroup ID|
|`countryCode` <br>string|Billing country code of the project|
|`cloud` <br>string|Target cloud, example: aws-eu-central-1|
|`copyFromProject` <br>string|Project name from which to copy settings to the new project|
|`technicalEmails` <br>[]string|Technical contact emails of the project|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`tags` <br>map[string]string|Tags are key-value pairs that allow you to categorize projects|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### ProjectStatus 

ProjectStatus defines the observed state of Project  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an Project state|
|`vatId` <br>string|EU VAT Identification Number|
|`availableCredits` <br>string|Available credirs|
|`country` <br>string|Country name|
|`estimatedBalance` <br>string|Estimated balance|
|`paymentMethod` <br>string|Payment method name| 

### ProjectVPC 

ProjectVPC is the Schema for the projectvpcs API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[ProjectVPCSpec](#projectvpcspec)|N/A|
|`status` <br>[ProjectVPCStatus](#projectvpcstatus)|N/A| 

### ProjectVPCSpec 

ProjectVPCSpec defines the desired state of ProjectVPC  

| Field | Description|
|---|---|
|`project` <br>string|The project the VPC belongs to|
|`cloudName` <br>string|Cloud the VPC is in|
|`networkCidr` <br>string|Network address range used by the VPC like 192.168.0.0/24|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### ProjectVPCStatus 

ProjectVPCStatus defines the observed state of ProjectVPC  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an ProjectVPC state|
|`state` <br>string|State of VPC|
|`id` <br>string|Project VPC id| 

### Redis 

Redis is the Schema for the redis API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[RedisSpec](#redisspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 

### RedisSpec 

RedisSpec defines the desired state of Redis  

| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/redis.RedisUserConfig|Redis specific user configuration options| 

### ResourceReference 

ResourceReference is a generic reference to another resource.Resource referring to another (dependency) won&#39;t start reconciliation until dependency is not ready   

| Field | Description|
|---|---|
|`name` <br>string|N/A|
|`namespace` <br>string|N/A| 

### ResourceReferenceObject 

ResourceReferenceObject is a composite &#34;key&#34; to resourceGroupVersionKind is for resource &#34;type&#34;: GroupVersionKind{Group: &#34;aiven.io&#34;, Version: &#34;v1alpha1&#34;, Kind: &#34;Kafka&#34;} NamespacedName is for specific instance: NamespacedName{Name: &#34;my-kafka&#34;, Namespace: &#34;default&#34;}   

| Field | Description|
|---|---|
|`GroupVersionKind` <br>[k8s.io/apimachinery/pkg/runtime/schema.GroupVersionKind](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#groupversionkind-schema-runtime)|N/A|
|`NamespacedName` <br>[k8s.io/apimachinery/pkg/types.NamespacedName](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#namespacedname-types-pkg)|N/A| 

### ServiceCommonSpec 

| Field | Description|
|---|---|
|`project` <br>string|Target project.|
|`plan` <br>string|Subscription plan.|
|`cloudName` <br>string|Cloud the service runs in.|
|`projectVpcId` <br>string|Identifier of the VPC the service should be in, if any.|
|`projectVPCRef` <br>[ResourceReference](#resourcereference)|ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically|
|`maintenanceWindowDow` <br>string|Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.|
|`maintenanceWindowTime` <br>string|Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.|
|`terminationProtection` <br>bool|Prevent service from being deleted. It is recommended to have this enabled for all services.|
|`tags` <br>map[string]string|Tags are key-value pairs that allow you to categorize services.|
|`serviceIntegrations` <br>[[]*./api/v1alpha1.ServiceIntegrationItem](#aiven.io/v1alpha1.*./api/v1alpha1.ServiceIntegrationItem)|N/A| 

### ServiceIntegration 

ServiceIntegration is the Schema for the serviceintegrations API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[ServiceIntegrationSpec](#serviceintegrationspec)|N/A|
|`status` <br>[ServiceIntegrationStatus](#serviceintegrationstatus)|N/A| 

### ServiceIntegrationDatadogUserConfig 

| Field | Description|
|---|---|
|`exclude_consumer_groups` <br>[]string|Consumer groups to exclude|
|`exclude_topics` <br>[]string|List of topics to exclude|
|`include_consumer_groups` <br>[]string|Consumer groups to include|
|`include_topics` <br>[]string|Topics to include|
|`kafka_custom_metrics` <br>[]string|List of custom metrics| 

### ServiceIntegrationItem 

ServiceIntegrationItem Service integrations to specify when creating a service. Not applied after initial service creation  

| Field | Description|
|---|---|
|`integrationType` <br>string|N/A|
|`sourceServiceName` <br>string|N/A| 

### ServiceIntegrationKafkaConnect 

| Field | Description|
|---|---|
|`config_storage_topic` <br>string|The name of the topic where connector and task configuration data are stored. This must be the same for all workers with the same group_id.|
|`group_id` <br>string|A unique string that identifies the Connect cluster group this worker belongs to.|
|`offset_storage_topic` <br>string|The name of the topic where connector and task configuration offsets are stored. This must be the same for all workers with the same group_id.|
|`status_storage_topic` <br>string|The name of the topic where connector and task configuration status updates are stored.This must be the same for all workers with the same group_id.| 

### ServiceIntegrationKafkaConnectUserConfig 

| Field | Description|
|---|---|
|`kafka_connect` <br>[ServiceIntegrationKafkaConnect](#serviceintegrationkafkaconnect)|N/A| 

### ServiceIntegrationKafkaLogsUserConfig 

| Field | Description|
|---|---|
|`kafka_topic` <br>string|Topic name| 

### ServiceIntegrationMetricsUserConfig 

| Field | Description|
|---|---|
|`database` <br>string|Name of the database where to store metric datapoints. Only affects PostgreSQL destinations|
|`retention_days` <br>int|Number of days to keep old metrics. Only affects PostgreSQL destinations. Set to 0 for no automatic cleanup. Defaults to 30 days.|
|`ro_username` <br>string|Name of a user that can be used to read metrics. This will be used for Grafana integration (if enabled) to prevent Grafana users from making undesired changes. Only affects PostgreSQL destinations. Defaults to &#39;metrics_reader&#39;. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service.|
|`username` <br>string|Name of the user used to write metrics. Only affects PostgreSQL destinations. Defaults to &#39;metrics_writer&#39;. Note that this must be the same for all metrics integrations that write data to the same PostgreSQL service.| 

### ServiceIntegrationSpec 

ServiceIntegrationSpec defines the desired state of ServiceIntegration  

| Field | Description|
|---|---|
|`project` <br>string|Project the integration belongs to|
|`integrationType` <br>string|Type of the service integration|
|`sourceEndpointID` <br>string|Source endpoint for the integration (if any)|
|`sourceServiceName` <br>string|Source service for the integration (if any)|
|`destinationEndpointId` <br>string|Destination endpoint for the integration (if any)|
|`destinationServiceName` <br>string|Destination service for the integration (if any)|
|`datadog` <br>[ServiceIntegrationDatadogUserConfig](#serviceintegrationdatadoguserconfig)|Datadog specific user configuration options|
|`kafkaConnect` <br>[ServiceIntegrationKafkaConnectUserConfig](#serviceintegrationkafkaconnectuserconfig)|Kafka Connect service configuration values|
|`kafkaLogs` <br>[ServiceIntegrationKafkaLogsUserConfig](#serviceintegrationkafkalogsuserconfig)|Kafka logs configuration values|
|`metrics` <br>[ServiceIntegrationMetricsUserConfig](#serviceintegrationmetricsuserconfig)|Metrics configuration values|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### ServiceIntegrationStatus 

ServiceIntegrationStatus defines the observed state of ServiceIntegration  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an ServiceIntegration state|
|`id` <br>string|Service integration ID| 

### ServiceStatus 

ServiceStatus defines the observed state of service  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of a service state|
|`state` <br>string|Service state| 

### ServiceUser 

ServiceUser is the Schema for the serviceusers API  

| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[ServiceUserSpec](#serviceuserspec)|N/A|
|`status` <br>[ServiceUserStatus](#serviceuserstatus)|N/A| 

### ServiceUserSpec 

ServiceUserSpec defines the desired state of ServiceUser  

| Field | Description|
|---|---|
|`project` <br>string|Project to link the user to|
|`serviceName` <br>string|Service to link the user to|
|`authentication` <br>string|Authentication details|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret| 

### ServiceUserStatus 

ServiceUserStatus defines the observed state of ServiceUser  

| Field | Description|
|---|---|
|`conditions` <br>[[]Kubernetes meta/v1.Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#condition-v1-meta)|Conditions represent the latest available observations of an ServiceUser state|
|`type` <br>string|Type of the user account| 

## References

Generated with [gen-crd-api-reference-docs](https://github.com/ahmetb/gen-crd-api-reference-docs)  on git commit `f7e2b8a`.
