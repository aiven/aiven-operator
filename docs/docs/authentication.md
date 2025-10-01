---
title: "Authentication"
linkTitle: "Authentication"
weight: 10
---

# Authentication

Set up the communication between the Aiven Operator and the Aiven Platform by using a token stored in a Kubernetes Secret. 
You can then refer to the Secret's name on every custom resource in the `authSecretRef` field.

## Prerequisites

An Aiven user account. [Sign up for free](https://console.aiven.io/signup?utm_source=github&utm_medium=organic&utm_campaign=k8s-operator&utm_content=signup).

## Store a token in a Secret

1\. Create an [application user](https://aiven.io/docs/platform/concepts/application-users) in the Aiven Console.

2\. Create an [application token](https://aiven.io/docs/platform/howto/manage-application-users#create-a-token-for-an-application-user) for the application user.

!!! note
    You can also use a [personal token](https://aiven.io/docs/platform/howto/create_authentication_token).

3\. To create a Kubernetes Secret with the token, run:

```shell
kubectl create secret generic aiven-token --from-literal=token="TOKEN"
```

Where `TOKEN` is the token. This creates a Secret named `aiven-token`.

When managing your Aiven resources, you use the Secret in the `authSecretRef` field. The following is an example
for a PostgreSQL service with the token:

```yaml
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: pg-sample
spec:
  authSecretRef:
    name: aiven-token
    key: token
  [ ... ]
```
