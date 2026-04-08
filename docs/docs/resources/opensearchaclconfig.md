---
title: "OpenSearchACLConfig"
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
| [ServiceGet](https://api.aiven.io/doc/#operation/ServiceGet) | `project:services:read` |
| [ServiceOpenSearchAclGet](https://api.aiven.io/doc/#operation/ServiceOpenSearchAclGet) | `service:data:write` |
| [ServiceOpenSearchAclSet](https://api.aiven.io/doc/#operation/ServiceOpenSearchAclSet) | `service:data:write` |
| [ServiceOpenSearchAclUpdate](https://api.aiven.io/doc/#operation/ServiceOpenSearchAclUpdate) | `service:data:write` |

## Usage example

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: OpenSearch
metadata:
  name: my-os
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: startup-4

---

apiVersion: aiven.io/v1alpha1
kind: OpenSearchACLConfig
metadata:
  name: my-os-acl-config
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  serviceName: my-os
  enabled: true
  acls:
    - username: admin*
      rules:
        - index: ind*
          permission: deny
        - index: logs*
          permission: read
    - username: ops*
      rules:
        - index: metrics*
          permission: write
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `OpenSearchACLConfig`:

```shell
kubectl get opensearchaclconfigs my-os-acl-config
```

The output is similar to the following:
```shell
Name                Service Name    Project             Enabled    
my-os-acl-config    my-os           my-aiven-project    true       
```

---

## OpenSearchACLConfig {: #OpenSearchACLConfig }

OpenSearchACLConfig is the Schema for the opensearchaclconfigs API.
Manages the full OpenSearch ACL configuration for one Aiven OpenSearch service.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `OpenSearchACLConfig`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). OpenSearchACLConfigSpec defines the desired state of OpenSearchACLConfig. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`OpenSearchACLConfig`](#OpenSearchACLConfig)._

OpenSearchACLConfigSpec defines the desired state of OpenSearchACLConfig.

**Required**

- [`enabled`](#spec.enabled-property){: name='spec.enabled-property'} (boolean). Enable OpenSearch ACLs. When disabled, authenticated service users have unrestricted access.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.

**Optional**

- [`acls`](#spec.acls-property){: name='spec.acls-property'} (array of objects). List of OpenSearch ACLs. See below for [nested schema](#spec.acls).
- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).

## acls {: #spec.acls }

_Appears on [`spec`](#spec)._

OpenSearchACLConfigACL defines a single OpenSearch ACL entry.

**Required**

- [`rules`](#spec.acls.rules-property){: name='spec.acls.rules-property'} (array of objects). OpenSearch rules. See below for [nested schema](#spec.acls.rules).
- [`username`](#spec.acls.username-property){: name='spec.acls.username-property'} (string, MinLength: 1). Username.

### rules {: #spec.acls.rules }

_Appears on [`spec.acls`](#spec.acls)._

OpenSearchACLConfigRule defines a single OpenSearch ACL rule.

**Required**

- [`index`](#spec.acls.rules.index-property){: name='spec.acls.rules.index-property'} (string, MinLength: 1). OpenSearch index pattern.
- [`permission`](#spec.acls.rules.permission-property){: name='spec.acls.rules.permission-property'} (string, Enum: `admin`, `deny`, `read`, `readwrite`, `write`). OpenSearch permission.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
