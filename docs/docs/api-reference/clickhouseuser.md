---
title: "ClickhouseUser"
---

## Schema {: #Schema }

ClickhouseUser is the Schema for the clickhouseusers API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Must be equal to `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Must be equal to `ClickhouseUser`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ClickhouseUserSpec defines the desired state of ClickhouseUser. See below for [nested schema](#spec).

## spec {: #spec }

ClickhouseUserSpec defines the desired state of ClickhouseUser.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, MaxLength: 63). Project to link the user to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, MaxLength: 63). Service to link the user to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`authentication`](#spec.authentication-property){: name='spec.authentication-property'} (string, Enum: `caching_sha2_password`, `mysql_native_password`). Authentication details.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Information regarding secret creation. See below for [nested schema](#spec.connInfoSecretTarget).

## authSecretRef {: #spec.authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string). Name of the secret resource to be created. By default, is equal to the resource name.

