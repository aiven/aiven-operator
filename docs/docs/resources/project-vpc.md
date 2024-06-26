---
title: "Aiven Project VPC"
linkTitle: "Aiven Project VPC"
weight: 10
---

Virtual Private Cloud (VPC) peering is a method of connecting separate AWS, Google Cloud or Microsoft Azure private
networks to each other. It makes it possible for the virtual machines in the different VPCs to talk to each other
directly without going through the public internet.

Within the Aiven Kubernetes Operator, you can create a `ProjectVPC` on Aiven's side to connect to your cloud provider.

!!! note
    Before going through this guide, make sure you have a [Kubernetes cluster](../installation/prerequisites.md) with the operator installed (see instructions for [helm](../installation/helm.md) or [kubectl](../installation/kubectl.md)),
    and a [Kubernetes Secret with an Aiven authentication token](../authentication.md).

## Create an Aiven VPC

1\. Create a file named `vpc-sample.yaml` with the following content:

```yaml
apiVersion: aiven.io/v1alpha1
kind: ProjectVPC
metadata:
  name: vpc-sample
spec:
  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  project: PROJECT_NAME

  # creates a VPC to link an AWS account on the South Africa region
  cloudName: aws-af-south-1

  # the network range used by the VPC
  networkCidr: 192.168.0.0/24
```

2\. Create the Project by applying the configuration:

```shell
kubectl apply -f vpc-sample.yaml
```

3\. Review the resource you created with the following command:

```shell
kubectl get projects.aiven.io vpc-sample
```

The output is similar to the following:

```{ .shell .no-copy }
NAME         PROJECT          CLOUD            NETWORK CIDR
vpc-sample   PROJECT_NAME   aws-af-south-1   192.168.0.0/24
```

## Use the Aiven VPC

Follow the
official [VPC documentation](https://aiven.io/docs/platform/howto/manage-vpc-peering) to
complete the VPC peering on your cloud of choice.
