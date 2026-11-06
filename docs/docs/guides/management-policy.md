# Management Policy

By default the operator applies every spec change to Aiven and reverts changes made outside Kubernetes. The `controllers.aiven.io/management-policy: CreateOnly` annotation turns this off for one resource. Use it when the Aiven Console or another tool owns the service's settings and the operator should only create the service, keep its connection Secret in sync and manage its deletion.

## Supported resources

The annotation is supported on `Grafana` and `Valkey`. Other kinds ignore it: they apply spec changes as usual and emit a `ManagementPolicyNotSupported` warning event once per change while the annotation is set.

## What the operator corrects by default

On every poll the operator compares these fields with the live service and restores the spec value when they differ:

- `plan`, `cloudName`
- `projectVpcId` or `projectVPCRef`
- `maintenanceWindowDow`, `maintenanceWindowTime`
- `terminationProtection`, `powered`, `disk_space`
- `tags`, `technicalEmails`

A field left out of the spec is not compared, so the value set at Aiven stays. Set `tags: {}` or `technicalEmails: []` to clear them. A service deleted at Aiven is recreated from the spec.

## Using the Management Policy

Add the annotation to the resource:

```yaml
apiVersion: aiven.io/v1alpha1
kind: Grafana
metadata:
  name: my-grafana
  annotations:
    controllers.aiven.io/management-policy: CreateOnly
spec:
  # ... existing configuration
```

With the annotation set:

- A missing service is still created from the spec, including one deleted at Aiven. A service that already exists at Aiven is adopted as is, without a warning.
- Spec changes are not applied. Each one is reported once with a `SpecChangeIgnored` warning event, and the resource stays `Running`.
- Changes made outside Kubernetes are left alone. The fields that differ from the spec are listed in the `Drifted` status condition.
- The connection Secret is published from what Aiven reports.
- Deletion follows the [deletion policy](deletion-policy.md) as usual. A service the Console owns usually also wants `controllers.aiven.io/deletion-policy: Orphan`, otherwise deleting the Kubernetes object deletes the service at Aiven.

To apply the spec again, remove the annotation or set it to `Full`. The next reconcile applies every change skipped while it was set and reverts the drifted fields.

`Full` and `CreateOnly` are the only accepted values. Any other value puts the resource into an `Error` condition with reason `CreateOrUpdate` until it is corrected.
