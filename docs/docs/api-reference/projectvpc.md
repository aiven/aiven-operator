---
title: "ProjectVPC"
---

## Schema {: #Schema }

ProjectVPC is the Schema for the projectvpcs API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Must be equal to `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Must be equal to `ProjectVPC`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ProjectVPCSpec defines the desired state of ProjectVPC. See below for [nested schema](#spec).

## spec {: #spec }

ProjectVPCSpec defines the desired state of ProjectVPC.

**Required**

- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, Immutable, MaxLength: 256). Cloud the VPC is in.
- [`networkCidr`](#spec.networkCidr-property){: name='spec.networkCidr-property'} (string, Immutable, MaxLength: 36). Network address range used by the VPC like 192.168.0.0/24.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, MaxLength: 63). The project the VPC belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).

## authSecretRef {: #spec.authSecretRef }

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

