---
title: "Authentication"
linkTitle: "Authentication"
weight: 10
---

The communication between the Operator and Aiven is authenticated and authorized using a token stored in a Kubernetes Secret. We will refer to the Secret's name on every custom resource in the `authSecretRef` field.

**If you don't have an Aiven account yet, please signup [here](https://console.aiven.io/signup?utm_source=github&utm_medium=organic&utm_campaign=k8s-operator&utm_content=signup) and enjoy our free trial 🦀**

## Generating an Authentication Token
Please follow [our documentation](https://help.aiven.io/en/articles/2059201-authentication-tokens) to generate your authentication token.

## Creating the Kubernetes Secret
Let's create a Secret named `aiven-token` with a `token` field containing the authentication token:
```bash
$ kubectl create secret generic aiven-token --from-literal=token="<your-token-here>"
```

## Using the Token
All Aiven custom resources require a reference to the created Secret with the `authSecretRef` field. It looks like this:
```yaml
apiVersion: aiven.io/v1alpha1
kind: PG
metadata:
  name: pg-sample
spec:
  authSecretRef:
    name: aiven-token
    key: token
[...]
```

## Project Name
With Aiven, all resources are conceptually inside a _Project_. By default, a random Project name is generated when you signup – but you can also [create new projects](https://help.aiven.io/en/articles/5039826-how-to-create-new-project).

Please take note of the Project name, as it is also required for all the custom resources, like the example below:
```yaml
apiVersion: aiven.io/v1alpha1
kind: PG
metadata:
  name: pg-sample
spec:
  project: <your-project-name-here>
[...]
```