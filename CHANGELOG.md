# Changelog

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
