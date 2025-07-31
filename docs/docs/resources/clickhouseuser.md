---
title: "ClickhouseUser"
---

## Usage examples

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

	
=== "custom_credentials"

    ```yaml linenums="1"
    # This example demonstrates how to use ClickhouseUser with connInfoSecretSource
    # for credential management. The ClickhouseUser will use a
    # predefined password from an existing secret.
    
    apiVersion: v1
    kind: Secret
    metadata:
      name: predefined-credentials
    data:
      # MyCustomPassword123! base64 encoded
      PASSWORD: TXlDdXN0b21QYXNzd29yZDEyMyE= # gitleaks:allow
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: Clickhouse
    metadata:
      name: my-clickhouse
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: startup-16
    
      connInfoSecretTarget:
        name: clickhouse-connection
        prefix: CH_
        annotations:
          example: clickhouse-service
        labels:
          service: clickhouse
    
    ---
    
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
        prefix: MY_CLICKHOUSE_PREFIX_
        annotations:
          foo: bar
        labels:
          baz: egg
    
      # Use existing secret for credential management
      connInfoSecretSource:
        name: predefined-credentials
        # namespace: my-namespace  # Optional: defaults to same namespace as ClickhouseUser
        passwordKey: PASSWORD
    
      project: aiven-project-name
      serviceName: my-clickhouse
      username: example-username
    ```

	
=== "example"

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
my-clickhouse-user    example-username    my-clickhouse    aiven-project-name    
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
- [`connInfoSecretSource`](#spec.connInfoSecretSource-property){: name='spec.connInfoSecretSource-property'} (object). ConnInfoSecretSource allows specifying an existing secret to read credentials from.
    The password from this secret will be used to modify the ClickHouse user credentials.
    Password must be 8-256 characters long as per Aiven API requirements.
    This can be used to set passwords for new users or modify passwords for existing users.
    The source secret is watched for changes, and reconciliation will be automatically triggered
    when the secret data is updated. See below for [nested schema](#spec.connInfoSecretSource).
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

## connInfoSecretSource {: #spec.connInfoSecretSource }

_Appears on [`spec`](#spec)._

ConnInfoSecretSource allows specifying an existing secret to read credentials from.
The password from this secret will be used to modify the ClickHouse user credentials.
Password must be 8-256 characters long as per Aiven API requirements.
This can be used to set passwords for new users or modify passwords for existing users.
The source secret is watched for changes, and reconciliation will be automatically triggered
when the secret data is updated.

**Required**

- [`name`](#spec.connInfoSecretSource.name-property){: name='spec.connInfoSecretSource.name-property'} (string, MinLength: 1). Name of the secret resource to read connection parameters from.
- [`passwordKey`](#spec.connInfoSecretSource.passwordKey-property){: name='spec.connInfoSecretSource.passwordKey-property'} (string, MinLength: 1). Key in the secret containing the password to use for authentication.

**Optional**

- [`namespace`](#spec.connInfoSecretSource.namespace-property){: name='spec.connInfoSecretSource.namespace-property'} (string). Namespace of the source secret. If not specified, defaults to the same namespace as the resource.

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
