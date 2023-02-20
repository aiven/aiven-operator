---
title: "Kafka Connect"
linkTitle: "Kafka Connect"
weight: 50
---

[Aiven for Apache Kafka Connect](https://aiven.io/kafka-connect) is a framework and a runtime for integrating Kafka with other systems. Kafka connectors can either be a source (for pulling data from other systems into Kafka) or sink (for pushing data into other systems from Kafka).

This section involves a few different Kubernetes CRDs:
1. A `KafkaService` service with a `KafkaTopic`
2. A `KafkaConnect` service
3. A `ServiceIntegration` to integrate the `Kafka` and `KafkaConnect` services
4. A `PostgreSQL` used as a sink to receive messages from `Kafka`
5. A `KafkaConnector` to finally connect the `Kafka` with the `PostgreSQL`

## Creating the resources
Create a file named `kafka-sample-connect.yaml` with the following content:

```yaml
apiVersion: aiven.io/v1alpha1
kind: Kafka
metadata:
  name: kafka-sample-connect
spec:
  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token
  
  # outputs the Kafka connection on the `kafka-connection` Secret
  connInfoSecretTarget:
    name: kafka-auth

  # add your Project name here
  project: <your-project-name>

  # cloud provider and plan of your choice
  # you can check all of the possibilities here https://aiven.io/pricing
  cloudName: google-europe-west1
  plan: business-4

  # general Aiven configuration
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00

  # specific Kafka configuration
  userConfig:
    kafka_version: '2.7'
    kafka_connect: true

---

apiVersion: aiven.io/v1alpha1
kind: KafkaTopic
metadata:
  name: kafka-topic-connect
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: <your-project-name>
  serviceName: kafka-sample-connect

  replication: 2
  partitions: 1
```

Next, create a file named `kafka-connect.yaml` and add the following `KafkaConnect` resource:

```yaml
apiVersion: aiven.io/v1alpha1
kind: KafkaConnect
metadata:
  name: kafka-connect
spec:
  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # add your Project name here
  project: <your-project-name>

  # cloud provider and plan of your choice
  # you can check all of the possibilities here https://aiven.io/pricing
  cloudName: google-europe-west1
  plan: startup-4

  # general Aiven configuration
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
```

Now let's create a `ServiceIntegration`. It will use the fields `sourceServiceName` and `destinationServiceName` to integrate the previously created `kafka-sample-connect` and `kafka-connect`. Open a new file named `service-integration-connect.yaml` and add the content below:

```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceIntegration
metadata:
  name: service-integration-kafka-connect
spec:

  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  project: <your-project-name>

  # indicates the type of the integration
  integrationType: kafka_connect

  # we will send messages from the `kafka-sample-connect` to `kafka-connect`
  sourceServiceName: kafka-sample-connect
  destinationServiceName: kafka-connect
```

Let's add an Aiven for PostgreSQL service. It will be the service used as a _sink_, receiving messages from the `kafka-sample-connect` cluster. Create a file named `pg-sample-connect.yaml` with the content below:

```yaml
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: pg-connect
spec:

  # gets the authentication token from the `aiven-token` Secret
  authSecretRef:
    name: aiven-token
    key: token

  # outputs the PostgreSQL connection on the `pg-connection` Secret
  connInfoSecretTarget:
    name: pg-connection

  # add your Project name here
  project: <your-project-name>

  # cloud provider and plan of your choice
  # you can check all of the possibilities here https://aiven.io/pricing
  cloudName: google-europe-west1
  plan: startup-4

  # general Aiven configuration
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
```

Finally, let's add the glue of everything: a `KafkaConnector`. As described in the specification, it will send receive messages from the `kafka-sample-connect` and send them to the `pg-connect` service. Check our [official documentation](https://help.aiven.io/en/articles/1231452-kafka-connect-connectors) for more connectors.

Create a file named `kafka-connector-connect.yaml` and with the content below:

```yaml
apiVersion: aiven.io/v1alpha1
kind: KafkaConnector
metadata:
  name: kafka-connector
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: <your-project-name>

  # the Kafka cluster name
  serviceName: kafka-sample-connect

  # the connector we will be using
  connectorClass: io.aiven.connect.jdbc.JdbcSinkConnector

  userConfig:
    auto.create: "true"

    # constructs the pg-connect connection information
    connection.url: 'jdbc:postgresql://{{ fromSecret "pg-connection" "PGHOST"}}:{{ fromSecret "pg-connection" "PGPORT" }}/{{ fromSecret "pg-connection" "PGDATABASE" }}'
    connection.user: '{{ fromSecret "pg-connection" "PGUSER" }}'
    connection.password: '{{ fromSecret "pg-connection" "PGPASSWORD" }}'

    # specify which topics it will watch
    topics: kafka-topic-connect

    key.converter: org.apache.kafka.connect.json.JsonConverter
    value.converter: org.apache.kafka.connect.json.JsonConverter
    value.converter.schemas.enable: "true"
```

With all the files created, apply the new Kubernetes resources:

```shell
kubectl apply \
  -f kafka-sample-connect.yaml \
  -f kafka-connect.yaml \
  -f service-integration-connect.yaml \
  -f pg-sample-connect.yaml \
  -f kafka-connector-connect.yaml
```

It will take some time for all the services to be up and running. You can check their status with the following command:

```shell
kubectl get \
    kafkas.aiven.io/kafka-sample-connect \
    kafkaconnects.aiven.io/kafka-connect \
    postgresqls.aiven.io/pg-connect \
    kafkaconnectors.aiven.io/kafka-connector
```

The output is similar to the following:

```{ .shell .no-copy }
NAME                                  PROJECT        REGION                PLAN         STATE
kafka.aiven.io/kafka-sample-connect   your-project   google-europe-west1   business-4   RUNNING

NAME                                  STATE
kafkaconnect.aiven.io/kafka-connect   RUNNING

NAME                             PROJECT        REGION                PLAN        STATE
postgresql.aiven.io/pg-connect   your-project   google-europe-west1   startup-4   RUNNING

NAME                                      SERVICE NAME           PROJECT        CONNECTOR CLASS                           STATE     TASKS TOTAL   TASKS RUNNING
kafkaconnector.aiven.io/kafka-connector   kafka-sample-connect   your-project   io.aiven.connect.jdbc.JdbcSinkConnector   RUNNING   1             1
```
The deployment is finished when all services have the state `RUNNING`.

## Testing
To test the connection integration, let's produce a Kafka message using [kcat](https://github.com/edenhill/kcat) from within the Kubernetes cluster. We will deploy a Pod responsible for crafting a message and sending to the Kafka cluster, using the `kafka-auth` secret generate by the `Kafka` CRD.

Create a new file named `kcat-connect.yaml` and add the content below:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kafka-message
spec:
  containers:

  restartPolicy: Never
    - image: edenhill/kcat:1.7.0
      name: kcat

      command: ['/bin/sh']
      args: [
        '-c',
        'echo {\"schema\":{\"type\":\"struct\",\"fields\":[{ \"field\": \"text\", \"type\": \"string\", \"optional\": false } ] }, \"payload\": { \"text\": \"Hello World\" } } > /tmp/msg;

        kcat
          -b $(HOST):$(PORT)
          -X security.protocol=SSL
          -X ssl.key.location=/kafka-auth/ACCESS_KEY
          -X ssl.key.password=$(PASSWORD)
          -X ssl.certificate.location=/kafka-auth/ACCESS_CERT
          -X ssl.ca.location=/kafka-auth/CA_CERT
          -P -t kafka-topic-connect /tmp/msg'
      ]

      envFrom:
      - secretRef:
          name: kafka-auth

      volumeMounts:
      - name: kafka-auth
        mountPath: "/kafka-auth"

  volumes:
  - name: kafka-auth
    secret:
      secretName: kafka-auth
```

Apply the file with:

```shell
kubectl apply -f kcat-connect.yaml
```

The Pod will execute the commands and finish. You can confirm its `Completed` state with:

```shell
kubectl get pod kafka-message
```

The output is similar to the following:

```{ .shell .no-copy }
NAME            READY   STATUS      RESTARTS   AGE
kafka-message   0/1     Completed   0          5m35s
```

If everything went smoothly, we should have our produced message in the PostgreSQL service. Let's check that out.

Create a file named `psql-connect.yaml` with the content below:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: psql-connect
spec:
  restartPolicy: Never
  containers:
    - image: postgres:13
      name: postgres
      # "kafka-topic-connect" is the table automatically created by KafkaConnect
      command: ['psql', '$(DATABASE_URI)', '-c', 'SELECT * from "kafka-topic-connect";']
      
      envFrom:
      - secretRef:
          name: pg-connection
```

Apply the file with:

```shell
kubectl apply -f psql-connect.yaml
```

After a couple of seconds, inspect its log with this command: 

```shell
kubectl logs psql-connect 
```

The output is similar to the following: 

```{ .shell .no-copy }
    text     
-------------
 Hello World
(1 row)
```

## Clean up
To clean up all the created resources, use the following command:

```shell
kubectl delete \
  -f kafka-sample-connect.yaml \
  -f kafka-connect.yaml \
  -f service-integration-connect.yaml \
  -f pg-sample-connect.yaml \
  -f kafka-connector-connect.yaml \
  -f kcat-connect.yaml \
  -f psql-connect.yaml
```