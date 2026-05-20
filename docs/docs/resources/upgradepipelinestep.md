---
title: "UpgradePipelineStep"
---

## Prerequisites
	
* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

## Usage example

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: UpgradePipelineStep
metadata:
  name: upgrade-pipeline-step
spec:
  authSecretRef:
    name: aiven-token
    key: token

  organizationId: org123
  sourceProjectName: sandbox
  sourceServiceName: billing-pg-sandbox
  destinationProjectName: prod
  destinationServiceName: billing-pg-prod
  autoValidationDelayDays: 7
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `UpgradePipelineStep`:

```shell
kubectl get upgradepipelinesteps upgrade-pipeline-step
```

The output is similar to the following:
```shell
Name                     Organization    Source Project    Source Service        Destination Project    Destination Service    Step ID    
upgrade-pipeline-step    org123          sandbox           billing-pg-sandbox    prod                   billing-pg-prod        <id>       
```

---

## UpgradePipelineStep {: #UpgradePipelineStep }

UpgradePipelineStep is the Schema for the upgradepipelinesteps API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `UpgradePipelineStep`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). UpgradePipelineStepSpec defines the desired state of UpgradePipelineStep. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`UpgradePipelineStep`](#UpgradePipelineStep)._

UpgradePipelineStepSpec defines the desired state of UpgradePipelineStep.

**Required**

- [`destinationProjectName`](#spec.destinationProjectName-property){: name='spec.destinationProjectName-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MinLength: 1, MaxLength: 63). DestinationProjectName is the project name of the service that waits for the source service.
- [`destinationServiceName`](#spec.destinationServiceName-property){: name='spec.destinationServiceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MinLength: 1, MaxLength: 63). DestinationServiceName is the service name that waits for the source service.
- [`organizationId`](#spec.organizationId-property){: name='spec.organizationId-property'} (string, Immutable, MinLength: 1). OrganizationID is the Aiven organization ID that owns the upgrade pipeline step.
- [`sourceProjectName`](#spec.sourceProjectName-property){: name='spec.sourceProjectName-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MinLength: 1, MaxLength: 63). SourceProjectName is the project name of the service that must be upgraded first.
- [`sourceServiceName`](#spec.sourceServiceName-property){: name='spec.sourceServiceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MinLength: 1, MaxLength: 63). SourceServiceName is the service name that must be upgraded first.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`autoValidationDelayDays`](#spec.autoValidationDelayDays-property){: name='spec.autoValidationDelayDays-property'} (integer, Minimum: 0, Default value: `7`). AutoValidationDelayDays is the number of days before Aiven can automatically validate the source service.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).
