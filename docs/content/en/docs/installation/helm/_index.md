---
title: "Installing with Helm (recommended)"
linkTitle: "Installing with Helm (recommended)"
weight: 10
---

## Installing 

The Aiven Operator for Kubernetes can be installed via [Helm](https://helm.sh/). 

First add the [Aiven Helm repository](https://github.com/aiven/aiven-charts):

```bash
$ helm repo add aiven https://aiven.github.io/aiven-charts && helm repo update
```

### Installing Custom Resource Definitions

```bash
$ helm install aiven-operator-crds aiven/aiven-operator-crds
```

Verify the installation:
```bash
$ kubectl api-resources --api-group=aiven.io

NAME                  SHORTNAMES   APIVERSION          NAMESPACED   KIND
connectionpools                    aiven.io/v1alpha1   true         ConnectionPool
databases                          aiven.io/v1alpha1   true         Database
... < several omitted lines >
```

### Installing the Operator

```bash
$ helm install aiven-operator aiven/aiven-operator
```

> Note: Installation will fail if webhooks are enabled and the CRDs for the cert-manager are not installed.

Verify the installation: 
```bash
$  helm status aiven-operator

NAME: aiven-operator
LAST DEPLOYED: Fri Sep 10 15:23:26 2021
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

It is also possible to install the operator without webhooks enabled:
```bash
$ helm install aiven-operator aiven/aiven-operator --set webhooks.enabled=false
```

### Configuration Options

Please refer to the [values.yaml](https://github.com/aiven/aiven-charts/blob/main/charts/aiven-operator/values.yaml) of the chart.

## Uninstalling 

Find out the name of your deployment:

```bash
$ helm list

NAME               	NAMESPACE	REVISION	UPDATED                                 	STATUS  	CHART                     	APP VERSION
aiven-operator     	default  	1       	2021-09-09 10:56:14.623700249 +0200 CEST	deployed	aiven-operator-v0.1.0     	v0.1.0     
aiven-operator-crds	default  	1       	2021-09-09 10:56:05.736411868 +0200 CEST	deployed	aiven-operator-crds-v0.1.0	v0.1.0
```

Remove the CRDs:

```bash
$ helm uninstall aiven-operator-crds

release "aiven-operator-crds" uninstalled
```

Remove the operator:

```bash
$ helm uninstall aiven-operator

release "aiven-operator" uninstalled
```

