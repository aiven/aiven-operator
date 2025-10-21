---
title: "Flink"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: Flink
metadata:
  name: my-flink
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: flink-secret
    annotations:
      foo: bar
    labels:
      baz: egg

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: business-4

  maintenanceWindowDow: sunday
  maintenanceWindowTime: 11:00:00

  userConfig:
    number_of_task_slots: 10
    ip_filter:
      - network: 0.0.0.0/32
        description: whatever
      - network: 10.20.0.0/16
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `Flink`:

```shell
kubectl get flinks my-flink
```

The output is similar to the following:
```shell
Name        Project             Region                 Plan          State      
my-flink    my-aiven-project    google-europe-west1    business-4    RUNNING    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret flink-secret
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret flink-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"FLINK_HOST": "<secret>",
	"FLINK_PORT": "<secret>",
	"FLINK_USER": "<secret>",
	"FLINK_PASSWORD": "<secret>",
	"FLINK_URI": "<secret>",
	"FLINK_HOSTS": "<secret>",
}
```

---

## Flink {: #Flink }

Flink is the Schema for the flinks API.

!!! Info "Exposes secret keys"

    `FLINK_HOST`, `FLINK_PORT`, `FLINK_USER`, `FLINK_PASSWORD`, `FLINK_URI`, `FLINK_HOSTS`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `Flink`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). FlinkSpec defines the desired state of Flink. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`Flink`](#Flink)._

FlinkSpec defines the desired state of Flink.

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
- [`powered`](#spec.powered-property){: name='spec.powered-property'} (boolean, Default value: `true`). Determines the power state of the service. When `true` (default), the service is running.
    When `false`, the service is powered off.
    For more information please see [Aiven documentation](https://aiven.io/docs/platform/concepts/service-power-cycle).
    Note that:
    - When set to `false` the annotation `controllers.aiven.io/instance-is-running` is also set to `false`.
    - Services cannot be created in a powered off state. The value is ignored during creation.
    - It is highly recommended to not run dependent resources when the service is powered off.
      Creating a new resource or updating an existing resource that depends on a powered off service will result in an error.
      Existing resources will need to be manually recreated after the service is powered on.
    - Existing secrets will not be updated or removed when the service is powered off.
    - For Kafka services with backups: Topic configuration, schemas and connectors are all backed up, but not the data in topics. All topic data is lost on power off.
    - For Kafka services without backups: Topic configurations including all topic data is lost on power off.
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

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Deprecated. Additional Cloud Regions for Backup Replication.
- [`custom_code`](#spec.userConfig.custom_code-property){: name='spec.userConfig.custom_code-property'} (boolean, Immutable). Enable to upload Custom JARs for Flink applications.
- [`flink_version`](#spec.userConfig.flink_version-property){: name='spec.userConfig.flink_version-property'} (string, Immutable). Available versions: `1.16`, `1.19`, `1.20`. Newer versions may also be available.
    Flink major version. Deprecated values: `1.16`.
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 8000). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`number_of_task_slots`](#spec.userConfig.number_of_task_slots-property){: name='spec.userConfig.number_of_task_slots-property'} (integer, Minimum: 1, Maximum: 1024). Task slots per node. For a 3 node plan, total number of task slots is 3x this value.
- [`pekko_ask_timeout_s`](#spec.userConfig.pekko_ask_timeout_s-property){: name='spec.userConfig.pekko_ask_timeout_s-property'} (integer, Minimum: 5, Maximum: 60). Timeout in seconds used for all futures and blocking Pekko requests.
- [`pekko_framesize_b`](#spec.userConfig.pekko_framesize_b-property){: name='spec.userConfig.pekko_framesize_b-property'} (integer, Minimum: 1048576, Maximum: 52428800). Maximum size in bytes for messages exchanged between the JobManager and the TaskManagers.
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### privatelink_access {: #spec.userConfig.privatelink_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service components through Privatelink.

**Optional**

- [`flink`](#spec.userConfig.privatelink_access.flink-property){: name='spec.userConfig.privatelink_access.flink-property'} (boolean). Enable flink.
- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Required**

- [`flink`](#spec.userConfig.public_access.flink-property){: name='spec.userConfig.public_access.flink-property'} (boolean). Allow clients to connect to flink from the public internet for service nodes that are in a project VPC or another type of private network.
