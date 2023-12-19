---
title: "ServiceUser"
---

## Usage example

```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: my-service-user
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: service-user-secret
    prefix: MY_SECRET_PREFIX_
    annotations:
      foo: bar
    labels:
      baz: egg

  project: aiven-project-name
  serviceName: my-service-name
```

## ServiceUser {: #ServiceUser }

ServiceUser is the Schema for the serviceusers API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ServiceUser`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ServiceUserSpec defines the desired state of ServiceUser. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ServiceUser`](#ServiceUser)._

ServiceUserSpec defines the desired state of ServiceUser.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, MaxLength: 63, Format: `^[a-zA-Z0-9_-]*$`). Project to link the user to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, MaxLength: 63). Service to link the user to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`authentication`](#spec.authentication-property){: name='spec.authentication-property'} (string, Enum: `caching_sha2_password`, `mysql_native_password`). Authentication details.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Information regarding secret creation. Exposed keys: `SERVICEUSER_HOST`, `SERVICEUSER_PORT`, `SERVICEUSER_USERNAME`, `SERVICEUSER_PASSWORD`, `SERVICEUSER_CA_CERT`, `SERVICEUSER_ACCESS_CERT`, `SERVICEUSER_ACCESS_KEY`. See below for [nested schema](#spec.connInfoSecretTarget).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

_Appears on [`spec`](#spec)._

Information regarding secret creation. Exposed keys: `SERVICEUSER_HOST`, `SERVICEUSER_PORT`, `SERVICEUSER_USERNAME`, `SERVICEUSER_PASSWORD`, `SERVICEUSER_CA_CERT`, `SERVICEUSER_ACCESS_CERT`, `SERVICEUSER_ACCESS_KEY`.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string). Name of the secret resource to be created. By default, is equal to the resource name.

**Optional**

- [`annotations`](#spec.connInfoSecretTarget.annotations-property){: name='spec.connInfoSecretTarget.annotations-property'} (object, AdditionalProperties: string). Annotations added to the secret.
- [`labels`](#spec.connInfoSecretTarget.labels-property){: name='spec.connInfoSecretTarget.labels-property'} (object, AdditionalProperties: string). Labels added to the secret.
- [`prefix`](#spec.connInfoSecretTarget.prefix-property){: name='spec.connInfoSecretTarget.prefix-property'} (string). Prefix for the secret's keys. Added "as is" without any transformations. By default, is equal to the kind name in uppercase + underscore, e.g. `KAFKA_`, `REDIS_`, etc.

