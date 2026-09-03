# Migrating from Project to OrganizationProject

[`OrganizationProject`](../resources/organizationproject.md) is the replacement for [`Project`](../resources/project.md).

This guide moves a project managed by `Project` over to `OrganizationProject` **without deleting the
Aiven project** and without touching the services running in it.

!!! warning "Read this first"
    `Project` deletes the Aiven project — and every service in it — when the Kubernetes resource is
    deleted. This procedure relies on the [`Orphan` deletion policy](deletion-policy.md) to prevent
    that. Do not skip step 2.

## Before you start

Check whether your `Project` uses `spec.cloud` or `spec.copyFromProject`. Neither exists in the
organization API, and `OrganizationProject` cannot express them — see
[What you lose](#what-you-lose) below. Everything else has an equivalent or is already deprecated
on the Aiven platform.

## What changes

`OrganizationProject` addresses a project by `spec.organizationId` + `spec.projectId` rather than
using `metadata.name` as the project name, which frees `metadata.name` to be any Kubernetes name.

| `Project`              | `OrganizationProject`                                               |
| ---------------------- | ------------------------------------------------------------------- |
| `metadata.name`        | `spec.projectId`                                                    |
| `spec.accountId`       | `spec.parentId` (organization or organizational unit)               |
| `spec.billingGroupId`  | `spec.billingGroupId` — now **required**, and no longer immutable   |
| `spec.technicalEmails` | `spec.technicalEmails`                                              |
| `spec.tags`            | `spec.tags`                                                         |
| —                      | `spec.organizationId` — **required**                                |
| —                      | `spec.basePort`                                                     |

### Billing fields

These `Project` fields have no counterpart:

`cardId`, `billingAddress`, `billingEmails`, `billingCurrency`, `billingExtraText`, `countryCode`.

Every one of them is deprecated in the Aiven API itself. Manage them on the billing group in the
[Aiven Console](https://console.aiven.io/) instead. Removing them from your manifests does not
change anything in Aiven.

### What you lose

Two fields are genuinely unavailable in `OrganizationProject`, because the organization API has no
equivalent:

* **`spec.cloud`** sets the project's default cloud for newly launched services. Migrating does not
  reset it — whatever it is today stays — but you can no longer manage it declaratively. Set
  `spec.cloudName` explicitly on each service resource instead, which is the recommended practice
  regardless.
* **`spec.copyFromProject`** only ever applied when the project was first created, so it has no
  effect on an existing project.

The `Project` status fields `vatId`, `country`, `estimatedBalance`, `paymentMethod` and
`availableCredits` are also gone. If you read them, get the equivalent from the billing group.

### Behavior to be aware of

!!! note "`tags` and `technicalEmails` are authoritative"
    `OrganizationProject` treats both as the complete desired state: omitting them **removes** tags
    and technical emails that were added outside Kubernetes. Copy the current values into your spec
    unless you intend to clear them. `spec.basePort` behaves the other way — when omitted it is left
    unmanaged.

Other resources are unaffected. `PostgreSQL`, `Kafka`, `ServiceUser`, `ProjectVPC` and the rest
reference the project by name in `spec.project`, and that name does not change.

## Migration

The example migrates a project named `my-project`.

### Step 1: Collect the new field values

`OrganizationProject` requires `organizationId`, `parentId` and `billingGroupId`. All three come
from the project itself:

```shell
curl -sH "Authorization: aivenv1 $AIVEN_TOKEN" \
  https://api.aiven.io/v1/project/my-project \
  | jq '.project | {organization_id, account_id, billing_group_id, tags, tech_emails}'
```

The output is similar to the following:

```json
{
  "organization_id": "org123456789a",
  "account_id": "a123456789a",
  "billing_group_id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  "tags": { "env": "prod" },
  "tech_emails": [{ "email": "tech@example.com" }]
}
```

Map the values as follows:

* `organization_id` → `spec.organizationId`
* `account_id` → `spec.parentId`
* `billing_group_id` → `spec.billingGroupId`

Carry `tags` and `tech_emails` into the spec as well, otherwise the operator clears them.

### Step 2: Protect the Aiven project

Annotate the existing `Project` so that deleting it leaves the Aiven project intact:

```shell
kubectl annotate project my-project controllers.aiven.io/deletion-policy=Orphan
```

Verify the annotation before continuing:

```shell
kubectl get project my-project -o jsonpath='{.metadata.annotations}'
```

If you manage the resource with GitOps, add the annotation to the manifest and let it sync **first**.
An annotation applied in the same commit that removes the resource is never reconciled.

### Step 3: Delete the Project resource

```shell
kubectl delete project my-project
```

The Aiven project and its services are preserved.

The connection secret is owned by the `Project` and is garbage collected along with it. If you want
to reuse the same secret name in the next step, wait for it to actually disappear:

```shell
kubectl wait --for=delete secret/my-project --timeout=60s
```

!!! warning "Do not run both resources at once"
    Never point a `Project` and an `OrganizationProject` at the same Aiven project. Both controllers
    reconcile tags and technical emails and will overwrite each other. Finish step 3 before step 4.

### Step 4: Apply the OrganizationProject

`OrganizationProject` adopts an existing project when `spec.projectId` matches it, so the same
project is now managed by the new resource:

```yaml
apiVersion: aiven.io/v1alpha1
kind: OrganizationProject
metadata:
  name: my-project
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: my-project
    prefix: PROJECT_

  organizationId: org123456789a
  projectId: my-project
  parentId: a123456789a
  billingGroupId: bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb

  technicalEmails:
    - tech@example.com

  tags:
    env: prod
```

```shell
kubectl apply -f organizationproject.yaml
```

### Step 5: Verify

```shell
kubectl get organizationproject my-project
```

The output is similar to the following:

```shell
NAME         ORGANIZATION     PROJECT      PARENT
my-project   org123456789a    my-project   a123456789a
```

Confirm the resource reconciled:

```shell
kubectl wait --for=condition=Running organizationproject/my-project --timeout=2m
```

## Connection secret

`Project` writes `PROJECT_CA_CERT` into its secret, plus a legacy unprefixed `CA_CERT`.
`OrganizationProject` writes `ORGANIZATIONPROJECT_CA_CERT` by default, because the key is prefixed
with the resource kind.

Setting `connInfoSecretTarget.prefix: PROJECT_`, as in step 4, keeps the key named `PROJECT_CA_CERT`
so that workloads reading it need no change. The certificate itself is unchanged — it is the same
project, so the same CA is fetched.

The legacy unprefixed `CA_CERT` key cannot be reproduced. Update any workload that reads it.

!!! note "Reusing the secret name"
    Reuse the old secret's name only after the old secret is gone (see step 3). The operator makes
    the new resource the owner of the secret, and while the deleted `Project` is still listed as its
    owner, that fails with an `already owned` error. The reconciler recovers on its own once
    Kubernetes finishes the cleanup, but waiting avoids the error entirely.
