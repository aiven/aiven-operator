---
title: "Cloning the GitHub Repository"
linkTitle: "Cloning the GitHub Repository"
weight: 20
---

Let's install the Operator from the GitHub repository.

First, clone this repository:
```bash
$ git clone git@github.com:aiven/aiven-kubernetes-operator.git
$ cd aiven-kubernetes-operator
```

Install the `cert-manager` Operator:
> cert-manager is used to manage the Operator [webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/) TLS certificates.
```bash
$ make install-cert-manager
```

Verify if the `cert-manager` was installed correctly by checking its namespace for running pods:
```bash
$ kubectl get pods --namespace cert-manager

NAME                                       READY   STATUS    RESTARTS   AGE
cert-manager-7dd5854bb4-vcg7f              1/1     Running   0          3m
cert-manager-cainjector-64c949654c-99z6k   1/1     Running   0          3m
cert-manager-webhook-6bdffc7c9d-dd4m5      1/1     Running   0          3m
```

Install the Custom Resources Definitions:
```bash
$ make install
```

Deploy the Operator:
```bash
$ make deploy
```

> Alternatively, you can execute `make run` to run the Operator directly from your local machine, without deploying it to the Kubernetes cluster. This method is recommended for local development.

Verify the deployment by checking the Operator running pod:
```bash
$ kubectl get pods --namespace aiven-kubernetes-operator-system 

NAME                                                           READY   STATUS    RESTARTS   AGE
aiven-kubernetes-operator-controller-manager-b5487dff7-2pzb8   1/1     Running   0          5m55s
```