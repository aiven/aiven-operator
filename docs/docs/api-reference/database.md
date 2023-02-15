---
title: "Database"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | Database |

DatabaseSpec defines the desired state of Database.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`lcCollate`](#lcCollate){: name='lcCollate'} (string, MaxLength: 128). Default string sort order (LC_COLLATE) of the database. Default value: en_US.UTF-8. 
- [`lcCtype`](#lcCtype){: name='lcCtype'} (string, MaxLength: 128). Default character classification (LC_CTYPE) of the database. Default value: en_US.UTF-8. 
- [`project`](#project){: name='project'} (string, MaxLength: 63). Project to link the database to. 
- [`serviceName`](#serviceName){: name='serviceName'} (string, MaxLength: 63). PostgreSQL service to link the database to. 
- [`terminationProtection`](#terminationProtection){: name='terminationProtection'} (boolean). It is a Kubernetes side deletion protections, which prevents the database from being deleted by Kubernetes. It is recommended to enable this for any production databases containing critical data. 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

