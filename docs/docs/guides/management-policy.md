# Management Policy

The Aiven Operator provides a management policy annotation that stops the operator from changing an existing Aiven resource. Use it when the Aiven Console or another tool owns the resource's settings and the operator should only create it, keep its connection Secret in sync and manage its deletion.

## Overview

By default the operator applies every spec change to Aiven and reverts changes made outside Kubernetes. The `controllers.aiven.io/management-policy: CreateOnly` annotation turns this off for one resource:

- A missing resource is still created from the spec. A resource that already exists at Aiven is adopted as is, without a warning.
- Spec changes, `connInfoSecretSource` rotations and out-of-band changes are left alone. Each ignored spec change or rotation is reported once with a `SpecChangeIgnored` warning event; changes made outside Kubernetes are not reported. The resource stays `Running`.
- The connection Secret is published from what Aiven reports. A `ClickhouseUser` with `connInfoSecretSource` is the exception: without the annotation its Secret mirrors the source password, with it the operator stops reading the source, because Aiven does not return ClickHouse passwords and a skipped rotation would otherwise publish a password that does not work. The Secret keeps the password from the last applied change.
- Deletion follows the [deletion policy](deletion-policy.md) as usual.

## Using the Management Policy

Add the annotation to the resource:

```yaml
apiVersion: aiven.io/v1alpha1
kind: KafkaTopic  # or any other supported resource
metadata:
  name: my-topic
  namespace: my-namespace
  annotations:
    controllers.aiven.io/management-policy: CreateOnly
spec:
  # ... existing configuration
```

A resource the Console owns usually also wants `controllers.aiven.io/deletion-policy: Orphan`, otherwise deleting the Kubernetes object deletes it at Aiven.

To apply the pending spec again, remove the annotation. The next reconcile applies every change skipped while it was set.

`CreateOnly` is the only accepted value. Any other value puts the resource into an `Error` condition with reason `CreateOrUpdate` until it is corrected.

## Supported resources

The annotation works on every resource except `ServiceUserRotation`, which ignores it, and the service kinds: `Clickhouse`, `Flink`, `Grafana`, `Kafka`, `KafkaConnect`, `MySQL`, `OpenSearch`, `PostgreSQL` and `Valkey`. On those the annotation is not supported yet and the operator reports a `ManagementPolicyNotSupported` warning event each time it applies a spec change while the annotation is set.

`ClickhouseGrant` does not support the annotation: ClickHouse does not expose which grants the operator owns, so there is nothing to observe or adopt. Setting it puts the resource into an `Error` condition with reason `CreateOrUpdate` until it is removed; deletion is unaffected.

A `kafkaSchemaRef` to an adopted `KafkaSchema` is not supported. An adopted subject becomes `Running`, but the operator never applied it, so `status.id` and `status.version` stay `0` and a dependent `KafkaSchema` referencing it waits indefinitely. Reference adopted subjects with explicit `subject` and `version` entries instead.

`UpgradePipelineStep` skips the update as described, but a step whose live settings differ from the spec is reported as not `Running`, so dependents do not act on it.
