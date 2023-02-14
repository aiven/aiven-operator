# Changelog

## 🚧 Under development

- Add `serviceIntegrations` on service types. Only `read_replica` type for now.
- Add CRD validation rules for immutable fields
- **Breaking change:** `ip_filter` field is now of `object` type.
- **Breaking change:** Update user configs for following kinds: PostgreSQL, Kafka, Redis, Clickhouse, OpenSearch, KafkaConnect.  
- Add KafkaTopic `min_cleanable_dirty_ratio` config field support
- Add Clickhouse `spec.disk_space` property
- Use updated aiven-go-client with retries

## v0.7.1 - 2023-01-24

- Add Cassandra Kind
- Add Grafana Kind
- Recreate Kafka ACL if modified. 
  Note: Modification of ACL created prior to v0.5.1 won't delete existing instance at Aiven.
  It must be deleted manually.
- Fix MySQL webhook

## v0.6.0 - 2023-01-16

- Remove "never" from choices of maintenance dow
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
