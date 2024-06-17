---
title: "Redis"
---

## Usage example

??? example 
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: Redis
    metadata:
      name: k8s-redis
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      connInfoSecretTarget:
        name: redis-token
        prefix: MY_SECRET_PREFIX_
        annotations:
          foo: bar
        labels:
          baz: egg
    
      project: my-aiven-project
      cloudName: google-europe-west1
      plan: startup-4
    
      maintenanceWindowDow: friday
      maintenanceWindowTime: 23:00:00
    
      userConfig:
        redis_maxmemory_policy: allkeys-random
    ```

## Redis {: #Redis }

Redis is the Schema for the redis API.

!!! Info "Exposes secret keys"

    `REDIS_HOST`, `REDIS_PORT`, `REDIS_USER`, `REDIS_PASSWORD`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `Redis`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). RedisSpec defines the desired state of Redis. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`Redis`](#Redis)._

RedisSpec defines the desired state of Redis.

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
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). Redis specific user configuration options. See below for [nested schema](#spec.userConfig).

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

Redis specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Additional Cloud Regions for Backup Replication.
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`migration`](#spec.userConfig.migration-property){: name='spec.userConfig.migration-property'} (object). Migrate data from existing server. See below for [nested schema](#spec.userConfig.migration).
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`project_to_fork_from`](#spec.userConfig.project_to_fork_from-property){: name='spec.userConfig.project_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created.
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`recovery_basebackup_name`](#spec.userConfig.recovery_basebackup_name-property){: name='spec.userConfig.recovery_basebackup_name-property'} (string, Pattern: `^[a-zA-Z0-9-_:.]+$`, MaxLength: 128). Name of the basebackup to restore in forked service.
- [`redis_acl_channels_default`](#spec.userConfig.redis_acl_channels_default-property){: name='spec.userConfig.redis_acl_channels_default-property'} (string, Enum: `allchannels`, `resetchannels`). Determines default pub/sub channels' ACL for new users if ACL is not supplied. When this option is not defined, all_channels is assumed to keep backward compatibility. This option doesn't affect Redis configuration acl-pubsub-default.
- [`redis_io_threads`](#spec.userConfig.redis_io_threads-property){: name='spec.userConfig.redis_io_threads-property'} (integer, Minimum: 1, Maximum: 32). Set Redis IO thread count. Changing this will cause a restart of the Redis service.
- [`redis_lfu_decay_time`](#spec.userConfig.redis_lfu_decay_time-property){: name='spec.userConfig.redis_lfu_decay_time-property'} (integer, Minimum: 1, Maximum: 120). LFU maxmemory-policy counter decay time in minutes.
- [`redis_lfu_log_factor`](#spec.userConfig.redis_lfu_log_factor-property){: name='spec.userConfig.redis_lfu_log_factor-property'} (integer, Minimum: 0, Maximum: 100). Counter logarithm factor for volatile-lfu and allkeys-lfu maxmemory-policies.
- [`redis_maxmemory_policy`](#spec.userConfig.redis_maxmemory_policy-property){: name='spec.userConfig.redis_maxmemory_policy-property'} (string, Enum: `noeviction`, `allkeys-lru`, `volatile-lru`, `allkeys-random`, `volatile-random`, `volatile-ttl`, `volatile-lfu`, `allkeys-lfu`). Redis maxmemory-policy.
- [`redis_notify_keyspace_events`](#spec.userConfig.redis_notify_keyspace_events-property){: name='spec.userConfig.redis_notify_keyspace_events-property'} (string, Pattern: `^[KEg\$lshzxentdmA]*$`, MaxLength: 32). Set notify-keyspace-events option.
- [`redis_number_of_databases`](#spec.userConfig.redis_number_of_databases-property){: name='spec.userConfig.redis_number_of_databases-property'} (integer, Minimum: 1, Maximum: 128). Set number of Redis databases. Changing this will cause a restart of the Redis service.
- [`redis_persistence`](#spec.userConfig.redis_persistence-property){: name='spec.userConfig.redis_persistence-property'} (string, Enum: `off`, `rdb`). When persistence is `rdb`, Redis does RDB dumps each 10 minutes if any key is changed. Also RDB dumps are done according to the backup schedule for backup purposes. When persistence is `off`, no RDB dumps or backups are done, so data can be lost at any moment if the service is restarted for any reason, or if the service is powered off. Also, the service can't be forked.
- [`redis_pubsub_client_output_buffer_limit`](#spec.userConfig.redis_pubsub_client_output_buffer_limit-property){: name='spec.userConfig.redis_pubsub_client_output_buffer_limit-property'} (integer, Minimum: 32, Maximum: 512). Set output buffer limit for pub / sub clients in MB. The value is the hard limit, the soft limit is 1/4 of the hard limit. When setting the limit, be mindful of the available memory in the selected service plan.
- [`redis_ssl`](#spec.userConfig.redis_ssl-property){: name='spec.userConfig.redis_ssl-property'} (boolean). Require SSL to access Redis.
- [`redis_timeout`](#spec.userConfig.redis_timeout-property){: name='spec.userConfig.redis_timeout-property'} (integer, Minimum: 0, Maximum: 31536000). Redis idle connection timeout in seconds.
- [`redis_version`](#spec.userConfig.redis_version-property){: name='spec.userConfig.redis_version-property'} (string, Enum: `7.0`). Redis major version.
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`service_to_fork_from`](#spec.userConfig.service_to_fork_from-property){: name='spec.userConfig.service_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### migration {: #spec.userConfig.migration }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Migrate data from existing server.

**Required**

- [`host`](#spec.userConfig.migration.host-property){: name='spec.userConfig.migration.host-property'} (string, MaxLength: 255). Hostname or IP address of the server where to migrate data from.
- [`port`](#spec.userConfig.migration.port-property){: name='spec.userConfig.migration.port-property'} (integer, Minimum: 1, Maximum: 65535). Port number of the server where to migrate data from.

**Optional**

- [`dbname`](#spec.userConfig.migration.dbname-property){: name='spec.userConfig.migration.dbname-property'} (string, MaxLength: 63). Database name for bootstrapping the initial connection.
- [`ignore_dbs`](#spec.userConfig.migration.ignore_dbs-property){: name='spec.userConfig.migration.ignore_dbs-property'} (string, MaxLength: 2048). Comma-separated list of databases, which should be ignored during migration (supported by MySQL and PostgreSQL only at the moment).
- [`method`](#spec.userConfig.migration.method-property){: name='spec.userConfig.migration.method-property'} (string, Enum: `dump`, `replication`). The migration method to be used (currently supported only by Redis, Dragonfly, MySQL and PostgreSQL service types).
- [`password`](#spec.userConfig.migration.password-property){: name='spec.userConfig.migration.password-property'} (string, MaxLength: 256). Password for authentication with the server where to migrate data from.
- [`ssl`](#spec.userConfig.migration.ssl-property){: name='spec.userConfig.migration.ssl-property'} (boolean). The server where to migrate data from is secured with SSL.
- [`username`](#spec.userConfig.migration.username-property){: name='spec.userConfig.migration.username-property'} (string, MaxLength: 256). User name for authentication with the server where to migrate data from.

### private_access {: #spec.userConfig.private_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from private networks.

**Optional**

- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`redis`](#spec.userConfig.private_access.redis-property){: name='spec.userConfig.private_access.redis-property'} (boolean). Allow clients to connect to redis with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service components through Privatelink.

**Optional**

- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.
- [`redis`](#spec.userConfig.privatelink_access.redis-property){: name='spec.userConfig.privatelink_access.redis-property'} (boolean). Enable redis.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Optional**

- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.
- [`redis`](#spec.userConfig.public_access.redis-property){: name='spec.userConfig.public_access.redis-property'} (boolean). Allow clients to connect to redis from the public internet for service nodes that are in a project VPC or another type of private network.
