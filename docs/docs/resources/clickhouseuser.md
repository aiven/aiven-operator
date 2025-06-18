---
title: "ClickhouseUser"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: ClickhouseUser
metadata:
  name: my-clickhouse-user
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: clickhouse-user-secret
    annotations:
      foo: bar
    labels:
      baz: egg

  project: my-aiven-project
  serviceName: my-clickhouse
  username: example-username

---

apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: my-clickhouse
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: startup-16
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `ClickhouseUser`:

```shell
kubectl get clickhouseusers my-clickhouse-user
```

The output is similar to the following:
```shell
Name                  Username            Service Name     Project             
my-clickhouse-user    example-username    my-clickhouse    my-aiven-project    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret clickhouse-user-secret
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret clickhouse-user-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"CLICKHOUSEUSER_HOST": "<secret>",
	"CLICKHOUSEUSER_PORT": "<secret>",
	"CLICKHOUSEUSER_USER": "<secret>",
	"CLICKHOUSEUSER_PASSWORD": "<secret>",
}
```

---

## ClickhouseUser {: #ClickhouseUser }

ClickhouseUser is the Schema for the clickhouseusers API.

!!! Info "Exposes secret keys"

    `CLICKHOUSEUSER_HOST`, `CLICKHOUSEUSER_PORT`, `CLICKHOUSEUSER_USER`, `CLICKHOUSEUSER_PASSWORD`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ClickhouseUser`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ClickhouseUserSpec defines the desired state of ClickhouseUser. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ClickhouseUser`](#ClickhouseUser)._

ClickhouseUserSpec defines the desired state of ClickhouseUser.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Secret configuration. See below for [nested schema](#spec.connInfoSecretTarget).
- [`connInfoSecretTargetDisabled`](#spec.connInfoSecretTargetDisabled-property){: name='spec.connInfoSecretTargetDisabled-property'} (boolean, Immutable). When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
- [`username`](#spec.username-property){: name='spec.username-property'} (string, Immutable, MaxLength: 63). Name of the Clickhouse user. Defaults to `metadata.name` if omitted.

    !!! Note

        `metadata.name` is ASCII-only. For UTF-8 names, use `spec.username`, but ASCII is advised for compatibility.

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
