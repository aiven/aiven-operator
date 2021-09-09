---
title: "Installing from the source code"
linkTitle: "Installing from the source code"
weight: 20
---

The Aiven Operator for Kubernetes can be installed from the following GitHub repository:
[aiven/aiven-operator](https://github.com/aiven/aiven-operator).

1. Clone this repository.

```bash
$ git clone git@github.com:aiven/aiven-operator.git
$ cd aiven-operator
```

2. Install the `cert-manager` operator.

> cert-manager is used to manage the Operator [webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) TLS certificates.

```bash
$ make install-cert-manager
```

3. Verify that the `cert-manager` is installed correctly by checking its namespace for running Pods.

```bash
$ kubectl get pods --namespace cert-manager

NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-7dd5854bb4-vcg7f              1/1     Running   0          3m
cert-manager-cainjector-64c949654c-99z6k   1/1     Running   0          3m
cert-manager-webhook-6bdffc7c9d-dd4m5      1/1     Running   0          3m
```

4. Install the Custom Resources Definitions.

```bash
$ make install
```

5. Deploy the operator.

```bash
$ make deploy
```

You've now installed the Aiven Operator for Kubernetes. [Verify your installation](./verifying).
