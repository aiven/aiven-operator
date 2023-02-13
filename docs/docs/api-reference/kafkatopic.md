---
title: "KafkaTopic"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | KafkaTopic |

KafkaTopicSpec defines the desired state of KafkaTopic.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`config`](#config){: name='config'} (object). Kafka topic configuration. See [below for nested schema](#config).
- [`partitions`](#partitions){: name='partitions'} (integer, Minimum: 1, Maximum: 1000000). Number of partitions to create in the topic. 
- [`project`](#project){: name='project'} (string, MaxLength: 63). Target project. 
- [`replication`](#replication){: name='replication'} (integer, Minimum: 2). Replication factor for the topic. 
- [`serviceName`](#serviceName){: name='serviceName'} (string, MaxLength: 63). Service name. 
- [`tags`](#tags){: name='tags'} (array). Kafka topic tags. See [below for nested schema](#tags).
- [`termination_protection`](#termination_protection){: name='termination_protection'} (boolean). It is a Kubernetes side deletion protections, which prevents the kafka topic from being deleted by Kubernetes. It is recommended to enable this for any production databases containing critical data. 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

## config {: #config }

Kafka topic configuration.

**Optional**

- [`cleanup_policy`](#cleanup_policy){: name='cleanup_policy'} (string). cleanup.policy value. 
- [`compression_type`](#compression_type){: name='compression_type'} (string). compression.type value. 
- [`delete_retention_ms`](#delete_retention_ms){: name='delete_retention_ms'} (integer). delete.retention.ms value. 
- [`file_delete_delay_ms`](#file_delete_delay_ms){: name='file_delete_delay_ms'} (integer). file.delete.delay.ms value. 
- [`flush_messages`](#flush_messages){: name='flush_messages'} (integer). flush.messages value. 
- [`flush_ms`](#flush_ms){: name='flush_ms'} (integer). flush.ms value. 
- [`index_interval_bytes`](#index_interval_bytes){: name='index_interval_bytes'} (integer). index.interval.bytes value. 
- [`max_compaction_lag_ms`](#max_compaction_lag_ms){: name='max_compaction_lag_ms'} (integer). max.compaction.lag.ms value. 
- [`max_message_bytes`](#max_message_bytes){: name='max_message_bytes'} (integer). max.message.bytes value. 
- [`message_downconversion_enable`](#message_downconversion_enable){: name='message_downconversion_enable'} (boolean). message.downconversion.enable value. 
- [`message_format_version`](#message_format_version){: name='message_format_version'} (string). message.format.version value. 
- [`message_timestamp_difference_max_ms`](#message_timestamp_difference_max_ms){: name='message_timestamp_difference_max_ms'} (integer). message.timestamp.difference.max.ms value. 
- [`message_timestamp_type`](#message_timestamp_type){: name='message_timestamp_type'} (string). message.timestamp.type value. 
- [`min_cleanable_dirty_ratio`](#min_cleanable_dirty_ratio){: name='min_cleanable_dirty_ratio'} (number). min.cleanable.dirty.ratio value. 
- [`min_compaction_lag_ms`](#min_compaction_lag_ms){: name='min_compaction_lag_ms'} (integer). min.compaction.lag.ms value. 
- [`min_insync_replicas`](#min_insync_replicas){: name='min_insync_replicas'} (integer). min.insync.replicas value. 
- [`preallocate`](#preallocate){: name='preallocate'} (boolean). preallocate value. 
- [`retention_bytes`](#retention_bytes){: name='retention_bytes'} (integer). retention.bytes value. 
- [`retention_ms`](#retention_ms){: name='retention_ms'} (integer). retention.ms value. 
- [`segment_bytes`](#segment_bytes){: name='segment_bytes'} (integer). segment.bytes value. 
- [`segment_index_bytes`](#segment_index_bytes){: name='segment_index_bytes'} (integer). segment.index.bytes value. 
- [`segment_jitter_ms`](#segment_jitter_ms){: name='segment_jitter_ms'} (integer). segment.jitter.ms value. 
- [`segment_ms`](#segment_ms){: name='segment_ms'} (integer). segment.ms value. 
- [`unclean_leader_election_enable`](#unclean_leader_election_enable){: name='unclean_leader_election_enable'} (boolean). unclean.leader.election.enable value. 

## tags {: #tags }

**Required**

- [`key`](#key){: name='key'} (string, MinLength: 1, MaxLength: 64).  

**Optional**

- [`value`](#value){: name='value'} (string, MaxLength: 256).  

