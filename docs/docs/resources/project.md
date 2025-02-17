---
title: "Project"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

??? example 
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: Project
    metadata:
      name: my-project
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      connInfoSecretTarget:
        name: project-secret
        prefix: MY_SECRET_PREFIX_
        annotations:
          foo: bar
        labels:
          baz: egg
    
      tags:
        env: prod
    
      billingAddress: NYC
      cloud: aws-eu-west-1
    ```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `Project`:

```shell
kubectl get projects my-project
```

The output is similar to the following:
```shell
Name          
my-project    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret project-secret
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret project-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"PROJECT_CA_CERT": "<secret>",
}
```

## Project {: #Project }

Project is the Schema for the projects API.

!!! Info "Exposes secret keys"

    `PROJECT_CA_CERT`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `Project`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ProjectSpec defines the desired state of Project. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`Project`](#Project)._

ProjectSpec defines the desired state of Project.

**Optional**

- [`accountId`](#spec.accountId-property){: name='spec.accountId-property'} (string, MaxLength: 32). Account ID.
- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`billingAddress`](#spec.billingAddress-property){: name='spec.billingAddress-property'} (string, MaxLength: 1000). Billing name and address of the project.
- [`billingCurrency`](#spec.billingCurrency-property){: name='spec.billingCurrency-property'} (string, Enum: `AUD`, `CAD`, `CHF`, `DKK`, `EUR`, `GBP`, `NOK`, `SEK`, `USD`). Billing currency.
- [`billingEmails`](#spec.billingEmails-property){: name='spec.billingEmails-property'} (array of strings, MaxItems: 10). Billing contact emails of the project.
- [`billingExtraText`](#spec.billingExtraText-property){: name='spec.billingExtraText-property'} (string, MaxLength: 1000). Extra text to be included in all project invoices, e.g. purchase order or cost center number.
- [`billingGroupId`](#spec.billingGroupId-property){: name='spec.billingGroupId-property'} (string, Immutable, MinLength: 36, MaxLength: 36). BillingGroup ID.
- [`cardId`](#spec.cardId-property){: name='spec.cardId-property'} (string, MaxLength: 64). Credit card ID; The ID may be either last 4 digits of the card or the actual ID.
- [`cloud`](#spec.cloud-property){: name='spec.cloud-property'} (string, MaxLength: 256). Target cloud, example: aws-eu-central-1.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Secret configuration. See below for [nested schema](#spec.connInfoSecretTarget).
- [`connInfoSecretTargetDisabled`](#spec.connInfoSecretTargetDisabled-property){: name='spec.connInfoSecretTargetDisabled-property'} (boolean, Immutable). When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
- [`copyFromProject`](#spec.copyFromProject-property){: name='spec.copyFromProject-property'} (string, Immutable, MaxLength: 63). Project name from which to copy settings to the new project.
- [`countryCode`](#spec.countryCode-property){: name='spec.countryCode-property'} (string, MinLength: 2, MaxLength: 2). Billing country code of the project.
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize projects.
- [`technicalEmails`](#spec.technicalEmails-property){: name='spec.technicalEmails-property'} (array of strings, MaxItems: 10). Technical contact emails of the project.

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

_Appears on [`spec`](#spec)._

Secret configuration.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string, Immutable). Name of the secret resource to be created. By default, it is equal to the resource name.

**Optional**

- [`annotations`](#spec.connInfoSecretTarget.annotations-property){: name='spec.connInfoSecretTarget.annotations-property'} (object, AdditionalProperties: string). Annotations added to the secret.
- [`labels`](#spec.connInfoSecretTarget.labels-property){: name='spec.connInfoSecretTarget.labels-property'} (object, AdditionalProperties: string). Labels added to the secret.
- [`prefix`](#spec.connInfoSecretTarget.prefix-property){: name='spec.connInfoSecretTarget.prefix-property'} (string). Prefix for the secret's keys.
Added "as is" without any transformations.
By default, is equal to the kind name in uppercase + underscore, e.g. `KAFKA_`, `REDIS_`, etc.
