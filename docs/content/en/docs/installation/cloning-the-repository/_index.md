---
title: "Cloning the GitHub repository"
linkTitle: "Cloning the GitHub repository"
weight: 20
---

The Aiven Kubernetes Operator can be installed from the following GitHub repository:
[aiven/aiven-kubernetes-operator.git](https://github.com/aiven/aiven-kubernetes-operator)

**-> To perfrom the installation:**

1. Clone this repository.
```bash
$ git clone git@github.com:aiven/aiven-kubernetes-operator.git
$ cd aiven-kubernetes-operator
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

6. Verify the deployment by checking the operator running Pod.

> Alternatively, you can execute `make run` to run the Operator directly from your local machine, without deploying it to the Kubernetes cluster. This method is recommended for local development.

```bash
$ kubectl get pods --namespace aiven-kubernetes-operator-system 

NAME                                                           READY   STATUS    RESTARTS   AGE
aiven-kubernetes-operator-controller-manager-b5487dff7-2pzb8   1/1     Running   0          5m55s
```

You've now installed the Aiven Kubernetes Operator. [Verify your installation](./verifying).
