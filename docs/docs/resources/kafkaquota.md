---
title: "KafkaQuota"
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
| [ServiceKafkaQuotaCreate](https://api.aiven.io/doc/#operation/ServiceKafkaQuotaCreate) | `service:data:write` |
| [ServiceKafkaQuotaDelete](https://api.aiven.io/doc/#operation/ServiceKafkaQuotaDelete) | `service:data:write` |
| [ServiceKafkaQuotaDescribe](https://api.aiven.io/doc/#operation/ServiceKafkaQuotaDescribe) | `service:data:write` |

## Usage example

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: my-kafka
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: kafka-secret

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: startup-4

  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

---

apiVersion: aiven.io/v1alpha1
kind: KafkaQuota
metadata:
  name: my-kafka-quota
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  serviceName: my-kafka

  user: my-user
  clientId: my-client
  consumerByteRate: 1000
  producerByteRate: 1000
  requestPercentage: 50
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `KafkaQuota`:

```shell
kubectl get kafkaquotas my-kafka-quota
```

The output is similar to the following:
```shell
Name              Project             Service Name    User       Client ID    
my-kafka-quota    my-aiven-project    my-kafka        my-user    my-client    
```

---

## KafkaQuota {: #KafkaQuota }

KafkaQuota is the Schema for the kafkaquotas API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `KafkaQuota`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). KafkaQuotaSpec defines the desired state of KafkaQuota. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`KafkaQuota`](#KafkaQuota)._

KafkaQuotaSpec defines the desired state of KafkaQuota.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`clientId`](#spec.clientId-property){: name='spec.clientId-property'} (string, Immutable, MaxLength: 255). Represents a logical group of clients, assigned a unique name by the client application.
    Quotas can be applied based on user, client-id, or both.
    The most relevant quota is chosen for each connection. All connections within a quota group share the same quota.
    It is possible to set default quotas for each (user, client-id), user or client-id group by specifying `default`.
- [`consumerByteRate`](#spec.consumerByteRate-property){: name='spec.consumerByteRate-property'} (integer, Minimum: 0, Maximum: 1073741824). Defines the bandwidth limit in bytes/sec for each group of clients sharing a quota.
    Every distinct client group is allocated a specific quota, as defined by the cluster, on a per-broker basis.
    Exceeding this limit results in client throttling.
- [`producerByteRate`](#spec.producerByteRate-property){: name='spec.producerByteRate-property'} (integer, Minimum: 0, Maximum: 1073741824). Defines the bandwidth limit in bytes/sec for each group of clients sharing a quota.
    Every distinct client group is allocated a specific quota, as defined by the cluster, on a per-broker basis.
    Exceeding this limit results in client throttling.
- [`requestPercentage`](#spec.requestPercentage-property){: name='spec.requestPercentage-property'} (number, Minimum: 0, Maximum: 100). Sets the maximum percentage of CPU time that a client group can use on request handler I/O and network threads per broker within a quota window.
    Exceeding this limit triggers throttling. The quota, expressed as a percentage, also indicates the total allowable CPU usage
    for the client groups sharing the quota.
- [`user`](#spec.user-property){: name='spec.user-property'} (string, Immutable, MaxLength: 255). Represents a logical group of clients, assigned a unique name by the client application.
    Quotas can be applied based on user, client-id, or both.
    The most relevant quota is chosen for each connection. All connections within a quota group share the same quota.
    It is possible to set default quotas for each (user, client-id), user or client-id group by specifying `default`.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
