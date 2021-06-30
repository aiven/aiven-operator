---
title: "Authenticating"
linkTitle: "Authenticating"
weight: 10
---

To get authenticated and authorized, set up the communication between the Aiven Kubernetes Operator and Aiven by using a token stored in a Kubernetes secret. 
You can then refer to the secret name on every custom resource in the `authSecretRef` field.

**If you don't have an Aiven account yet, sign up [here](https://console.aiven.io/signup?utm_source=github&utm_medium=organic&utm_campaign=k8s-operator&utm_content=signup) for a free trial. 🦀**

**-> To authenticate**

1. Generate an authentication token
Refer to [our documentation article](https://help.aiven.io/en/articles/2059201-authentication-tokens) to generate your authentication token.

2. Create the Kubernetes Secret
The following command creates a secret named `aiven-token` with a `token` field containing the authentication token:
```bash
$ kubectl create secret generic aiven-token --from-literal=token="<your-token-here>"
```

3. Use the token
To access and manage your Aiven custom resources, refer to the secret created using the `authSecretRef` field. It looks like this:
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

3. Create a project name
With Aiven, all resources are conceptually inside a _Project_. 
By default, a random project name is generated when you signup, but you can also [create new projects](https://help.aiven.io/en/articles/5039826-how-to-create-new-project).

Note the project name, as it is required for all the custom resources. It looks like the example below:
```yaml
apiVersion: aiven.io/v1alpha1
kind: PG
metadata:
  name: pg-sample
spec:
  project: <your-project-name-here>
[...]
```
