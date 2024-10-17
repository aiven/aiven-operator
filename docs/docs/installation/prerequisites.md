---
title: "Prerequisites"
linkTitle: "Prerequisites"
weight: 1
---

# Prerequisites

The Aiven Operator for Kubernetes® supports all major Kubernetes distributions, both locally and in the cloud.

Make sure you have the following:

- Admin access to a Kubernetes cluster.
- [Cert manager installed](https://cert-manager.io/docs/installation/helm/): The operator uses this to configure the service reference of the webhooks. Webhooks are used for setting defaults
  and enforcing invariants that are expected by the Aiven API and will lead to errors if ignored.

    !!! note
        This is not required in the Helm installation if you select to [disable webhooks](./helm.md),
        but that is not recommended outside of playground use.

- For production usage, [Helm](https://helm.sh) is recommended.
- Optional: For playground usage, you can use [kind](https://kind.sigs.k8s.io/).
