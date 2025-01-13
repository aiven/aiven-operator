---
title: "Valkey"
---

## Valkey {: #Valkey }

Valkey is the Schema for the valkeys API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `Valkey`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ValkeySpec defines the desired state of Valkey. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`Valkey`](#Valkey)._

ValkeySpec defines the desired state of Valkey.

**Required**

- [`foo`](#spec.foo-property){: name='spec.foo-property'} (string). Foo is an example field of Valkey. Edit valkey_types.go to remove/update.
