---
title: "ProjectVPC"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: ProjectVPC
metadata:
  name: my-project-vpc
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: aiven-project-name
  cloudName: google-europe-west1
  networkCidr: 10.0.0.0/24
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `ProjectVPC`:

```shell
kubectl get projectvpcs my-project-vpc
```

The output is similar to the following:
```shell
Name              Project               Cloud                  Network CIDR    State      
my-project-vpc    aiven-project-name    google-europe-west1    10.0.0.0/24     RUNNING    
```

---

## ProjectVPC {: #ProjectVPC }

ProjectVPC is the Schema for the projectvpcs API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ProjectVPC`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ProjectVPCSpec defines the desired state of ProjectVPC. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ProjectVPC`](#ProjectVPC)._

ProjectVPCSpec defines the desired state of ProjectVPC.

**Required**

- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, Immutable, MaxLength: 256). Cloud the VPC is in.
- [`networkCidr`](#spec.networkCidr-property){: name='spec.networkCidr-property'} (string, Immutable, MaxLength: 36). Network address range used by the VPC like 192.168.0.0/24.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
