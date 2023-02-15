---
title: "ProjectVPC"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | ProjectVPC |

ProjectVPCSpec defines the desired state of ProjectVPC.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`cloudName`](#cloudName){: name='cloudName'} (string, Immutable, MaxLength: 256). Cloud the VPC is in. 
- [`networkCidr`](#networkCidr){: name='networkCidr'} (string, Immutable, MaxLength: 36). Network address range used by the VPC like 192.168.0.0/24. 
- [`project`](#project){: name='project'} (string, Immutable, MaxLength: 63). The project the VPC belongs to. 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

