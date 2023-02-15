---
title: "Project"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | Project |

ProjectSpec defines the desired state of Project.

- [`accountId`](#accountId){: name='accountId'} (string, MaxLength: 32). Account ID. 
- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`billingAddress`](#billingAddress){: name='billingAddress'} (string, MaxLength: 1000). Billing name and address of the project. 
- [`billingCurrency`](#billingCurrency){: name='billingCurrency'} (string, Enum: `AUD`, `CAD`, `CHF`, `DKK`, `EUR`, `GBP`, `NOK`, `SEK`, `USD`). Billing currency. 
- [`billingEmails`](#billingEmails){: name='billingEmails'} (array, MaxItems: 10). Billing contact emails of the project. 
- [`billingExtraText`](#billingExtraText){: name='billingExtraText'} (string, MaxLength: 1000). Extra text to be included in all project invoices, e.g. purchase order or cost center number. 
- [`billingGroupId`](#billingGroupId){: name='billingGroupId'} (string, MinLength: 36, MaxLength: 36). BillingGroup ID. 
- [`cardId`](#cardId){: name='cardId'} (string, MaxLength: 64). Credit card ID; The ID may be either last 4 digits of the card or the actual ID. 
- [`cloud`](#cloud){: name='cloud'} (string, MaxLength: 256). Target cloud, example: aws-eu-central-1. 
- [`connInfoSecretTarget`](#connInfoSecretTarget){: name='connInfoSecretTarget'} (object). Information regarding secret creation. See [below for nested schema](#connInfoSecretTarget).
- [`copyFromProject`](#copyFromProject){: name='copyFromProject'} (string, MaxLength: 63). Project name from which to copy settings to the new project. 
- [`countryCode`](#countryCode){: name='countryCode'} (string, MinLength: 2, MaxLength: 2). Billing country code of the project. 
- [`tags`](#tags){: name='tags'} (object). Tags are key-value pairs that allow you to categorize projects. 
- [`technicalEmails`](#technicalEmails){: name='technicalEmails'} (array, MaxItems: 10). Technical contact emails of the project. 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

## connInfoSecretTarget {: #connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#name){: name='name'} (string). Name of the Secret resource to be created. 

