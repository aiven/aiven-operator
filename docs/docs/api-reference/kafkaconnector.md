---
title: "KafkaConnector"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | KafkaConnector |

KafkaConnectorSpec defines the desired state of KafkaConnector.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`connectorClass`](#connectorClass){: name='connectorClass'} (string, MaxLength: 1024). The Java class of the connector. 
- [`project`](#project){: name='project'} (string, MaxLength: 63). Target project. 
- [`serviceName`](#serviceName){: name='serviceName'} (string, MaxLength: 63). Service name. 
- [`userConfig`](#userConfig){: name='userConfig'} (object). The connector specific configuration To build config values from secret the template function `{{ fromSecret "name" "key" }}` is provided when interpreting the keys. 

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

