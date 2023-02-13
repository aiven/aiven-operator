---
title: "KafkaSchema"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | KafkaSchema |

KafkaSchemaSpec defines the desired state of KafkaSchema.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`compatibilityLevel`](#compatibilityLevel){: name='compatibilityLevel'} (string, Enum: `BACKWARD`, `BACKWARD_TRANSITIVE`, `FORWARD`, `FORWARD_TRANSITIVE`, `FULL`, `FULL_TRANSITIVE`, `NONE`). Kafka Schemas compatibility level. 
- [`project`](#project){: name='project'} (string, MaxLength: 63). Project to link the Kafka Schema to. 
- [`schema`](#schema){: name='schema'} (string). Kafka Schema configuration should be a valid Avro Schema JSON format. 
- [`serviceName`](#serviceName){: name='serviceName'} (string, MaxLength: 63). Service to link the Kafka Schema to. 
- [`subjectName`](#subjectName){: name='subjectName'} (string, MaxLength: 63). Kafka Schema Subject name. 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

