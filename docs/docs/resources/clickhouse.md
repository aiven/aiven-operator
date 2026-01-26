---
title: "Clickhouse"
---

## Prerequisites
	
* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

### Required permissions

To create and manage this resource, you must have the appropriate [roles or permissions](https://aiven.io/docs/platform/concepts/permissions).
See the [Aiven documentation](https://aiven.io/docs/platform/howto/manage-permissions) for details on managing permissions.

This resource uses the following API operations, and for each operation, _any_ of the listed permissions is sufficient:

| Operation | Permissions  |
| ----------- | ----------- |
| [ProjectKmsGetCA](https://api.aiven.io/doc/#operation/ProjectKmsGetCA) | `organization:projects:write` |
| [ProjectServiceTagsReplace](https://api.aiven.io/doc/#operation/ProjectServiceTagsReplace) | `service:configuration:write` |
| [ServiceBackupsGet](https://api.aiven.io/doc/#operation/ServiceBackupsGet) | `service:configuration:write` |
| [ServiceCreate](https://api.aiven.io/doc/#operation/ServiceCreate) | `project:services:write` or `role:services:recover` |
| [ServiceDelete](https://api.aiven.io/doc/#operation/ServiceDelete) | `project:services:write` |
| [ServiceGet](https://api.aiven.io/doc/#operation/ServiceGet) | `service:secrets:read` |
| [ServiceUpdate](https://api.aiven.io/doc/#operation/ServiceUpdate) | `project:services:write` or `role:services:maintenance`, or `role:services:recover`, or `service:configuration:write` |

## Usage example

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: Clickhouse
metadata:
  name: my-clickhouse
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: my-clickhouse
    annotations:
      foo: bar
    labels:
      baz: egg

  tags:
    env: test
    instance: foo

  userConfig:
    ip_filter:
      - network: 0.0.0.0/32
        description: bar
      - network: 10.20.0.0/16

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: startup-16

  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `Clickhouse`:

```shell
kubectl get clickhouses my-clickhouse
```

The output is similar to the following:
```shell
Name             Project             Region                 Plan          State      
my-clickhouse    my-aiven-project    google-europe-west1    startup-16    RUNNING    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret my-clickhouse
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret my-clickhouse -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"CLICKHOUSE_HOST": "<secret>",
	"CLICKHOUSE_PORT": "<secret>",
	"CLICKHOUSE_USER": "<secret>",
	"CLICKHOUSE_PASSWORD": "<secret>",
}
```

---

## Clickhouse {: #Clickhouse }

Clickhouse is the Schema for the clickhouses API.

!!! Info "Exposes secret keys"

    `CLICKHOUSE_HOST`, `CLICKHOUSE_PORT`, `CLICKHOUSE_USER`, `CLICKHOUSE_PASSWORD`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `Clickhouse`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ClickhouseSpec defines the desired state of Clickhouse. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`Clickhouse`](#Clickhouse)._

ClickhouseSpec defines the desired state of Clickhouse.

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
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). OpenSearch specific user configuration options. See below for [nested schema](#spec.userConfig).

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

OpenSearch specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Deprecated. Additional Cloud Regions for Backup Replication.
- [`backup_hour`](#spec.userConfig.backup_hour-property){: name='spec.userConfig.backup_hour-property'} (integer, Minimum: 0, Maximum: 23). The hour of day (in UTC) when backup for the service is started. New backup is only started if previous backup has already completed.
- [`backup_minute`](#spec.userConfig.backup_minute-property){: name='spec.userConfig.backup_minute-property'} (integer, Minimum: 0, Maximum: 59). The minute of an hour when backup for the service is started. New backup is only started if previous backup has already completed.
- [`enable_ipv6`](#spec.userConfig.enable_ipv6-property){: name='spec.userConfig.enable_ipv6-property'} (boolean). Register AAAA DNS records for the service, and allow IPv6 packets to service ports.
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 8000). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`project_to_fork_from`](#spec.userConfig.project_to_fork_from-property){: name='spec.userConfig.project_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created.
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`recovery_basebackup_name`](#spec.userConfig.recovery_basebackup_name-property){: name='spec.userConfig.recovery_basebackup_name-property'} (string, Pattern: `^[a-zA-Z0-9-_:.+]+$`, MaxLength: 128). Name of the basebackup to restore in forked service.
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`service_to_fork_from`](#spec.userConfig.service_to_fork_from-property){: name='spec.userConfig.service_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

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

**Optional**

- [`clickhouse`](#spec.userConfig.private_access.clickhouse-property){: name='spec.userConfig.private_access.clickhouse-property'} (boolean). Allow clients to connect to clickhouse with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`clickhouse_https`](#spec.userConfig.private_access.clickhouse_https-property){: name='spec.userConfig.private_access.clickhouse_https-property'} (boolean). Allow clients to connect to clickhouse_https with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`clickhouse_mysql`](#spec.userConfig.private_access.clickhouse_mysql-property){: name='spec.userConfig.private_access.clickhouse_mysql-property'} (boolean). Allow clients to connect to clickhouse_mysql with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service components through Privatelink.

**Optional**

- [`clickhouse`](#spec.userConfig.privatelink_access.clickhouse-property){: name='spec.userConfig.privatelink_access.clickhouse-property'} (boolean). Enable clickhouse.
- [`clickhouse_https`](#spec.userConfig.privatelink_access.clickhouse_https-property){: name='spec.userConfig.privatelink_access.clickhouse_https-property'} (boolean). Enable clickhouse_https.
- [`clickhouse_mysql`](#spec.userConfig.privatelink_access.clickhouse_mysql-property){: name='spec.userConfig.privatelink_access.clickhouse_mysql-property'} (boolean). Enable clickhouse_mysql.
- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Optional**

- [`clickhouse`](#spec.userConfig.public_access.clickhouse-property){: name='spec.userConfig.public_access.clickhouse-property'} (boolean). Allow clients to connect to clickhouse from the public internet for service nodes that are in a project VPC or another type of private network.
- [`clickhouse_https`](#spec.userConfig.public_access.clickhouse_https-property){: name='spec.userConfig.public_access.clickhouse_https-property'} (boolean). Allow clients to connect to clickhouse_https from the public internet for service nodes that are in a project VPC or another type of private network.
- [`clickhouse_mysql`](#spec.userConfig.public_access.clickhouse_mysql-property){: name='spec.userConfig.public_access.clickhouse_mysql-property'} (boolean). Allow clients to connect to clickhouse_mysql from the public internet for service nodes that are in a project VPC or another type of private network.
- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.
