---
title: "OrganizationProject"
---

## Prerequisites
	
* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

### Required permissions

To create and manage this resource, you must have the appropriate [roles or permissions](https://aiven.io/docs/platform/concepts/permissions).
See the [Aiven documentation](https://aiven.io/docs/platform/howto/manage-permissions) for details on managing permissions.

This resource uses the following API operations, and for each operation, _any_ of the listed permissions is sufficient:

| Operation | Permissions  |
| ----------- | ----------- |
| [OrganizationProjectsDelete](https://api.aiven.io/doc/#operation/OrganizationProjectsDelete) | `organization:projects:write` |
| [OrganizationProjectsGet](https://api.aiven.io/doc/#operation/OrganizationProjectsGet) | `project:services:read` |
| [OrganizationProjectsUpdate](https://api.aiven.io/doc/#operation/OrganizationProjectsUpdate) | `organization:projects:write` |
| [ProjectKmsGetCA](https://api.aiven.io/doc/#operation/ProjectKmsGetCA) | `organization:projects:write` |

## Usage example

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: OrganizationProject
metadata:
  name: my-organization-project
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: my-organization-project-ca-cert

  organizationId: org123456789a
  projectId: my-organization-project
  billingGroupId: bg123456789a
  parentId: org123456789a

  basePort: 12000

  technicalEmails:
    - tech@example.com

  tags:
    env: prod
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `OrganizationProject`:

```shell
kubectl get organizationprojects my-organization-project
```

The output is similar to the following:
```shell
Name                       Organization     Project                    Parent           
my-organization-project    org123456789a    my-organization-project    org123456789a    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret my-organization-project-ca-cert
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret my-organization-project-ca-cert -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"ORGANIZATIONPROJECT_CA_CERT": "<secret>",
}
```

---

## OrganizationProject {: #OrganizationProject }

OrganizationProject is the Schema for the organizationprojects API.

!!! Warning "Adoption of existing projects"

    If `projectId` refers to a project that already exists in the organization, the operator adopts it:
    the remote state is overwritten to match the spec (billing group, parent, tags, technical emails, base port),
    and deleting the resource deletes the project in Aiven.

!!! Info "Exposes secret keys"

    `ORGANIZATIONPROJECT_CA_CERT`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `OrganizationProject`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). OrganizationProjectSpec defines the desired state of OrganizationProject. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`OrganizationProject`](#OrganizationProject)._

OrganizationProjectSpec defines the desired state of OrganizationProject.

**Required**

- [`billingGroupId`](#spec.billingGroupId-property){: name='spec.billingGroupId-property'} (string, MinLength: 1). BillingGroupID is the ID of the billing group the project is assigned to.
- [`organizationId`](#spec.organizationId-property){: name='spec.organizationId-property'} (string, Immutable, MinLength: 1). OrganizationID is the Aiven organization ID that owns the project.
    It is the addressing key for the project and cannot be changed (moving a project
    between organizations is not supported).
- [`parentId`](#spec.parentId-property){: name='spec.parentId-property'} (string, MinLength: 1). ParentID is the ID of the organization or organizational unit the project belongs to.
    Moving a project between organizational units within the same organization is supported.
- [`projectId`](#spec.projectId-property){: name='spec.projectId-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MinLength: 1, MaxLength: 63). ProjectID is the name of the project. It is immutable once set.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`basePort`](#spec.basePort-property){: name='spec.basePort-property'} (integer, Minimum: 10000, Maximum: 30000). BasePort is the valid port number range for the project, from 10000 to 30000.
    When omitted, the field is unmanaged: Aiven assigns the value and changes made
    outside Kubernetes are left as is.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Secret configuration. See below for [nested schema](#spec.connInfoSecretTarget).
- [`connInfoSecretTargetDisabled`](#spec.connInfoSecretTargetDisabled-property){: name='spec.connInfoSecretTargetDisabled-property'} (boolean, Immutable). When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize projects.
    This map is authoritative: when omitted, tags added outside
    Kubernetes are removed.
- [`technicalEmails`](#spec.technicalEmails-property){: name='spec.technicalEmails-property'} (array of strings, MaxItems: 10). TechnicalEmails are the technical contact emails of the project.
    This list is authoritative: when omitted, emails added outside
    Kubernetes are removed.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

_Appears on [`spec`](#spec)._

Secret configuration.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string, Immutable). Name of the secret resource to be created. By default, it is equal to the resource name.

**Optional**

- [`annotations`](#spec.connInfoSecretTarget.annotations-property){: name='spec.connInfoSecretTarget.annotations-property'} (object, AdditionalProperties: string). Annotations added to the secret.
- [`labels`](#spec.connInfoSecretTarget.labels-property){: name='spec.connInfoSecretTarget.labels-property'} (object, AdditionalProperties: string). Labels added to the secret.
- [`prefix`](#spec.connInfoSecretTarget.prefix-property){: name='spec.connInfoSecretTarget.prefix-property'} (string). Prefix for the secret's keys.
    Added "as is" without any transformations.
    By default, is equal to the kind name in uppercase + underscore, e.g. `KAFKA_`, `REDIS_`, etc.
