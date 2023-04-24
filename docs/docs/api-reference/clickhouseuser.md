---
title: "ClickhouseUser"
---

## Usage example

```yaml
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: my-clickhouse-user
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  serviceName: my-clickhouse
```

## ClickhouseUser {: #ClickhouseUser }

ClickhouseUser is the Schema for the clickhouseusers API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ClickhouseUser`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ClickhouseUserSpec defines the desired state of ClickhouseUser. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ClickhouseUser`](#ClickhouseUser)._

ClickhouseUserSpec defines the desired state of ClickhouseUser.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, MaxLength: 63). Project to link the user to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, MaxLength: 63). Service to link the user to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Information regarding secret creation. See below for [nested schema](#spec.connInfoSecretTarget).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

_Appears on [`spec`](#spec)._

Information regarding secret creation.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string). Name of the secret resource to be created. By default, is equal to the resource name.

**Optional**

- [`annotations`](#spec.connInfoSecretTarget.annotations-property){: name='spec.connInfoSecretTarget.annotations-property'} (object, AdditionalProperties: string). Annotations added to the secret.
- [`labels`](#spec.connInfoSecretTarget.labels-property){: name='spec.connInfoSecretTarget.labels-property'} (object, AdditionalProperties: string). Labels added to the secret.
- [`prefix`](#spec.connInfoSecretTarget.prefix-property){: name='spec.connInfoSecretTarget.prefix-property'} (string). Prefix added to all secret's keys.

