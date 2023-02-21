---
title: "Project"
---

## Schema {: #Schema }

Project is the Schema for the projects API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Must be equal to `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Must be equal to `Project`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ProjectSpec defines the desired state of Project. See below for [nested schema](#spec).

## spec {: #spec }

ProjectSpec defines the desired state of Project.

**Optional**

- [`accountId`](#spec.accountId-property){: name='spec.accountId-property'} (string, MaxLength: 32). Account ID.
- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`billingAddress`](#spec.billingAddress-property){: name='spec.billingAddress-property'} (string, MaxLength: 1000). Billing name and address of the project.
- [`billingCurrency`](#spec.billingCurrency-property){: name='spec.billingCurrency-property'} (string, Enum: `AUD`, `CAD`, `CHF`, `DKK`, `EUR`, `GBP`, `NOK`, `SEK`, `USD`). Billing currency.
- [`billingEmails`](#spec.billingEmails-property){: name='spec.billingEmails-property'} (array of strings, MaxItems: 10). Billing contact emails of the project.
- [`billingExtraText`](#spec.billingExtraText-property){: name='spec.billingExtraText-property'} (string, MaxLength: 1000). Extra text to be included in all project invoices, e.g. purchase order or cost center number.
- [`billingGroupId`](#spec.billingGroupId-property){: name='spec.billingGroupId-property'} (string, MinLength: 36, MaxLength: 36). BillingGroup ID.
- [`cardId`](#spec.cardId-property){: name='spec.cardId-property'} (string, MaxLength: 64). Credit card ID; The ID may be either last 4 digits of the card or the actual ID.
- [`cloud`](#spec.cloud-property){: name='spec.cloud-property'} (string, MaxLength: 256). Target cloud, example: aws-eu-central-1.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Information regarding secret creation. See below for [nested schema](#spec.connInfoSecretTarget).
- [`copyFromProject`](#spec.copyFromProject-property){: name='spec.copyFromProject-property'} (string, MaxLength: 63). Project name from which to copy settings to the new project.
- [`countryCode`](#spec.countryCode-property){: name='spec.countryCode-property'} (string, MinLength: 2, MaxLength: 2). Billing country code of the project.
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize projects.
- [`technicalEmails`](#spec.technicalEmails-property){: name='spec.technicalEmails-property'} (array of strings, MaxItems: 10). Technical contact emails of the project.

## authSecretRef {: #spec.authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string). Name of the secret resource to be created. By default, is equal to the resource name.

