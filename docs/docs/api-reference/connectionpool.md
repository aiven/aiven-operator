---
title: "ConnectionPool"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | ConnectionPool |

ConnectionPoolSpec defines the desired state of ConnectionPool.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`connInfoSecretTarget`](#connInfoSecretTarget){: name='connInfoSecretTarget'} (object). Information regarding secret creation. See [below for nested schema](#connInfoSecretTarget).
- [`databaseName`](#databaseName){: name='databaseName'} (string, MaxLength: 40). Name of the database the pool connects to. 
- [`poolMode`](#poolMode){: name='poolMode'} (string, Enum: `session`, `transaction`, `statement`). Mode the pool operates in (session, transaction, statement). 
- [`poolSize`](#poolSize){: name='poolSize'} (integer). Number of connections the pool may create towards the backend server. 
- [`project`](#project){: name='project'} (string, MaxLength: 63). Target project. 
- [`serviceName`](#serviceName){: name='serviceName'} (string, MaxLength: 63). Service name. 
- [`username`](#username){: name='username'} (string, MaxLength: 64). Name of the service user used to connect to the database. 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

## connInfoSecretTarget {: #connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#name){: name='name'} (string). Name of the Secret resource to be created. 

