---
title: "ConnectionPool"
---

## Schema {: #Schema }

ConnectionPool is the Schema for the connectionpools API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Must be equal to `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Must be equal to `ConnectionPool`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ConnectionPoolSpec defines the desired state of ConnectionPool. See below for [nested schema](#spec).

## spec {: #spec }

ConnectionPoolSpec defines the desired state of ConnectionPool.

**Required**

- [`databaseName`](#spec.databaseName-property){: name='spec.databaseName-property'} (string, MaxLength: 40). Name of the database the pool connects to.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, MaxLength: 63). Target project.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, MaxLength: 63). Service name.
- [`username`](#spec.username-property){: name='spec.username-property'} (string, MaxLength: 64). Name of the service user used to connect to the database.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Information regarding secret creation. See below for [nested schema](#spec.connInfoSecretTarget).
- [`poolMode`](#spec.poolMode-property){: name='spec.poolMode-property'} (string, Enum: `session`, `transaction`, `statement`). Mode the pool operates in (session, transaction, statement).
- [`poolSize`](#spec.poolSize-property){: name='spec.poolSize-property'} (integer). Number of connections the pool may create towards the backend server.

## authSecretRef {: #spec.authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string). Name of the secret resource to be created. By default, is equal to the resource name.

