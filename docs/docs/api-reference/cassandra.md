---
title: "Cassandra"
---

## Usage example

??? example 
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: Cassandra
    metadata:
      name: my-cassandra
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      connInfoSecretTarget:
        name: cassandra-secret
        prefix: MY_SECRET_PREFIX_
        annotations:
          foo: bar
        labels:
          baz: egg
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: startup-4
    
      maintenanceWindowDow: sunday
      maintenanceWindowTime: 11:00:00
    
      userConfig:
        migrate_sstableloader: true
        public_access:
          prometheus: true
        ip_filter:
          - network: 0.0.0.0
            description: whatever
          - network: 10.20.0.0/16
    ```

!!! info
	To create this resource, a `Secret` containing Aiven token must be [created](/aiven-operator/authentication.html) first.

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `Cassandra`:

```shell
kubectl get cassandras my-cassandra
```

The output is similar to the following:
```shell
Name            Project               Region                 Plan         State      
my-cassandra    aiven-project-name    google-europe-west1    startup-4    RUNNING    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret cassandra-secret
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret cassandra-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"CASSANDRA_HOST": "<secret>",
	"CASSANDRA_PORT": "<secret>",
	"CASSANDRA_USER": "<secret>",
	"CASSANDRA_PASSWORD": "<secret>",
	"CASSANDRA_URI": "<secret>",
	"CASSANDRA_HOSTS": "<secret>",
	"CASSANDRA_CA_CERT": "<secret>",
}
```

## Cassandra {: #Cassandra }

Cassandra is the Schema for the cassandras API.

!!! Info "Exposes secret keys"

    `CASSANDRA_HOST`, `CASSANDRA_PORT`, `CASSANDRA_USER`, `CASSANDRA_PASSWORD`, `CASSANDRA_URI`, `CASSANDRA_HOSTS`, `CASSANDRA_CA_CERT`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `Cassandra`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). CassandraSpec defines the desired state of Cassandra. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`Cassandra`](#Cassandra)._

CassandraSpec defines the desired state of Cassandra.

**Required**

- [`plan`](#spec.plan-property){: name='spec.plan-property'} (string, MaxLength: 128). Subscription plan.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, MaxLength: 256). Cloud the service runs in.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Secret configuration. See below for [nested schema](#spec.connInfoSecretTarget).
- [`connInfoSecretTargetDisabled`](#spec.connInfoSecretTargetDisabled-property){: name='spec.connInfoSecretTargetDisabled-property'} (boolean, Immutable). When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
- [`disk_space`](#spec.disk_space-property){: name='spec.disk_space-property'} (string, Pattern: `(?i)^[1-9][0-9]*(GiB|G)?$`). The disk space of the service, possible values depend on the service type, the cloud provider and the project.
Reducing will result in the service re-balancing.
The removal of this field does not change the value.
- [`maintenanceWindowDow`](#spec.maintenanceWindowDow-property){: name='spec.maintenanceWindowDow-property'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
- [`maintenanceWindowTime`](#spec.maintenanceWindowTime-property){: name='spec.maintenanceWindowTime-property'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
- [`projectVPCRef`](#spec.projectVPCRef-property){: name='spec.projectVPCRef-property'} (object). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See below for [nested schema](#spec.projectVPCRef).
- [`projectVpcId`](#spec.projectVpcId-property){: name='spec.projectVpcId-property'} (string, MaxLength: 36). Identifier of the VPC the service should be in, if any.
- [`serviceIntegrations`](#spec.serviceIntegrations-property){: name='spec.serviceIntegrations-property'} (array of objects, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See below for [nested schema](#spec.serviceIntegrations).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize services.
- [`technicalEmails`](#spec.technicalEmails-property){: name='spec.technicalEmails-property'} (array of objects, MaxItems: 10). Defines the email addresses that will receive alerts about upcoming maintenance updates or warnings about service instability. See below for [nested schema](#spec.technicalEmails).
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services.
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). Cassandra specific user configuration options. See below for [nested schema](#spec.userConfig).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

_Appears on [`spec`](#spec)._

Secret configuration.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string, Immutable). Name of the secret resource to be created. By default, it is equal to the resource name.

**Optional**

- [`annotations`](#spec.connInfoSecretTarget.annotations-property){: name='spec.connInfoSecretTarget.annotations-property'} (object, AdditionalProperties: string). Annotations added to the secret.
- [`labels`](#spec.connInfoSecretTarget.labels-property){: name='spec.connInfoSecretTarget.labels-property'} (object, AdditionalProperties: string). Labels added to the secret.
- [`prefix`](#spec.connInfoSecretTarget.prefix-property){: name='spec.connInfoSecretTarget.prefix-property'} (string). Prefix for the secret's keys.
Added "as is" without any transformations.
By default, is equal to the kind name in uppercase + underscore, e.g. `KAFKA_`, `REDIS_`, etc.

## projectVPCRef {: #spec.projectVPCRef }

_Appears on [`spec`](#spec)._

ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically.

**Required**

- [`name`](#spec.projectVPCRef.name-property){: name='spec.projectVPCRef.name-property'} (string, MinLength: 1).

**Optional**

- [`namespace`](#spec.projectVPCRef.namespace-property){: name='spec.projectVPCRef.namespace-property'} (string, MinLength: 1).

## serviceIntegrations {: #spec.serviceIntegrations }

_Appears on [`spec`](#spec)._

Service integrations to specify when creating a service. Not applied after initial service creation.

**Required**

- [`integrationType`](#spec.serviceIntegrations.integrationType-property){: name='spec.serviceIntegrations.integrationType-property'} (string, Enum: `read_replica`).
- [`sourceServiceName`](#spec.serviceIntegrations.sourceServiceName-property){: name='spec.serviceIntegrations.sourceServiceName-property'} (string, MinLength: 1, MaxLength: 64).

## technicalEmails {: #spec.technicalEmails }

_Appears on [`spec`](#spec)._

Defines the email addresses that will receive alerts about upcoming maintenance updates or warnings about service instability.

**Required**

- [`email`](#spec.technicalEmails.email-property){: name='spec.technicalEmails.email-property'} (string). Email address.

## userConfig {: #spec.userConfig }

_Appears on [`spec`](#spec)._

Cassandra specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Additional Cloud Regions for Backup Replication.
- [`backup_hour`](#spec.userConfig.backup_hour-property){: name='spec.userConfig.backup_hour-property'} (integer, Minimum: 0, Maximum: 23). The hour of day (in UTC) when backup for the service is started. New backup is only started if previous backup has already completed.
- [`backup_minute`](#spec.userConfig.backup_minute-property){: name='spec.userConfig.backup_minute-property'} (integer, Minimum: 0, Maximum: 59). The minute of an hour when backup for the service is started. New backup is only started if previous backup has already completed.
- [`cassandra`](#spec.userConfig.cassandra-property){: name='spec.userConfig.cassandra-property'} (object). cassandra configuration values. See below for [nested schema](#spec.userConfig.cassandra).
- [`cassandra_version`](#spec.userConfig.cassandra_version-property){: name='spec.userConfig.cassandra_version-property'} (string, Enum: `4`, `4.1`, Pattern: `^[0-9]+(\.[0-9]+)?$`). Cassandra version.
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`migrate_sstableloader`](#spec.userConfig.migrate_sstableloader-property){: name='spec.userConfig.migrate_sstableloader-property'} (boolean). Sets the service into migration mode enabling the sstableloader utility to be used to upload Cassandra data files. Available only on service create.
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`project_to_fork_from`](#spec.userConfig.project_to_fork_from-property){: name='spec.userConfig.project_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created.
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`service_to_fork_from`](#spec.userConfig.service_to_fork_from-property){: name='spec.userConfig.service_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created.
- [`service_to_join_with`](#spec.userConfig.service_to_join_with-property){: name='spec.userConfig.service_to_join_with-property'} (string, Pattern: `^[a-z][-a-z0-9]{0,63}$`, MaxLength: 64). When bootstrapping, instead of creating a new Cassandra cluster try to join an existing one from another service. Can only be set on service creation.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

### cassandra {: #spec.userConfig.cassandra }

_Appears on [`spec.userConfig`](#spec.userConfig)._

cassandra configuration values.

**Optional**

- [`batch_size_fail_threshold_in_kb`](#spec.userConfig.cassandra.batch_size_fail_threshold_in_kb-property){: name='spec.userConfig.cassandra.batch_size_fail_threshold_in_kb-property'} (integer, Minimum: 1, Maximum: 1000000). Fail any multiple-partition batch exceeding this value. 50kb (10x warn threshold) by default.
- [`batch_size_warn_threshold_in_kb`](#spec.userConfig.cassandra.batch_size_warn_threshold_in_kb-property){: name='spec.userConfig.cassandra.batch_size_warn_threshold_in_kb-property'} (integer, Minimum: 1, Maximum: 1000000). Log a warning message on any multiple-partition batch size exceeding this value.5kb per batch by default.Caution should be taken on increasing the size of this thresholdas it can lead to node instability.
- [`datacenter`](#spec.userConfig.cassandra.datacenter-property){: name='spec.userConfig.cassandra.datacenter-property'} (string, MaxLength: 128). Name of the datacenter to which nodes of this service belong. Can be set only when creating the service.
- [`read_request_timeout_in_ms`](#spec.userConfig.cassandra.read_request_timeout_in_ms-property){: name='spec.userConfig.cassandra.read_request_timeout_in_ms-property'} (integer, Minimum: 1000, Maximum: 10000). How long the coordinator waits for read operations to complete before timing it out. 5 seconds by default.
- [`write_request_timeout_in_ms`](#spec.userConfig.cassandra.write_request_timeout_in_ms-property){: name='spec.userConfig.cassandra.write_request_timeout_in_ms-property'} (integer, Minimum: 1000, Maximum: 10000). How long the coordinator waits for write requests to complete with at least one node in the local datacenter. 2 seconds by default.

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### private_access {: #spec.userConfig.private_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from private networks.

**Required**

- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Required**

- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.
