---
title: "PostgreSQL"
---

## Usage example

??? example 
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: PostgreSQL
    metadata:
      name: my-postgresql
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      connInfoSecretTarget:
        name: postgresql-secret
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
        pg_version: "15"
    ```

!!! info
	To create this resource, a `Secret` containing Aiven token must be [created](/aiven-operator/authentication.html) first.

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `PostgreSQL`:

```shell
kubectl get postgresqls my-postgresql
```

The output is similar to the following:
```shell
Name             Project               Region                 Plan         State      
my-postgresql    aiven-project-name    google-europe-west1    startup-4    RUNNING    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret postgresql-secret
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret postgresql-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"POSTGRESQL_HOST": "<secret>",
	"POSTGRESQL_PORT": "<secret>",
	"POSTGRESQL_DATABASE": "<secret>",
	"POSTGRESQL_USER": "<secret>",
	"POSTGRESQL_PASSWORD": "<secret>",
	"POSTGRESQL_SSLMODE": "<secret>",
	"POSTGRESQL_DATABASE_URI": "<secret>",
	"POSTGRESQL_CA_CERT": "<secret>",
}
```

## PostgreSQL {: #PostgreSQL }

PostgreSQL is the Schema for the postgresql API.

!!! Info "Exposes secret keys"

    `POSTGRESQL_HOST`, `POSTGRESQL_PORT`, `POSTGRESQL_DATABASE`, `POSTGRESQL_USER`, `POSTGRESQL_PASSWORD`, `POSTGRESQL_SSLMODE`, `POSTGRESQL_DATABASE_URI`, `POSTGRESQL_CA_CERT`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `PostgreSQL`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). PostgreSQLSpec defines the desired state of postgres instance. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`PostgreSQL`](#PostgreSQL)._

PostgreSQLSpec defines the desired state of postgres instance.

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
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). PostgreSQL specific user configuration options. See below for [nested schema](#spec.userConfig).

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

PostgreSQL specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Additional Cloud Regions for Backup Replication.
- [`admin_password`](#spec.userConfig.admin_password-property){: name='spec.userConfig.admin_password-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9-_]+$`, MinLength: 8, MaxLength: 256). Custom password for admin user. Defaults to random string. This must be set only when a new service is being created.
- [`admin_username`](#spec.userConfig.admin_username-property){: name='spec.userConfig.admin_username-property'} (string, Immutable, Pattern: `^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$`, MaxLength: 64). Custom username for admin user. This must be set only when a new service is being created.
- [`backup_hour`](#spec.userConfig.backup_hour-property){: name='spec.userConfig.backup_hour-property'} (integer, Minimum: 0, Maximum: 23). The hour of day (in UTC) when backup for the service is started. New backup is only started if previous backup has already completed.
- [`backup_minute`](#spec.userConfig.backup_minute-property){: name='spec.userConfig.backup_minute-property'} (integer, Minimum: 0, Maximum: 59). The minute of an hour when backup for the service is started. New backup is only started if previous backup has already completed.
- [`enable_ipv6`](#spec.userConfig.enable_ipv6-property){: name='spec.userConfig.enable_ipv6-property'} (boolean). Register AAAA DNS records for the service, and allow IPv6 packets to service ports.
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`migration`](#spec.userConfig.migration-property){: name='spec.userConfig.migration-property'} (object). Migrate data from existing server. See below for [nested schema](#spec.userConfig.migration).
- [`pg`](#spec.userConfig.pg-property){: name='spec.userConfig.pg-property'} (object). postgresql.conf configuration values. See below for [nested schema](#spec.userConfig.pg).
- [`pg_qualstats`](#spec.userConfig.pg_qualstats-property){: name='spec.userConfig.pg_qualstats-property'} (object). Deprecated. System-wide settings for the pg_qualstats extension. See below for [nested schema](#spec.userConfig.pg_qualstats).
- [`pg_read_replica`](#spec.userConfig.pg_read_replica-property){: name='spec.userConfig.pg_read_replica-property'} (boolean). Should the service which is being forked be a read replica (deprecated, use read_replica service integration instead).
- [`pg_service_to_fork_from`](#spec.userConfig.pg_service_to_fork_from-property){: name='spec.userConfig.pg_service_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 64). Name of the PG Service from which to fork (deprecated, use service_to_fork_from). This has effect only when a new service is being created.
- [`pg_stat_monitor_enable`](#spec.userConfig.pg_stat_monitor_enable-property){: name='spec.userConfig.pg_stat_monitor_enable-property'} (boolean). Enable the pg_stat_monitor extension. Enabling this extension will cause the cluster to be restarted.When this extension is enabled, pg_stat_statements results for utility commands are unreliable.
- [`pg_version`](#spec.userConfig.pg_version-property){: name='spec.userConfig.pg_version-property'} (string, Enum: `13`, `14`, `15`, `16`, `17`). PostgreSQL major version.
- [`pgaudit`](#spec.userConfig.pgaudit-property){: name='spec.userConfig.pgaudit-property'} (object). Deprecated. System-wide settings for the pgaudit extension. See below for [nested schema](#spec.userConfig.pgaudit).
- [`pgbouncer`](#spec.userConfig.pgbouncer-property){: name='spec.userConfig.pgbouncer-property'} (object). PGBouncer connection pooling settings. See below for [nested schema](#spec.userConfig.pgbouncer).
- [`pglookout`](#spec.userConfig.pglookout-property){: name='spec.userConfig.pglookout-property'} (object). System-wide settings for pglookout. See below for [nested schema](#spec.userConfig.pglookout).
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`project_to_fork_from`](#spec.userConfig.project_to_fork_from-property){: name='spec.userConfig.project_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created.
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`recovery_target_time`](#spec.userConfig.recovery_target_time-property){: name='spec.userConfig.recovery_target_time-property'} (string, Immutable, MaxLength: 32). Recovery target time when forking a service. This has effect only when a new service is being created.
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`service_to_fork_from`](#spec.userConfig.service_to_fork_from-property){: name='spec.userConfig.service_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created.
- [`shared_buffers_percentage`](#spec.userConfig.shared_buffers_percentage-property){: name='spec.userConfig.shared_buffers_percentage-property'} (number, Minimum: 20, Maximum: 60). Percentage of total RAM that the database server uses for shared memory buffers. Valid range is 20-60 (float), which corresponds to 20% - 60%. This setting adjusts the shared_buffers configuration value.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.
- [`synchronous_replication`](#spec.userConfig.synchronous_replication-property){: name='spec.userConfig.synchronous_replication-property'} (string, Enum: `off`, `quorum`). Synchronous replication type. Note that the service plan also needs to support synchronous replication.
- [`timescaledb`](#spec.userConfig.timescaledb-property){: name='spec.userConfig.timescaledb-property'} (object). System-wide settings for the timescaledb extension. See below for [nested schema](#spec.userConfig.timescaledb).
- [`variant`](#spec.userConfig.variant-property){: name='spec.userConfig.variant-property'} (string, Enum: `aiven`, `timescale`). Variant of the PostgreSQL service, may affect the features that are exposed by default.
- [`work_mem`](#spec.userConfig.work_mem-property){: name='spec.userConfig.work_mem-property'} (integer, Minimum: 1, Maximum: 1024). Sets the maximum amount of memory to be used by a query operation (such as a sort or hash table) before writing to temporary disk files, in MB. Default is 1MB + 0.075% of total RAM (up to 32MB).

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

CIDR address block, either as a string, or in a dict with an optional description field.

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
- [`ignore_roles`](#spec.userConfig.migration.ignore_roles-property){: name='spec.userConfig.migration.ignore_roles-property'} (string, MaxLength: 2048). Comma-separated list of database roles, which should be ignored during migration (supported by PostgreSQL only at the moment).
- [`method`](#spec.userConfig.migration.method-property){: name='spec.userConfig.migration.method-property'} (string, Enum: `dump`, `replication`). The migration method to be used (currently supported only by Redis, Dragonfly, MySQL and PostgreSQL service types).
- [`password`](#spec.userConfig.migration.password-property){: name='spec.userConfig.migration.password-property'} (string, MaxLength: 256). Password for authentication with the server where to migrate data from.
- [`ssl`](#spec.userConfig.migration.ssl-property){: name='spec.userConfig.migration.ssl-property'} (boolean). The server where to migrate data from is secured with SSL.
- [`username`](#spec.userConfig.migration.username-property){: name='spec.userConfig.migration.username-property'} (string, MaxLength: 256). User name for authentication with the server where to migrate data from.

### pg {: #spec.userConfig.pg }

_Appears on [`spec.userConfig`](#spec.userConfig)._

postgresql.conf configuration values.

**Optional**

- [`autovacuum_analyze_scale_factor`](#spec.userConfig.pg.autovacuum_analyze_scale_factor-property){: name='spec.userConfig.pg.autovacuum_analyze_scale_factor-property'} (number, Minimum: 0, Maximum: 1). Specifies a fraction of the table size to add to autovacuum_analyze_threshold when deciding whether to trigger an ANALYZE. The default is 0.2 (20% of table size).
- [`autovacuum_analyze_threshold`](#spec.userConfig.pg.autovacuum_analyze_threshold-property){: name='spec.userConfig.pg.autovacuum_analyze_threshold-property'} (integer, Minimum: 0, Maximum: 2147483647). Specifies the minimum number of inserted, updated or deleted tuples needed to trigger an ANALYZE in any one table. The default is 50 tuples.
- [`autovacuum_freeze_max_age`](#spec.userConfig.pg.autovacuum_freeze_max_age-property){: name='spec.userConfig.pg.autovacuum_freeze_max_age-property'} (integer, Minimum: 200000000, Maximum: 1500000000). Specifies the maximum age (in transactions) that a table's pg_class.relfrozenxid field can attain before a VACUUM operation is forced to prevent transaction ID wraparound within the table. Note that the system will launch autovacuum processes to prevent wraparound even when autovacuum is otherwise disabled. This parameter will cause the server to be restarted.
- [`autovacuum_max_workers`](#spec.userConfig.pg.autovacuum_max_workers-property){: name='spec.userConfig.pg.autovacuum_max_workers-property'} (integer, Minimum: 1, Maximum: 20). Specifies the maximum number of autovacuum processes (other than the autovacuum launcher) that may be running at any one time. The default is three. This parameter can only be set at server start.
- [`autovacuum_naptime`](#spec.userConfig.pg.autovacuum_naptime-property){: name='spec.userConfig.pg.autovacuum_naptime-property'} (integer, Minimum: 1, Maximum: 86400). Specifies the minimum delay between autovacuum runs on any given database. The delay is measured in seconds, and the default is one minute.
- [`autovacuum_vacuum_cost_delay`](#spec.userConfig.pg.autovacuum_vacuum_cost_delay-property){: name='spec.userConfig.pg.autovacuum_vacuum_cost_delay-property'} (integer, Minimum: -1, Maximum: 100). Specifies the cost delay value that will be used in automatic VACUUM operations. If -1 is specified, the regular vacuum_cost_delay value will be used. The default value is 20 milliseconds.
- [`autovacuum_vacuum_cost_limit`](#spec.userConfig.pg.autovacuum_vacuum_cost_limit-property){: name='spec.userConfig.pg.autovacuum_vacuum_cost_limit-property'} (integer, Minimum: -1, Maximum: 10000). Specifies the cost limit value that will be used in automatic VACUUM operations. If -1 is specified (which is the default), the regular vacuum_cost_limit value will be used.
- [`autovacuum_vacuum_scale_factor`](#spec.userConfig.pg.autovacuum_vacuum_scale_factor-property){: name='spec.userConfig.pg.autovacuum_vacuum_scale_factor-property'} (number, Minimum: 0, Maximum: 1). Specifies a fraction of the table size to add to autovacuum_vacuum_threshold when deciding whether to trigger a VACUUM. The default is 0.2 (20% of table size).
- [`autovacuum_vacuum_threshold`](#spec.userConfig.pg.autovacuum_vacuum_threshold-property){: name='spec.userConfig.pg.autovacuum_vacuum_threshold-property'} (integer, Minimum: 0, Maximum: 2147483647). Specifies the minimum number of updated or deleted tuples needed to trigger a VACUUM in any one table. The default is 50 tuples.
- [`bgwriter_delay`](#spec.userConfig.pg.bgwriter_delay-property){: name='spec.userConfig.pg.bgwriter_delay-property'} (integer, Minimum: 10, Maximum: 10000). Specifies the delay between activity rounds for the background writer in milliseconds. Default is 200.
- [`bgwriter_flush_after`](#spec.userConfig.pg.bgwriter_flush_after-property){: name='spec.userConfig.pg.bgwriter_flush_after-property'} (integer, Minimum: 0, Maximum: 2048). Whenever more than bgwriter_flush_after bytes have been written by the background writer, attempt to force the OS to issue these writes to the underlying storage. Specified in kilobytes, default is 512. Setting of 0 disables forced writeback.
- [`bgwriter_lru_maxpages`](#spec.userConfig.pg.bgwriter_lru_maxpages-property){: name='spec.userConfig.pg.bgwriter_lru_maxpages-property'} (integer, Minimum: 0, Maximum: 1073741823). In each round, no more than this many buffers will be written by the background writer. Setting this to zero disables background writing. Default is 100.
- [`bgwriter_lru_multiplier`](#spec.userConfig.pg.bgwriter_lru_multiplier-property){: name='spec.userConfig.pg.bgwriter_lru_multiplier-property'} (number, Minimum: 0, Maximum: 10). The average recent need for new buffers is multiplied by bgwriter_lru_multiplier to arrive at an estimate of the number that will be needed during the next round, (up to bgwriter_lru_maxpages). 1.0 represents a “just in time” policy of writing exactly the number of buffers predicted to be needed. Larger values provide some cushion against spikes in demand, while smaller values intentionally leave writes to be done by server processes. The default is 2.0.
- [`deadlock_timeout`](#spec.userConfig.pg.deadlock_timeout-property){: name='spec.userConfig.pg.deadlock_timeout-property'} (integer, Minimum: 500, Maximum: 1800000). This is the amount of time, in milliseconds, to wait on a lock before checking to see if there is a deadlock condition.
- [`default_toast_compression`](#spec.userConfig.pg.default_toast_compression-property){: name='spec.userConfig.pg.default_toast_compression-property'} (string, Enum: `lz4`, `pglz`). Specifies the default TOAST compression method for values of compressible columns (the default is lz4).
- [`idle_in_transaction_session_timeout`](#spec.userConfig.pg.idle_in_transaction_session_timeout-property){: name='spec.userConfig.pg.idle_in_transaction_session_timeout-property'} (integer, Minimum: 0, Maximum: 604800000). Time out sessions with open transactions after this number of milliseconds.
- [`jit`](#spec.userConfig.pg.jit-property){: name='spec.userConfig.pg.jit-property'} (boolean). Controls system-wide use of Just-in-Time Compilation (JIT).
- [`log_autovacuum_min_duration`](#spec.userConfig.pg.log_autovacuum_min_duration-property){: name='spec.userConfig.pg.log_autovacuum_min_duration-property'} (integer, Minimum: -1, Maximum: 2147483647). Causes each action executed by autovacuum to be logged if it ran for at least the specified number of milliseconds. Setting this to zero logs all autovacuum actions. Minus-one (the default) disables logging autovacuum actions.
- [`log_error_verbosity`](#spec.userConfig.pg.log_error_verbosity-property){: name='spec.userConfig.pg.log_error_verbosity-property'} (string, Enum: `DEFAULT`, `TERSE`, `VERBOSE`). Controls the amount of detail written in the server log for each message that is logged.
- [`log_line_prefix`](#spec.userConfig.pg.log_line_prefix-property){: name='spec.userConfig.pg.log_line_prefix-property'} (string, Enum: `'%m [%p] %q[user=%u,db=%d,app=%a] '`, `'%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h '`, `'pid=%p,user=%u,db=%d,app=%a,client=%h '`, `'pid=%p,user=%u,db=%d,app=%a,client=%h,txid=%x,qid=%Q '`). Choose from one of the available log formats.
- [`log_min_duration_statement`](#spec.userConfig.pg.log_min_duration_statement-property){: name='spec.userConfig.pg.log_min_duration_statement-property'} (integer, Minimum: -1, Maximum: 86400000). Log statements that take more than this number of milliseconds to run, -1 disables.
- [`log_temp_files`](#spec.userConfig.pg.log_temp_files-property){: name='spec.userConfig.pg.log_temp_files-property'} (integer, Minimum: -1, Maximum: 2147483647). Log statements for each temporary file created larger than this number of kilobytes, -1 disables.
- [`max_files_per_process`](#spec.userConfig.pg.max_files_per_process-property){: name='spec.userConfig.pg.max_files_per_process-property'} (integer, Minimum: 1000, Maximum: 4096). PostgreSQL maximum number of files that can be open per process.
- [`max_locks_per_transaction`](#spec.userConfig.pg.max_locks_per_transaction-property){: name='spec.userConfig.pg.max_locks_per_transaction-property'} (integer, Minimum: 64, Maximum: 6400). PostgreSQL maximum locks per transaction.
- [`max_logical_replication_workers`](#spec.userConfig.pg.max_logical_replication_workers-property){: name='spec.userConfig.pg.max_logical_replication_workers-property'} (integer, Minimum: 4, Maximum: 64). PostgreSQL maximum logical replication workers (taken from the pool of max_parallel_workers).
- [`max_parallel_workers`](#spec.userConfig.pg.max_parallel_workers-property){: name='spec.userConfig.pg.max_parallel_workers-property'} (integer, Minimum: 0, Maximum: 96). Sets the maximum number of workers that the system can support for parallel queries.
- [`max_parallel_workers_per_gather`](#spec.userConfig.pg.max_parallel_workers_per_gather-property){: name='spec.userConfig.pg.max_parallel_workers_per_gather-property'} (integer, Minimum: 0, Maximum: 96). Sets the maximum number of workers that can be started by a single Gather or Gather Merge node.
- [`max_pred_locks_per_transaction`](#spec.userConfig.pg.max_pred_locks_per_transaction-property){: name='spec.userConfig.pg.max_pred_locks_per_transaction-property'} (integer, Minimum: 64, Maximum: 5120). PostgreSQL maximum predicate locks per transaction.
- [`max_prepared_transactions`](#spec.userConfig.pg.max_prepared_transactions-property){: name='spec.userConfig.pg.max_prepared_transactions-property'} (integer, Minimum: 0, Maximum: 10000). PostgreSQL maximum prepared transactions.
- [`max_replication_slots`](#spec.userConfig.pg.max_replication_slots-property){: name='spec.userConfig.pg.max_replication_slots-property'} (integer, Minimum: 8, Maximum: 64). PostgreSQL maximum replication slots.
- [`max_slot_wal_keep_size`](#spec.userConfig.pg.max_slot_wal_keep_size-property){: name='spec.userConfig.pg.max_slot_wal_keep_size-property'} (integer, Minimum: -1, Maximum: 2147483647). PostgreSQL maximum WAL size (MB) reserved for replication slots. Default is -1 (unlimited). wal_keep_size minimum WAL size setting takes precedence over this.
- [`max_stack_depth`](#spec.userConfig.pg.max_stack_depth-property){: name='spec.userConfig.pg.max_stack_depth-property'} (integer, Minimum: 2097152, Maximum: 6291456). Maximum depth of the stack in bytes.
- [`max_standby_archive_delay`](#spec.userConfig.pg.max_standby_archive_delay-property){: name='spec.userConfig.pg.max_standby_archive_delay-property'} (integer, Minimum: 1, Maximum: 43200000). Max standby archive delay in milliseconds.
- [`max_standby_streaming_delay`](#spec.userConfig.pg.max_standby_streaming_delay-property){: name='spec.userConfig.pg.max_standby_streaming_delay-property'} (integer, Minimum: 1, Maximum: 43200000). Max standby streaming delay in milliseconds.
- [`max_wal_senders`](#spec.userConfig.pg.max_wal_senders-property){: name='spec.userConfig.pg.max_wal_senders-property'} (integer, Minimum: 20, Maximum: 64). PostgreSQL maximum WAL senders.
- [`max_worker_processes`](#spec.userConfig.pg.max_worker_processes-property){: name='spec.userConfig.pg.max_worker_processes-property'} (integer, Minimum: 8, Maximum: 96). Sets the maximum number of background processes that the system can support.
- [`password_encryption`](#spec.userConfig.pg.password_encryption-property){: name='spec.userConfig.pg.password_encryption-property'} (string, Enum: `md5`, `scram-sha-256`). Chooses the algorithm for encrypting passwords.
- [`pg_partman_bgw.interval`](#spec.userConfig.pg.pg_partman_bgw.interval-property){: name='spec.userConfig.pg.pg_partman_bgw.interval-property'} (integer, Minimum: 3600, Maximum: 604800). Sets the time interval to run pg_partman's scheduled tasks.
- [`pg_partman_bgw.role`](#spec.userConfig.pg.pg_partman_bgw.role-property){: name='spec.userConfig.pg.pg_partman_bgw.role-property'} (string, Pattern: `^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$`, MaxLength: 64). Controls which role to use for pg_partman's scheduled background tasks.
- [`pg_stat_monitor.pgsm_enable_query_plan`](#spec.userConfig.pg.pg_stat_monitor.pgsm_enable_query_plan-property){: name='spec.userConfig.pg.pg_stat_monitor.pgsm_enable_query_plan-property'} (boolean). Enables or disables query plan monitoring.
- [`pg_stat_monitor.pgsm_max_buckets`](#spec.userConfig.pg.pg_stat_monitor.pgsm_max_buckets-property){: name='spec.userConfig.pg.pg_stat_monitor.pgsm_max_buckets-property'} (integer, Minimum: 1, Maximum: 10). Sets the maximum number of buckets.
- [`pg_stat_statements.track`](#spec.userConfig.pg.pg_stat_statements.track-property){: name='spec.userConfig.pg.pg_stat_statements.track-property'} (string, Enum: `all`, `none`, `top`). Controls which statements are counted. Specify top to track top-level statements (those issued directly by clients), all to also track nested statements (such as statements invoked within functions), or none to disable statement statistics collection. The default value is top.
- [`temp_file_limit`](#spec.userConfig.pg.temp_file_limit-property){: name='spec.userConfig.pg.temp_file_limit-property'} (integer, Minimum: -1, Maximum: 2147483647). PostgreSQL temporary file limit in KiB, -1 for unlimited.
- [`timezone`](#spec.userConfig.pg.timezone-property){: name='spec.userConfig.pg.timezone-property'} (string, Pattern: `^[\w/]*$`, MaxLength: 64). PostgreSQL service timezone.
- [`track_activity_query_size`](#spec.userConfig.pg.track_activity_query_size-property){: name='spec.userConfig.pg.track_activity_query_size-property'} (integer, Minimum: 1024, Maximum: 10240). Specifies the number of bytes reserved to track the currently executing command for each active session.
- [`track_commit_timestamp`](#spec.userConfig.pg.track_commit_timestamp-property){: name='spec.userConfig.pg.track_commit_timestamp-property'} (string, Enum: `off`, `on`). Record commit time of transactions.
- [`track_functions`](#spec.userConfig.pg.track_functions-property){: name='spec.userConfig.pg.track_functions-property'} (string, Enum: `all`, `none`, `pl`). Enables tracking of function call counts and time used.
- [`track_io_timing`](#spec.userConfig.pg.track_io_timing-property){: name='spec.userConfig.pg.track_io_timing-property'} (string, Enum: `off`, `on`). Enables timing of database I/O calls. This parameter is off by default, because it will repeatedly query the operating system for the current time, which may cause significant overhead on some platforms.
- [`wal_sender_timeout`](#spec.userConfig.pg.wal_sender_timeout-property){: name='spec.userConfig.pg.wal_sender_timeout-property'} (integer). Terminate replication connections that are inactive for longer than this amount of time, in milliseconds. Setting this value to zero disables the timeout.
- [`wal_writer_delay`](#spec.userConfig.pg.wal_writer_delay-property){: name='spec.userConfig.pg.wal_writer_delay-property'} (integer, Minimum: 10, Maximum: 200). WAL flush interval in milliseconds. Note that setting this value to lower than the default 200ms may negatively impact performance.

### pg_qualstats {: #spec.userConfig.pg_qualstats }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Deprecated. System-wide settings for the pg_qualstats extension.

**Optional**

- [`enabled`](#spec.userConfig.pg_qualstats.enabled-property){: name='spec.userConfig.pg_qualstats.enabled-property'} (boolean). Deprecated. Enable / Disable pg_qualstats.
- [`min_err_estimate_num`](#spec.userConfig.pg_qualstats.min_err_estimate_num-property){: name='spec.userConfig.pg_qualstats.min_err_estimate_num-property'} (integer, Minimum: 0). Deprecated. Error estimation num threshold to save quals.
- [`min_err_estimate_ratio`](#spec.userConfig.pg_qualstats.min_err_estimate_ratio-property){: name='spec.userConfig.pg_qualstats.min_err_estimate_ratio-property'} (integer, Minimum: 0). Deprecated. Error estimation ratio threshold to save quals.
- [`track_constants`](#spec.userConfig.pg_qualstats.track_constants-property){: name='spec.userConfig.pg_qualstats.track_constants-property'} (boolean). Deprecated. Enable / Disable pg_qualstats constants tracking.
- [`track_pg_catalog`](#spec.userConfig.pg_qualstats.track_pg_catalog-property){: name='spec.userConfig.pg_qualstats.track_pg_catalog-property'} (boolean). Deprecated. Track quals on system catalogs too.

### pgaudit {: #spec.userConfig.pgaudit }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Deprecated. System-wide settings for the pgaudit extension.

**Optional**

- [`feature_enabled`](#spec.userConfig.pgaudit.feature_enabled-property){: name='spec.userConfig.pgaudit.feature_enabled-property'} (boolean). Deprecated. Enable pgaudit extension. When enabled, pgaudit extension will be automatically installed.Otherwise, extension will be uninstalled but auditing configurations will be preserved.
- [`log`](#spec.userConfig.pgaudit.log-property){: name='spec.userConfig.pgaudit.log-property'} (array of strings). Deprecated. Specifies which classes of statements will be logged by session audit logging.
- [`log_catalog`](#spec.userConfig.pgaudit.log_catalog-property){: name='spec.userConfig.pgaudit.log_catalog-property'} (boolean). Deprecated. Specifies that session logging should be enabled in the casewhere all relations in a statement are in pg_catalog.
- [`log_client`](#spec.userConfig.pgaudit.log_client-property){: name='spec.userConfig.pgaudit.log_client-property'} (boolean). Deprecated. Specifies whether log messages will be visible to a client process such as psql.
- [`log_level`](#spec.userConfig.pgaudit.log_level-property){: name='spec.userConfig.pgaudit.log_level-property'} (string). Deprecated. Specifies the log level that will be used for log entries.
- [`log_max_string_length`](#spec.userConfig.pgaudit.log_max_string_length-property){: name='spec.userConfig.pgaudit.log_max_string_length-property'} (integer, Minimum: -1, Maximum: 102400). Deprecated. Crop parameters representation and whole statements if they exceed this threshold. A (default) value of -1 disable the truncation.
- [`log_nested_statements`](#spec.userConfig.pgaudit.log_nested_statements-property){: name='spec.userConfig.pgaudit.log_nested_statements-property'} (boolean). Deprecated. This GUC allows to turn off logging nested statements, that is, statements that are executed as part of another ExecutorRun.
- [`log_parameter`](#spec.userConfig.pgaudit.log_parameter-property){: name='spec.userConfig.pgaudit.log_parameter-property'} (boolean). Deprecated. Specifies that audit logging should include the parameters that were passed with the statement.
- [`log_parameter_max_size`](#spec.userConfig.pgaudit.log_parameter_max_size-property){: name='spec.userConfig.pgaudit.log_parameter_max_size-property'} (integer). Deprecated. Specifies that parameter values longer than this setting (in bytes) should not be logged, but replaced with <long param suppressed>.
- [`log_relation`](#spec.userConfig.pgaudit.log_relation-property){: name='spec.userConfig.pgaudit.log_relation-property'} (boolean). Deprecated. Specifies whether session audit logging should create a separate log entry for each relation (TABLE, VIEW, etc.) referenced in a SELECT or DML statement.
- [`log_rows`](#spec.userConfig.pgaudit.log_rows-property){: name='spec.userConfig.pgaudit.log_rows-property'} (boolean). Deprecated. Specifies that audit logging should include the rows retrieved or affected by a statement. When enabled the rows field will be included after the parameter field.
- [`log_statement`](#spec.userConfig.pgaudit.log_statement-property){: name='spec.userConfig.pgaudit.log_statement-property'} (boolean). Deprecated. Specifies whether logging will include the statement text and parameters (if enabled).
- [`log_statement_once`](#spec.userConfig.pgaudit.log_statement_once-property){: name='spec.userConfig.pgaudit.log_statement_once-property'} (boolean). Deprecated. Specifies whether logging will include the statement text and parameters with the first log entry for a statement/substatement combination or with every entry.
- [`role`](#spec.userConfig.pgaudit.role-property){: name='spec.userConfig.pgaudit.role-property'} (string, Pattern: `^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$`, MaxLength: 64). Deprecated. Specifies the master role to use for object audit logging.

### pgbouncer {: #spec.userConfig.pgbouncer }

_Appears on [`spec.userConfig`](#spec.userConfig)._

PGBouncer connection pooling settings.

**Optional**

- [`autodb_idle_timeout`](#spec.userConfig.pgbouncer.autodb_idle_timeout-property){: name='spec.userConfig.pgbouncer.autodb_idle_timeout-property'} (integer, Minimum: 0, Maximum: 86400). If the automatically created database pools have been unused this many seconds, they are freed. If 0 then timeout is disabled. [seconds].
- [`autodb_max_db_connections`](#spec.userConfig.pgbouncer.autodb_max_db_connections-property){: name='spec.userConfig.pgbouncer.autodb_max_db_connections-property'} (integer, Minimum: 0, Maximum: 2147483647). Do not allow more than this many server connections per database (regardless of user). Setting it to 0 means unlimited.
- [`autodb_pool_mode`](#spec.userConfig.pgbouncer.autodb_pool_mode-property){: name='spec.userConfig.pgbouncer.autodb_pool_mode-property'} (string, Enum: `session`, `statement`, `transaction`). PGBouncer pool mode.
- [`autodb_pool_size`](#spec.userConfig.pgbouncer.autodb_pool_size-property){: name='spec.userConfig.pgbouncer.autodb_pool_size-property'} (integer, Minimum: 0, Maximum: 10000). If non-zero then create automatically a pool of that size per user when a pool doesn't exist.
- [`ignore_startup_parameters`](#spec.userConfig.pgbouncer.ignore_startup_parameters-property){: name='spec.userConfig.pgbouncer.ignore_startup_parameters-property'} (array of strings, MaxItems: 32). List of parameters to ignore when given in startup packet.
- [`max_prepared_statements`](#spec.userConfig.pgbouncer.max_prepared_statements-property){: name='spec.userConfig.pgbouncer.max_prepared_statements-property'} (integer, Minimum: 0, Maximum: 3000). PgBouncer tracks protocol-level named prepared statements related commands sent by the client in transaction and statement pooling modes when max_prepared_statements is set to a non-zero value. Setting it to 0 disables prepared statements. max_prepared_statements defaults to 100, and its maximum is 3000.
- [`min_pool_size`](#spec.userConfig.pgbouncer.min_pool_size-property){: name='spec.userConfig.pgbouncer.min_pool_size-property'} (integer, Minimum: 0, Maximum: 10000). Add more server connections to pool if below this number. Improves behavior when usual load comes suddenly back after period of total inactivity. The value is effectively capped at the pool size.
- [`server_idle_timeout`](#spec.userConfig.pgbouncer.server_idle_timeout-property){: name='spec.userConfig.pgbouncer.server_idle_timeout-property'} (integer, Minimum: 0, Maximum: 86400). If a server connection has been idle more than this many seconds it will be dropped. If 0 then timeout is disabled. [seconds].
- [`server_lifetime`](#spec.userConfig.pgbouncer.server_lifetime-property){: name='spec.userConfig.pgbouncer.server_lifetime-property'} (integer, Minimum: 60, Maximum: 86400). The pooler will close an unused server connection that has been connected longer than this. [seconds].
- [`server_reset_query_always`](#spec.userConfig.pgbouncer.server_reset_query_always-property){: name='spec.userConfig.pgbouncer.server_reset_query_always-property'} (boolean). Run server_reset_query (DISCARD ALL) in all pooling modes.

### pglookout {: #spec.userConfig.pglookout }

_Appears on [`spec.userConfig`](#spec.userConfig)._

System-wide settings for pglookout.

**Required**

- [`max_failover_replication_time_lag`](#spec.userConfig.pglookout.max_failover_replication_time_lag-property){: name='spec.userConfig.pglookout.max_failover_replication_time_lag-property'} (integer, Minimum: 10). Number of seconds of master unavailability before triggering database failover to standby.

### private_access {: #spec.userConfig.private_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from private networks.

**Optional**

- [`pg`](#spec.userConfig.private_access.pg-property){: name='spec.userConfig.private_access.pg-property'} (boolean). Allow clients to connect to pg with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`pgbouncer`](#spec.userConfig.private_access.pgbouncer-property){: name='spec.userConfig.private_access.pgbouncer-property'} (boolean). Allow clients to connect to pgbouncer with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service components through Privatelink.

**Optional**

- [`pg`](#spec.userConfig.privatelink_access.pg-property){: name='spec.userConfig.privatelink_access.pg-property'} (boolean). Enable pg.
- [`pgbouncer`](#spec.userConfig.privatelink_access.pgbouncer-property){: name='spec.userConfig.privatelink_access.pgbouncer-property'} (boolean). Enable pgbouncer.
- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Optional**

- [`pg`](#spec.userConfig.public_access.pg-property){: name='spec.userConfig.public_access.pg-property'} (boolean). Allow clients to connect to pg from the public internet for service nodes that are in a project VPC or another type of private network.
- [`pgbouncer`](#spec.userConfig.public_access.pgbouncer-property){: name='spec.userConfig.public_access.pgbouncer-property'} (boolean). Allow clients to connect to pgbouncer from the public internet for service nodes that are in a project VPC or another type of private network.
- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.

### timescaledb {: #spec.userConfig.timescaledb }

_Appears on [`spec.userConfig`](#spec.userConfig)._

System-wide settings for the timescaledb extension.

**Required**

- [`max_background_workers`](#spec.userConfig.timescaledb.max_background_workers-property){: name='spec.userConfig.timescaledb.max_background_workers-property'} (integer, Minimum: 1, Maximum: 4096). The number of background workers for timescaledb operations. You should configure this setting to the sum of your number of databases and the total number of concurrent background workers you want running at any given point in time.
