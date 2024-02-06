---
title: "Authenticating"
linkTitle: "Authenticating"
weight: 10
---

To get authenticated and authorized, set up the communication between the Aiven Operator for Kubernetes and Aiven by
using a token stored in a Kubernetes secret. You can then refer to the secret name on every custom resource in
the `authSecretRef` field.

**If you don't have an Aiven account yet, sign
up [here](https://console.aiven.io/signup?utm_source=github&utm_medium=organic&utm_campaign=k8s-operator&utm_content=signup)
for a free trial. 🦀**

1\. Generate an authentication token

Refer to [our documentation article](https://aiven.io/docs/platform/concepts/authentication-tokens) to generate your
authentication token.

2\. Create the Kubernetes Secret

The following command creates a secret named `aiven-token` with a `token` field containing the authentication token:

```shell
kubectl create secret generic aiven-token --from-literal=token="<your-token-here>"
```

When managing your Aiven resources, we will be using the created Secret in the `authSecretRef` field. It will look like
the example below:

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

Also, note that within Aiven, all resources are conceptually inside a _Project_. By default, a random project name is
generated when you signup, but you can
also [create new projects](https://aiven.io/docs/platform/howto/manage-project).

The Project name is required in most of the resources. It will look like the example below:

```yaml
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: pg-sample
spec:
  project: <your-project-name-here>
  [ ... ]
```
