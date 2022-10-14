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
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>[ClickhouseUserConfig](#clickhouseuserconfig)|OpenSearch specific user configuration options| 
### ClickhouseUser 
ClickhouseUser is the Schema for the clickhouseusers API  
| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[ClickhouseUserSpec](#clickhouseuserspec)|N/A|
|`status` <br>[ClickhouseUserStatus](#clickhouseuserstatus)|N/A| 
### ClickhouseUserConfig 
| Field | Description|
|---|---|
|`ip_filter` <br>[]string|Glob pattern and number of indexes matching that pattern to be kept Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like &#39;logs.?&#39; and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note &#39;logs.?&#39; does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.IP filter Allow incoming connections from CIDR address block, e.g. &#39;10.20.0.0/16&#39; |
|`project_to_fork_from` <br>string|Name of another project to fork a service from. This has effect only when a new service is being created.|
|`service_to_fork_from` <br>string|Name of another service to fork from. This has effect only when a new service is being created.| 
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
### KafkServiceKafkaConnectUserConfig 
| Field | Description|
|---|---|
|`consumer_max_poll_records` <br>int64|The maximum number of records returned by a single poll The maximum number of records returned in a single call to poll() (defaults to 500).|
|`offset_flush_timeout_ms` <br>int64|Offset flush timeout Maximum number of milliseconds to wait for records to flush and partition offset data to be committed to offset storage before cancelling the process and restoring the offset data to be committed in a future attempt (defaults to 5000).|
|`connector_client_config_override_policy` <br>string|Client config override policy Defines what client configurations can be overridden by the connector. Default is None|
|`consumer_fetch_max_bytes` <br>int64|The maximum amount of data the server should return for a fetch request Records are fetched in batches by the consumer, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that the consumer can make progress. As such, this is not a absolute maximum.|
|`consumer_max_poll_interval_ms` <br>int64|The maximum delay between polls when using consumer group management The maximum delay in milliseconds between invocations of poll() when using consumer group management (defaults to 300000).|
|`offset_flush_interval_ms` <br>int64|The interval at which to try committing offsets for tasks The interval at which to try committing offsets for tasks (defaults to 60000).|
|`producer_max_request_size` <br>int64|The maximum size of a request in bytes This setting will limit the number of record batches the producer will send in a single request to avoid sending huge requests.|
|`session_timeout_ms` <br>int64|The timeout used to detect failures when using Kafka’s group management facilities The timeout in milliseconds used to detect failures when using Kafka’s group management facilities (defaults to 10000).|
|`consumer_auto_offset_reset` <br>string|Consumer auto offset reset What to do when there is no initial offset in Kafka or if the current offset does not exist any more on the server. Default is earliest|
|`consumer_isolation_level` <br>string|Consumer isolation level Transaction read isolation level. read_uncommitted is the default, but read_committed can be used if consume-exactly-once behavior is desired.|
|`consumer_max_partition_fetch_bytes` <br>int64|The maximum amount of data per-partition the server will return. Records are fetched in batches by the consumer.If the first record batch in the first non-empty partition of the fetch is larger than this limit, the batch will still be returned to ensure that the consumer can make progress.| 
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
### KafkaAuthenticationMethodsUserConfig 
| Field | Description|
|---|---|
|`certificate` <br>bool|Enable certificate/SSL authentication|
|`sasl` <br>bool|Enable SASL authentication| 
### KafkaConnect 
KafkaConnect is the Schema for the kafkaconnects API  
| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[KafkaConnectSpec](#kafkaconnectspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 
### KafkaConnectPrivateAccessUserConfig 
| Field | Description|
|---|---|
|`kafka_connect` <br>bool|Allow clients to connect to kafka_connect with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations|
|`prometheus` <br>bool|Allow clients to connect to prometheus with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations| 
### KafkaConnectPublicAccessUserConfig 
| Field | Description|
|---|---|
|`kafka_connect` <br>bool|Allow clients to connect to kafka_connect from the public internet for service nodes that are in a project VPC or another type of private network|
|`prometheus` <br>bool|Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network| 
### KafkaConnectSpec 
KafkaConnectSpec defines the desired state of KafkaConnect  
| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`userConfig` <br>[KafkaConnectUserConfig](#kafkaconnectuserconfig)|PostgreSQL specific user configuration options| 
### KafkaConnectUserConfig 
| Field | Description|
|---|---|
|`connector_client_config_override_policy` <br>string|Defines what client configurations can be overridden by the connector. Default is None|
|`consumer_auto_offset_reset` <br>string|What to do when there is no initial offset in Kafka or if the current offset does not exist any more on the server. Default is earliest|
|`consumer_fetch_max_bytes` <br>int64|Records are fetched in batches by the consumer, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that the consumer can make progress. As such, this is not a absolute maximum.|
|`consumer_isolation_level` <br>string|Transaction read isolation level. read_uncommitted is the default, but read_committed can be used if consume-exactly-once behavior is desired.|
|`consumer_max_partition_fetch_bytes` <br>int64|Records are fetched in batches by the consumer.If the first record batch in the first non-empty partition of the fetch is larger than this limit, the batch will still be returned to ensure that the consumer can make progress.|
|`consumer_max_poll_interval_ms` <br>int64|The maximum delay in milliseconds between invocations of poll() when using consumer group management (defaults to 300000).|
|`consumer_max_poll_records` <br>int64|The maximum number of records returned in a single call to poll() (defaults to 500).|
|`offset_flush_interval_ms` <br>int64|The interval at which to try committing offsets for tasks (defaults to 60000).|
|`producer_max_request_size` <br>int64|This setting will limit the number of record batches the producer will send in a single request to avoid sending huge requests.|
|`session_timeout_ms` <br>int64|The timeout in milliseconds used to detect failures when using Kafka’s group management facilities (defaults to 10000).|
|`private_access` <br>[KafkaConnectPrivateAccessUserConfig](#kafkaconnectprivateaccessuserconfig)|Allow access to selected service ports from private networks|
|`public_access` <br>[KafkaConnectPublicAccessUserConfig](#kafkaconnectpublicaccessuserconfig)|Allow access to selected service ports from the public Internet| 
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
### KafkaPrivateAccessUserConfig 
| Field | Description|
|---|---|
|`prometheus` <br>bool|Allow clients to connect to prometheus with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations| 
### KafkaPublicAccessUserConfig 
| Field | Description|
|---|---|
|`prometheus` <br>bool|Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network|
|`schema_registry` <br>bool|Allow clients to connect to schema_registry from the public internet for service nodes that are in a project VPC or another type of private network|
|`kafka` <br>bool|Allow clients to connect to kafka from the public internet for service nodes that are in a project VPC or another type of private network|
|`kafka_connect` <br>bool|Allow clients to connect to kafka_connect from the public internet for service nodes that are in a project VPC or another type of private network|
|`kafka_rest` <br>bool|Allow clients to connect to kafka_rest from the public internet for service nodes that are in a project VPC or another type of private network| 
### KafkaRestUserConfig 
| Field | Description|
|---|---|
|`consumer_request_max_bytes` <br>int64|consumer.request.max.bytes Maximum number of bytes in unencoded message keys and values by a single request|
|`consumer_request_timeout_ms` <br>int64|consumer.request.timeout.ms The maximum total time to wait for messages for a request if the maximum number of messages has not yet been reached|
|`producer_acks` <br>string|producer.acks The number of acknowledgments the producer requires the leader to have received before considering a request complete. If set to &#39;all&#39; or &#39;-1&#39;, the leader will wait for the full set of in-sync replicas to acknowledge the record.|
|`producer_linger_ms` <br>int64|producer.linger.ms Wait for up to the given delay to allow batching records together|
|`simpleconsumer_pool_size_max` <br>int64|simpleconsumer.pool.size.max Maximum number of SimpleConsumers that can be instantiated per broker|
|`consumer_enable_auto_commit` <br>bool|consumer.enable.auto.commit If true the consumer&#39;s offset will be periodically committed to Kafka in the background|
|`public_access` <br>[KafkaPublicAccessUserConfig](#kafkapublicaccessuserconfig)|Allow access to selected service ports from the public Internet|
|`custom_domain` <br>string|Custom domain Serve the web frontend using a custom CNAME pointing to the Aiven DNS name| 
### KafkaSchema 
KafkaSchema is the Schema for the kafkaschemas API  
| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[KafkaSchemaSpec](#kafkaschemaspec)|N/A|
|`status` <br>[KafkaSchemaStatus](#kafkaschemastatus)|N/A| 
### KafkaSchemaRegistryConfig 
| Field | Description|
|---|---|
|`leader_eligibility` <br>bool|leader_eligibility If true, Karapace / Schema Registry on the service nodes can participate in leader election. It might be needed to disable this when the schemas topic is replicated to a secondary cluster and Karapace / Schema Registry there must not participate in leader election. Defaults to &#39;true&#39;.|
|`topic_name` <br>string|topic_name The durable single partition topic that acts as the durable log for the data. This topic must be compacted to avoid losing data due to retention policy. Please note that changing this configuration in an existing Schema Registry / Karapace setup leads to previous schemas being inaccessible, data encoded with them potentially unreadable and schema ID sequence put out of order. It&#39;s only possible to do the switch while Schema Registry / Karapace is disabled. Defaults to &#39;_schemas&#39;.| 
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
|`userConfig` <br>[KafkaUserConfig](#kafkauserconfig)|Kafka specific user configuration options| 
### KafkaSubKafkaUserConfig 
| Field | Description|
|---|---|
|`message_max_bytes` <br>int64|message.max.bytes The maximum size of message that the server can receive.|
|`default_replication_factor` <br>int64|default.replication.factor Replication factor for autocreated topics|
|`log_cleaner_min_cleanable_ratio` <br>int64|log.cleaner.min.cleanable.ratio Controls log compactor frequency. Larger value means more frequent compactions but also more space wasted for logs. Consider setting log.cleaner.max.compaction.lag.ms to enforce compactions sooner, instead of setting a very high value for this option.|
|`log_index_interval_bytes` <br>int64|log.index.interval.bytes The interval with which Kafka adds an entry to the offset index|
|`log_segment_delete_delay_ms` <br>int64|log.segment.delete.delay.ms The amount of time to wait before deleting a file from the filesystem|
|`max_incremental_fetch_session_cache_slots` <br>int64|max.incremental.fetch.session.cache.slots The maximum number of incremental fetch sessions that the broker will maintain.|
|`socket_request_max_bytes` <br>int64|socket.request.max.bytes The maximum number of bytes in a socket request (defaults to 104857600).|
|`log_cleaner_delete_retention_ms` <br>int64|log.cleaner.delete.retention.ms How long are delete records retained?|
|`log_index_size_max_bytes` <br>int64|log.index.size.max.bytes The maximum size in bytes of the offset index|
|`log_roll_jitter_ms` <br>int64|log.roll.jitter.ms The maximum jitter to subtract from logRollTimeMillis (in milliseconds). If not set, the value in log.roll.jitter.hours is used|
|`max_connections_per_ip` <br>int64|max.connections.per.ip The maximum number of connections allowed from each ip address (defaults to 2147483647).|
|`replica_fetch_response_max_bytes` <br>int64|replica.fetch.response.max.bytes Maximum bytes expected for the entire fetch response (defaults to 10485760). Records are fetched in batches, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made. As such, this is not an absolute maximum.|
|`auto_create_topics_enable` <br>bool|auto.create.topics.enable Enable auto creation of topics|
|`log_flush_interval_ms` <br>int64|log.flush.interval.ms The maximum time in ms that a message in any topic is kept in memory before flushed to disk. If not set, the value in log.flush.scheduler.interval.ms is used|
|`log_message_downconversion_enable` <br>bool|log.message.downconversion.enable This configuration controls whether down-conversion of message formats is enabled to satisfy consume requests.|
|`log_roll_ms` <br>int64|log.roll.ms The maximum time before a new log segment is rolled out (in milliseconds).|
|`log_cleaner_min_compaction_lag_ms` <br>int64|log.cleaner.min.compaction.lag.ms The minimum time a message will remain uncompacted in the log. Only applicable for logs that are being compacted.|
|`log_message_timestamp_difference_max_ms` <br>int64|log.message.timestamp.difference.max.ms The maximum difference allowed between the timestamp when a broker receives a message and the timestamp specified in the message|
|`log_message_timestamp_type` <br>string|log.message.timestamp.type Define whether the timestamp in the message is message create time or log append time.|
|`log_retention_ms` <br>int64|log.retention.ms The number of milliseconds to keep a log file before deleting it (in milliseconds), If not set, the value in log.retention.minutes is used. If set to -1, no time limit is applied.|
|`group_min_session_timeout_ms` <br>int64|group.min.session.timeout.ms The minimum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures.|
|`log_segment_bytes` <br>int64|log.segment.bytes The maximum size of a single log file|
|`compression_type` <br>string|compression.type Specify the final compression type for a given topic. This configuration accepts the standard compression codecs (&#39;gzip&#39;, &#39;snappy&#39;, &#39;lz4&#39;, &#39;zstd&#39;). It additionally accepts &#39;uncompressed&#39; which is equivalent to no compression; and &#39;producer&#39; which means retain the original compression codec set by the producer.|
|`group_max_session_timeout_ms` <br>int64|group.max.session.timeout.ms The maximum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures.|
|`log_flush_interval_messages` <br>int64|log.flush.interval.messages The number of messages accumulated on a log partition before messages are flushed to disk|
|`log_preallocate` <br>bool|log.preallocate Should pre allocate file when create new segment?|
|`log_retention_bytes` <br>int64|log.retention.bytes The maximum size of the log before deleting messages|
|`log_cleaner_max_compaction_lag_ms` <br>int64|log.cleaner.max.compaction.lag.ms The maximum amount of time message will remain uncompacted. Only applicable for logs that are being compacted|
|`log_retention_hours` <br>int64|log.retention.hours The number of hours to keep a log file before deleting it|
|`min_insync_replicas` <br>int64|min.insync.replicas When a producer sets acks to &#39;all&#39; (or &#39;-1&#39;), min.insync.replicas specifies the minimum number of replicas that must acknowledge a write for the write to be considered successful.|
|`num_partitions` <br>int64|num.partitions Number of partitions for autocreated topics|
|`offsets_retention_minutes` <br>int64|offsets.retention.minutes Log retention window in minutes for offsets topic|
|`connections_max_idle_ms` <br>int64|connections.max.idle.ms Idle connections timeout: the server socket processor threads close the connections that idle for longer than this.|
|`log_cleanup_policy` <br>string|log.cleanup.policy The default cleanup policy for segments beyond the retention window|
|`producer_purgatory_purge_interval_requests` <br>int64|producer.purgatory.purge.interval.requests The purge interval (in number of requests) of the producer request purgatory(defaults to 1000).|
|`replica_fetch_max_bytes` <br>int64|replica.fetch.max.bytes The number of bytes of messages to attempt to fetch for each partition (defaults to 1048576). This is not an absolute maximum, if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made.| 
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
### KafkaUserConfig 
| Field | Description|
|---|---|
|`kafka_version` <br>string|Kafka major version|
|`schema_registry` <br>bool|Enable Schema-Registry service|
|`kafka` <br>[KafkaSubKafkaUserConfig](#kafkasubkafkauserconfig)|Kafka broker configuration values|
|`kafka_connect_user_config` <br>[KafkServiceKafkaConnectUserConfig](#kafkservicekafkaconnectuserconfig)|Kafka Connect configuration values|
|`private_access` <br>[KafkaPrivateAccessUserConfig](#kafkaprivateaccessuserconfig)|Allow access to selected service ports from private networks|
|`schema_registry_config` <br>[KafkaSchemaRegistryConfig](#kafkaschemaregistryconfig)|Schema Registry configuration|
|`ip_filter` <br>[]string|IP filter Allow incoming connections from CIDR address block, e.g. &#39;10.20.0.0/16&#39;|
|`kafka_authentication_methods` <br>[KafkaAuthenticationMethodsUserConfig](#kafkaauthenticationmethodsuserconfig)|Kafka authentication methods|
|`kafka_connect` <br>bool|Enable Kafka Connect service|
|`kafka_rest` <br>bool|Enable Kafka-REST service|
|`kafka_rest_config` <br>[KafkaRestUserConfig](#kafkarestuserconfig)|Kafka REST configuration| 
### MigrationUserConfig 
| Field | Description|
|---|---|
|`host` <br>string|Hostname or IP address of the server where to migrate data from|
|`password` <br>string|Password for authentication with the server where to migrate data from|
|`port` <br>int64|Port number of the server where to migrate data from|
|`ssl` <br>bool|The server where to migrate data from is secured with SSL|
|`username` <br>string|User name for authentication with the server where to migrate data from|
|`dbname` <br>string|Database name for bootstrapping the initial connection| 
### OpenSearch 
OpenSearch is the Schema for the opensearches API  
| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[OpenSearchSpec](#opensearchspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 
### OpenSearchIndexPatterns 
| Field | Description|
|---|---|
|`max_index_count` <br>int64|Maximum number of indexes to keep|
|`pattern` <br>string|Must consist of alpha-numeric characters, dashes, underscores, dots and glob characters (* and ?)| 
### OpenSearchIndexTemplate 
| Field | Description|
|---|---|
|`number_of_replicas` <br>int64|index.number_of_replicas The number of replicas each primary shard has.|
|`number_of_shards` <br>int64|index.number_of_shards The number of primary shards that an index should have.|
|`mapping_nested_objects_limit` <br>int64|index.mapping.nested_objects.limit The maximum number of nested JSON objects that a single document can contain across all nested types. This limit helps to prevent out of memory errors when a document contains too many nested objects. Default is 10000.| 
### OpenSearchPrivateAccess 
| Field | Description|
|---|---|
|`opensearch` <br>bool|Allow clients to connect to opensearch with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations|
|`opensearch_dashboards` <br>bool|Allow clients to connect to opensearch_dashboards with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations|
|`prometheus` <br>bool|Allow clients to connect to prometheus with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations| 
### OpenSearchPrivatelinkAccess 
| Field | Description|
|---|---|
|`opensearch` <br>bool|Enable opensearch|
|`opensearch_dashboards` <br>bool|Enable opensearch_dashboards| 
### OpenSearchPublicAccess 
| Field | Description|
|---|---|
|`opensearch_dashboards` <br>bool|Allow clients to connect to opensearch_dashboards from the public internet for service nodes that are in a project VPC or another type of private network|
|`prometheus` <br>bool|Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network|
|`opensearch` <br>bool|Allow clients to connect to opensearch from the public internet for service nodes that are in a project VPC or another type of private network| 
### OpenSearchSpec 
OpenSearchSpec defines the desired state of OpenSearch  
| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>[OpenSearchUserConfig](#opensearchuserconfig)|OpenSearch specific user configuration options| 
### OpenSearchUserConfig 
| Field | Description|
|---|---|
|`opensearch_version` <br>string|OpenSearch major version|
|`opensearch_dashboards` <br>[OpensearchDashboards](#opensearchdashboards)|OpenSearch Dashboards settings|
|`recovery_basebackup_name` <br>string|Name of the basebackup to restore in forked service|
|`privatelink_access` <br>[OpenSearchPrivatelinkAccess](#opensearchprivatelinkaccess)|Allow access to selected service components through Privatelink|
|`static_ips` <br>bool|Static IP addresses Use static public IP addresses|
|`public_access` <br>[OpenSearchPublicAccess](#opensearchpublicaccess)|Allow access to selected service ports from the public Internet|
|`custom_domain` <br>string|Custom domain Serve the web frontend using a custom CNAME pointing to the Aiven DNS name|
|`index_patterns` <br>[[]OpenSearchIndexPatterns](#opensearchindexpatterns)|Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like &#39;logs.?&#39; and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note &#39;logs.?&#39; does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.|
|`ip_filter` <br>[]string|Glob pattern and number of indexes matching that pattern to be kept Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like &#39;logs.?&#39; and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note &#39;logs.?&#39; does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.IP filter Allow incoming connections from CIDR address block, e.g. &#39;10.20.0.0/16&#39; |
|`private_access` <br>[OpenSearchPrivateAccess](#opensearchprivateaccess)|Allow access to selected service ports from private networks|
|`index_template` <br>[OpenSearchIndexTemplate](#opensearchindextemplate)|Template settings for all new indexes|
|`opensearch` <br>[OpenSearchUserConfigOpenSearch](#opensearchuserconfigopensearch)|OpenSearch settings|
|`max_index_count` <br>int64|Maximum index count Maximum number of indexes to keep before deleting the oldest one|
|`project_to_fork_from` <br>string|Name of another project to fork a service from. This has effect only when a new service is being created.|
|`service_to_fork_from` <br>string|Name of another service to fork from. This has effect only when a new service is being created.|
|`disable_replication_factor_adjustment` <br>bool|Disable replication factor adjustment DEPRECATED: Disable automatic replication factor adjustment for multi-node services. By default, Aiven ensures all indexes are replicated at least to two nodes. Note: Due to potential data loss in case of losing a service node, this setting can no longer be activated.|
|`keep_index_refresh_interval` <br>bool|Don&#39;t reset index.refresh_interval to the default value Aiven automation resets index.refresh_interval to default value for every index to be sure that indices are always visible to search. If it doesn&#39;t fit your case, you can disable this by setting up this flag to true.| 
### OpenSearchUserConfigOpenSearch 
| Field | Description|
|---|---|
|`thread_pool_analyze_queue_size` <br>int64|analyze thread pool queue size for the thread pool queue. See documentation for exact details.|
|`thread_pool_analyze_size` <br>int64|analyze thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.|
|`thread_pool_force_merge_size` <br>int64|force_merge thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.|
|`thread_pool_get_size` <br>int64|get thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.|
|`thread_pool_search_throttled_queue_size` <br>int64|search_throttled thread pool queue size for the thread pool queue. See documentation for exact details.|
|`thread_pool_write_queue_size` <br>int64|write thread pool queue size for the thread pool queue. See documentation for exact details.|
|`reindex_remote_whitelist` <br>[]string|reindex_remote_whitelist Whitelisted addresses for reindexing. Changing this value will cause all OpenSearch instances to restart.Address (hostname:port or IP:port) |
|`indices_fielddata_cache_size` <br>int64|indices.fielddata.cache.size Relative amount. Maximum amount of heap memory used for field data cache. This is an expert setting; decreasing the value too much will increase overhead of loading field data; too much memory used for field data cache will decrease amount of heap available for other operations.|
|`indices_query_bool_max_clause_count` <br>int64|indices.query.bool.max_clause_count Maximum number of clauses Lucene BooleanQuery can have. The default value (1024) is relatively high, and increasing it may cause performance issues. Investigate other approaches first before increasing this value.|
|`thread_pool_get_queue_size` <br>int64|get thread pool queue size for the thread pool queue. See documentation for exact details.|
|`thread_pool_index_size` <br>int64|index thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.|
|`thread_pool_search_queue_size` <br>int64|search thread pool queue size for the thread pool queue. See documentation for exact details.|
|`thread_pool_write_size` <br>int64|write thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.|
|`http_max_header_size` <br>int64|http.max_header_size The max size of allowed headers, in bytes|
|`indices_memory_index_buffer_size` <br>int64|indices.memory.index_buffer_size Percentage value. Default is 10%. Total amount of heap used for indexing buffer, before writing segments to disk. This is an expert setting. Too low value will slow down indexing; too high value will increase indexing performance but causes performance issues for query performance.|
|`indices_queries_cache_size` <br>int64|indices.queries.cache.size Percentage value. Default is 10%. Maximum amount of heap used for query cache. This is an expert setting. Too low value will decrease query performance and increase performance for other operations; too high value will cause issues with other OpenSearch functionality.|
|`thread_pool_search_size` <br>int64|search thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.|
|`thread_pool_search_throttled_size` <br>int64|search_throttled thread pool size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.|
|`http_max_initial_line_length` <br>int64|http.max_initial_line_length The max length of an HTTP URL, in bytes|
|`action_destructive_requires_name` <br>bool|Require explicit index names when deleting|
|`cluster_max_shards_per_node` <br>int64|cluster.max_shards_per_node Controls the number of shards allowed in the cluster per data node|
|`http_max_content_length` <br>int64|http.max_content_length Maximum content length for HTTP requests to the OpenSearch HTTP API, in bytes.|
|`search_max_buckets` <br>int64|search.max_buckets Maximum number of aggregation buckets allowed in a single response. OpenSearch default value is used when this is not defined.|
|`action_auto_create_index_enabled` <br>bool|action.auto_create_index Explicitly allow or block automatic creation of indices. Defaults to true| 
### OpensearchDashboards 
| Field | Description|
|---|---|
|`enabled` <br>bool|Enable or disable OpenSearch Dashboards|
|`max_old_space_size` <br>int64|max_old_space_size Limits the maximum amount of memory (in MiB) the OpenSearch Dashboards process can use. This sets the max_old_space_size option of the nodejs running the OpenSearch Dashboards. Note: the memory reserved by OpenSearch Dashboards is not available for OpenSearch.|
|`opensearch_request_timeout` <br>int64|Timeout in milliseconds for requests made by OpenSearch Dashboards towards OpenSearch| 
### PgLookoutUserConfig 
| Field | Description|
|---|---|
|`max_failover_replication_time_lag` <br>int64|max_failover_replication_time_lag Number of seconds of master unavailability before triggering database failover to standby| 
### PgbouncerUserConfig 
| Field | Description|
|---|---|
|`ignore_startup_parameters` <br>[]string|List of parameters to ignore when given in startup packet|
|`server_reset_query_always` <br>bool|Run server_reset_query (DISCARD ALL) in all pooling modes| 
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
|`userConfig` <br>[PostgreSQLUserconfig](#postgresqluserconfig)|PostgreSQL specific user configuration options| 
### PostgreSQLSubUserConfig 
| Field | Description|
|---|---|
|`log_min_duration_statement` <br>int64|log_min_duration_statement Log statements that take more than this number of milliseconds to run, -1 disables|
|`max_replication_slots` <br>int64|max_replication_slots PostgreSQL maximum replication slots|
|`max_standby_streaming_delay` <br>int64|max_standby_streaming_delay Max standby streaming delay in milliseconds|
|`pg_partman_bgw.interval` <br>int64|pg_partman_bgw.interval Sets the time interval to run pg_partman&#39;s scheduled tasks|
|`pg_stat_statements.track` <br>string|pg_stat_statements.track Controls which statements are counted. Specify top to track top-level statements (those issued directly by clients), all to also track nested statements (such as statements invoked within functions), or none to disable statement statistics collection. The default value is top.|
|`autovacuum_vacuum_threshold` <br>int64|autovacuum_vacuum_threshold Specifies the minimum number of updated or deleted tuples needed to trigger a VACUUM in any one table. The default is 50 tuples|
|`jit` <br>bool|jit Controls system-wide use of Just-in-Time Compilation (JIT).|
|`max_prepared_transactions` <br>int64|max_prepared_transactions PostgreSQL maximum prepared transactions|
|`autovacuum_freeze_max_age` <br>int64|autovacuum_freeze_max_age Specifies the maximum age (in transactions) that a table&#39;s pg_class.relfrozenxid field can attain before a VACUUM operation is forced to prevent transaction ID wraparound within the table. Note that the system will launch autovacuum processes to prevent wraparound even when autovacuum is otherwise disabled. This parameter will cause the server to be restarted.|
|`idle_in_transaction_session_timeout` <br>int64|idle_in_transaction_session_timeout Time out sessions with open transactions after this number of milliseconds|
|`wal_sender_timeout` <br>int64|wal_sender_timeout Terminate replication connections that are inactive for longer than this amount of time, in milliseconds.|
|`max_pred_locks_per_transaction` <br>int64|max_pred_locks_per_transaction PostgreSQL maximum predicate locks per transaction|
|`timezone` <br>string|timezone PostgreSQL service timezone|
|`max_wal_senders` <br>int64|max_wal_senders PostgreSQL maximum WAL senders|
|`track_activity_query_size` <br>int64|track_activity_query_size Specifies the number of bytes reserved to track the currently executing command for each active session.|
|`max_files_per_process` <br>int64|max_files_per_process PostgreSQL maximum number of files that can be open per process|
|`max_parallel_workers_per_gather` <br>int64|max_parallel_workers_per_gather Sets the maximum number of workers that can be started by a single Gather or Gather Merge node|
|`autovacuum_vacuum_scale_factor` <br>int64|autovacuum_vacuum_scale_factor Specifies a fraction of the table size to add to autovacuum_vacuum_threshold when deciding whether to trigger a VACUUM. The default is 0.2 (20% of table size)|
|`log_autovacuum_min_duration` <br>int64|log_autovacuum_min_duration Causes each action executed by autovacuum to be logged if it ran for at least the specified number of milliseconds. Setting this to zero logs all autovacuum actions. Minus-one (the default) disables logging autovacuum actions.|
|`max_locks_per_transaction` <br>int64|max_locks_per_transaction PostgreSQL maximum locks per transaction|
|`max_stack_depth` <br>int64|max_stack_depth Maximum depth of the stack in bytes|
|`max_worker_processes` <br>int64|max_worker_processes Sets the maximum number of background processes that the system can support|
|`pg_partman_bgw.role` <br>string|pg_partman_bgw.role Controls which role to use for pg_partman&#39;s scheduled background tasks.|
|`autovacuum_analyze_scale_factor` <br>int64|autovacuum_analyze_scale_factor Specifies a fraction of the table size to add to autovacuum_analyze_threshold when deciding whether to trigger an ANALYZE. The default is 0.2 (20% of table size)|
|`autovacuum_vacuum_cost_limit` <br>int64|autovacuum_vacuum_cost_limit Specifies the cost limit value that will be used in automatic VACUUM operations. If -1 is specified (which is the default), the regular vacuum_cost_limit value will be used.|
|`temp_file_limit` <br>int64|temp_file_limit PostgreSQL temporary file limit in KiB, -1 for unlimited|
|`track_functions` <br>string|track_functions Enables tracking of function call counts and time used.|
|`max_parallel_workers` <br>int64|max_parallel_workers Sets the maximum number of workers that the system can support for parallel queries|
|`track_commit_timestamp` <br>string|track_commit_timestamp Record commit time of transactions.|
|`max_standby_archive_delay` <br>int64|max_standby_archive_delay Max standby archive delay in milliseconds|
|`wal_writer_delay` <br>int64|wal_writer_delay WAL flush interval in milliseconds. Note that setting this value to lower than the default 200ms may negatively impact performance|
|`autovacuum_analyze_threshold` <br>int64|autovacuum_analyze_threshold Specifies the minimum number of inserted, updated or deleted tuples needed to trigger an  ANALYZE in any one table. The default is 50 tuples.|
|`autovacuum_naptime` <br>int64|autovacuum_naptime Specifies the minimum delay between autovacuum runs on any given database. The delay is measured in seconds, and the default is one minute|
|`deadlock_timeout` <br>int64|deadlock_timeout This is the amount of time, in milliseconds, to wait on a lock before checking to see if there is a deadlock condition.|
|`log_error_verbosity` <br>string|log_error_verbosity Controls the amount of detail written in the server log for each message that is logged.|
|`max_logical_replication_workers` <br>int64|max_logical_replication_workers PostgreSQL maximum logical replication workers (taken from the pool of max_parallel_workers)|
|`autovacuum_max_workers` <br>int64|autovacuum_max_workers Specifies the maximum number of autovacuum processes (other than the autovacuum launcher) that may be running at any one time. The default is three. This parameter can only be set at server start.|
|`autovacuum_vacuum_cost_delay` <br>int64|autovacuum_vacuum_cost_delay Specifies the cost delay value that will be used in automatic VACUUM operations. If -1 is specified, the regular vacuum_cost_delay value will be used. The default value is 20 milliseconds| 
### PostgreSQLUserconfig 
| Field | Description|
|---|---|
|`pg_version` <br>string|PostgreSQL major version|
|`backup_minute` <br>int64|The minute of an hour when backup for the service is started. New backup is only started if previous backup has already completed.|
|`pg_service_to_fork_from` <br>string|Name of the PostgreSQL Service from which to fork (deprecated, use service_to_fork_from). This has effect only when a new service is being created.|
|`backup_hour` <br>int64|The hour of day (in UTC) when backup for the service is started. New backup is only started if previous backup has already completed.|
|`pglookout` <br>[PgLookoutUserConfig](#pglookoutuserconfig)|PGLookout settings|
|`shared_buffers_percentage` <br>int64|shared_buffers_percentage Percentage of total RAM that the database server uses for shared memory buffers. Valid range is 20-60 (float), which corresponds to 20% - 60%. This setting adjusts the shared_buffers configuration value. The absolute maximum is 12 GB.|
|`synchronous_replication` <br>string|Synchronous replication type. Note that the service plan also needs to support synchronous replication.|
|`timescaledb` <br>[TimescaledbUserConfig](#timescaledbuserconfig)|TimescaleDB extension configuration values|
|`admin_password` <br>string|Custom password for admin user. Defaults to random string. This must be set only when a new service is being created.|
|`ip_filter` <br>[]string|IP filter Allow incoming connections from CIDR address block, e.g. &#39;10.20.0.0/16&#39;|
|`pgbouncer` <br>[PgbouncerUserConfig](#pgbounceruserconfig)|PGBouncer connection pooling settings|
|`recovery_target_time` <br>string|Recovery target time when forking a service. This has effect only when a new service is being created.|
|`admin_username` <br>string|Custom username for admin user. This must be set only when a new service is being created.|
|`migration` <br>[MigrationUserConfig](#migrationuserconfig)|Migrate data from existing server|
|`private_access` <br>[PrivateAccessUserConfig](#privateaccessuserconfig)|Allow access to selected service ports from private networks|
|`public_access` <br>[PublicAccessUserConfig](#publicaccessuserconfig)|Allow access to selected service ports from the public Internet|
|`service_to_fork_from` <br>string|Name of another service to fork from. This has effect only when a new service is being created.|
|`variant` <br>string|Variant of the PostgreSQL service, may affect the features that are exposed by default|
|`work_mem` <br>int64|work_mem Sets the maximum amount of memory to be used by a query operation (such as a sort or hash table) before writing to temporary disk files, in MB. Default is 1MB &#43; 0.075% of total RAM (up to 32MB).|
|`pg` <br>[PostgreSQLSubUserConfig](#postgresqlsubuserconfig)|postgresql.conf configuration values| 
### PrivateAccessUserConfig 
| Field | Description|
|---|---|
|`pg` <br>bool|Allow clients to connect to pg with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations|
|`pgbouncer` <br>bool|Allow clients to connect to pgbouncer with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations|
|`prometheus` <br>bool|Allow clients to connect to prometheus with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations| 
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
### PublicAccessUserConfig 
| Field | Description|
|---|---|
|`pg` <br>bool|Allow clients to connect to pg from the public internet for service nodes that are in a project VPC or another type of private network|
|`pgbouncer` <br>bool|Allow clients to connect to pgbouncer from the public internet for service nodes that are in a project VPC or another type of private network|
|`prometheus` <br>bool|Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network| 
### Redis 
Redis is the Schema for the redis API  
| Field | Description|
|---|---|
|`metadata` <br>[Kubernetes meta/v1.ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#objectmeta-v1-meta)|Refer to the Kubernetes API documentation for the fields of the `metadata` field.|
|`spec` <br>[RedisSpec](#redisspec)|N/A|
|`status` <br>[ServiceStatus](#servicestatus)|N/A| 
### RedisMigration 
| Field | Description|
|---|---|
|`ignore_dbs` <br>string|Comma-separated list of databases, which should be ignored during migration (supported by MySQL only at the moment)|
|`password` <br>string|Password for authentication with the server where to migrate data from|
|`port` <br>int64|Port number of the server where to migrate data from|
|`ssl` <br>bool|The server where to migrate data from is secured with SSL|
|`username` <br>string|User name for authentication with the server where to migrate data from|
|`dbname` <br>string|Database name for bootstrapping the initial connection|
|`host` <br>string|Hostname or IP address of the server where to migrate data from| 
### RedisPrivateAccess 
| Field | Description|
|---|---|
|`prometheus` <br>bool|Allow clients to connect to prometheus with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations|
|`redis` <br>bool|Allow clients to connect to redis with a DNS name that always resolves to the service&#39;s private IP addresses. Only available in certain network locations| 
### RedisPrivatelinkAccess 
| Field | Description|
|---|---|
|`redis` <br>bool|Enable redis| 
### RedisPublicAccess 
| Field | Description|
|---|---|
|`prometheus` <br>bool|Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network|
|`redis` <br>bool|Allow clients to connect to redis from the public internet for service nodes that are in a project VPC or another type of private network| 
### RedisSpec 
RedisSpec defines the desired state of Redis  
| Field | Description|
|---|---|
|`ServiceCommonSpec` <br>[ServiceCommonSpec](#servicecommonspec)|(Members of `ServiceCommonSpec`are embedded into this type.)|
|`disk_space` <br>string|The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.|
|`authSecretRef` <br>[AuthSecretReference](#authsecretreference)|Authentication reference to Aiven token in a secret|
|`connInfoSecretTarget` <br>[ConnInfoSecretTarget](#conninfosecrettarget)|Information regarding secret creation|
|`userConfig` <br>[RedisUserConfig](#redisuserconfig)|Redis specific user configuration options| 
### RedisUserConfig 
| Field | Description|
|---|---|
|`migration` <br>[RedisMigration](#redismigration)|Migrate data from existing server|
|`public_access` <br>[RedisPublicAccess](#redispublicaccess)|Allow access to selected service ports from the public internet|
|`private_access` <br>[RedisPrivateAccess](#redisprivateaccess)|Allow access to selected service ports from private networks|
|`privatelink_access` <br>[RedisPrivatelinkAccess](#redisprivatelinkaccess)|Allow access to selected service components through Privatelink|
|`service_to_fork_from` <br>string|Name of another service to fork from. This has effect only when a new service is being created.|
|`ip_filter` <br>[]string|IP filter Allow incoming connections from CIDR address block, e.g. &#39;10.20.0.0/16&#39;|
|`project_to_fork_from` <br>string|Name of another project to fork a service from. This has effect only when a new service is being created.|
|`redis_acl_channels_default` <br>string|Default ACL for pub/sub channels used when Redis user is created Determines default pub/sub channels&#39; ACL for new users if ACL is not supplied. When this option is not defined, all_channels is assumed to keep backward compatibility. This option doesn&#39;t affect Redis configuration acl-pubsub-default.|
|`redis_lfu_decay_time` <br>int64|LFU maxmemory-policy counter decay time in minutes|
|`redis_lfu_log_factor` <br>int64|Counter logarithm factor for volatile-lfu and allkeys-lfu maxmemory-policies|
|`redis_persistence` <br>string|Redis persistence When persistence is &#39;rdb&#39;, Redis does RDB dumps each 10 minutes if any key is changed. Also RDB dumps are done according to backup schedule for backup purposes. When persistence is &#39;off&#39;, no RDB dumps and backups are done, so data can be lost at any moment if service is restarted for any reason, or if service is powered off. Also service can&#39;t be forked.|
|`redis_pubsub_client_output_buffer_limit` <br>int64|Pub/sub client output buffer hard limit in MB Set output buffer limit for pub / sub clients in MB. The value is the hard limit, the soft limit is 1/4 of the hard limit. When setting the limit, be mindful of the available memory in the selected service plan.|
|`static_ips` <br>bool|Static IP addresses Use static public IP addresses|
|`redis_io_threads` <br>int64|Redis IO thread count|
|`redis_timeout` <br>int64|Redis idle connection timeout|
|`recovery_basebackup_name` <br>string|Name of the basebackup to restore in forked service|
|`redis_maxmemory_policy` <br>string|Redis maxmemory-policy|
|`redis_notify_keyspace_events` <br>string|Set notify-keyspace-events option|
|`redis_number_of_databases` <br>int64|Number of redis databases Set number of redis databases. Changing this will cause a restart of redis service.|
|`redis_ssl` <br>bool|Require SSL to access Redis| 
### ServiceCommonSpec 
| Field | Description|
|---|---|
|`project` <br>string|Target project.|
|`plan` <br>string|Subscription plan.|
|`cloudName` <br>string|Cloud the service runs in.|
|`projectVpcId` <br>string|Identifier of the VPC the service should be in, if any.|
|`maintenanceWindowDow` <br>string|Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.|
|`maintenanceWindowTime` <br>string|Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.|
|`terminationProtection` <br>bool|Prevent service from being deleted. It is recommended to have this enabled for all services.|
|`tags` <br>map[string]string|Tags are key-value pairs that allow you to categorize services.| 
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
### TimescaledbUserConfig 
| Field | Description|
|---|---|
|`max_background_workers` <br>int64|timescaledb.max_background_workers The number of background workers for timescaledb operations. You should configure this setting to the sum of your number of databases and the total number of concurrent background workers you want running at any given point in time.| 
## References
Generated with [gen-crd-api-reference-docs](https://github.com/ahmetb/gen-crd-api-reference-docs) .
