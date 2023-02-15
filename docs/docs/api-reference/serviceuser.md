---
title: "ServiceUser"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | ServiceUser |

ServiceUserSpec defines the desired state of ServiceUser.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`authentication`](#authentication){: name='authentication'} (string, Enum: `caching_sha2_password`, `mysql_native_password`). Authentication details. 
- [`connInfoSecretTarget`](#connInfoSecretTarget){: name='connInfoSecretTarget'} (object). Information regarding secret creation. See [below for nested schema](#connInfoSecretTarget).
- [`project`](#project){: name='project'} (string, MaxLength: 63). Project to link the user to. 
- [`serviceName`](#serviceName){: name='serviceName'} (string, MaxLength: 63). Service to link the user to. 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

## connInfoSecretTarget {: #connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#name){: name='name'} (string). Name of the Secret resource to be created. 

