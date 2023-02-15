---
title: "KafkaACL"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | KafkaACL |

KafkaACLSpec defines the desired state of KafkaACL.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`permission`](#permission){: name='permission'} (string, Enum: `admin`, `read`, `readwrite`, `write`). Kafka permission to grant (admin, read, readwrite, write). 
- [`project`](#project){: name='project'} (string, MaxLength: 63). Project to link the Kafka ACL to. 
- [`serviceName`](#serviceName){: name='serviceName'} (string, MaxLength: 63). Service to link the Kafka ACL to. 
- [`topic`](#topic){: name='topic'} (string). Topic name pattern for the ACL entry. 
- [`username`](#username){: name='username'} (string). Username pattern for the ACL entry. 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

