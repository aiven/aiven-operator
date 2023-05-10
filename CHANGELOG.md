# Changelog

## [MAJOR.MINOR.PATCH] - YYYY-MM-DD

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
- Remove `ip_filter` backward compatability
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
* add Redis CRD

improvements:
* watch CRDs to reconcile token secrets

fixes:
* fix RBACs of KafkaACL CRD

## v0.1.1 - 2021-09-13

improvements:
* update helm installation docs

fixes:
* fix typo in a kafka-connector kuttl test

## v0.1.0 - 2021-09-10

features:
* initial release
