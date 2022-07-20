---
title: "Prerequisites"
linkTitle: "Prerequisites"
weight: 1
---

The Aiven Operator for Kubernetes supports all major Kubernetes distributions, both locally and in the cloud.

Make sure you have the following:

- To use the operator, you need admin access to a Kubernetes cluster.
- For playground usage you can use [kind](https://kind.sigs.k8s.io/) for example.
- For productive usage [Helm](https://helm.sh) is recommended.

## [Cert Manager](https://cert-manager.io/)

The Aiven Operator for Kubernetes uses `cert-manager` to configure the [service reference](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#service-reference) of our webhooks.

Please follow the [installation instructions](https://cert-manager.io/docs/installation/helm/) on their website.

> Note: this is not required in the Helm installation if you select to [disable webhooks](../helm/_index.md), but that is not recommended outside of playground use. The Aiven Operator for Kubernetes uses webhooks for setting defaults and enforcing invariants that are expected by the aiven API and will lead to errors if ignored. In the future webhooks will also be used for conversion and supporting multiple CRD versions.

