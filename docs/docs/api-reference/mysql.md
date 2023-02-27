---
title: "MySQL"
---

## Usage example

```yaml
apiVersion: aiven.io/v1alpha1
kind: MySQL
metadata:
  name: my-mysql
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: mysql-secret

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: business-4

  maintenanceWindowDow: sunday
  maintenanceWindowTime: 11:00:00

  userConfig:
    backup_hour: 17
    backup_minute: 11
    ip_filter:
      - network: 0.0.0.0
        description: whatever
      - network: 10.20.0.0/16
```

## MySQL {: #MySQL }

MySQL is the Schema for the mysqls API.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `MySQL`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). MySQLSpec defines the desired state of MySQL. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`MySQL`](#MySQL)._

MySQLSpec defines the desired state of MySQL.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, MaxLength: 63). Target project.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, MaxLength: 256). Cloud the service runs in.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Information regarding secret creation. See below for [nested schema](#spec.connInfoSecretTarget).
- [`disk_space`](#spec.disk_space-property){: name='spec.disk_space-property'} (string). The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
- [`maintenanceWindowDow`](#spec.maintenanceWindowDow-property){: name='spec.maintenanceWindowDow-property'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
- [`maintenanceWindowTime`](#spec.maintenanceWindowTime-property){: name='spec.maintenanceWindowTime-property'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
- [`plan`](#spec.plan-property){: name='spec.plan-property'} (string, MaxLength: 128). Subscription plan.
- [`projectVPCRef`](#spec.projectVPCRef-property){: name='spec.projectVPCRef-property'} (object, Immutable). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See below for [nested schema](#spec.projectVPCRef).
- [`projectVpcId`](#spec.projectVpcId-property){: name='spec.projectVpcId-property'} (string, Immutable, MaxLength: 36). Identifier of the VPC the service should be in, if any.
- [`serviceIntegrations`](#spec.serviceIntegrations-property){: name='spec.serviceIntegrations-property'} (array of objects, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See below for [nested schema](#spec.serviceIntegrations).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize services.
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services.
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). MySQL specific user configuration options. See below for [nested schema](#spec.userConfig).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1). 
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1). 

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

_Appears on [`spec`](#spec)._

Information regarding secret creation.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string). Name of the secret resource to be created. By default, is equal to the resource name.

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

## userConfig {: #spec.userConfig }

_Appears on [`spec`](#spec)._

MySQL specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Additional Cloud Regions for Backup Replication.
- [`admin_password`](#spec.userConfig.admin_password-property){: name='spec.userConfig.admin_password-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9-_]+$`, MinLength: 8, MaxLength: 256). Custom password for admin user. Defaults to random string. This must be set only when a new service is being created.
- [`admin_username`](#spec.userConfig.admin_username-property){: name='spec.userConfig.admin_username-property'} (string, Immutable, Pattern: `^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$`, MaxLength: 64). Custom username for admin user. This must be set only when a new service is being created.
- [`backup_hour`](#spec.userConfig.backup_hour-property){: name='spec.userConfig.backup_hour-property'} (integer, Minimum: 0, Maximum: 23). The hour of day (in UTC) when backup for the service is started. New backup is only started if previous backup has already completed.
- [`backup_minute`](#spec.userConfig.backup_minute-property){: name='spec.userConfig.backup_minute-property'} (integer, Minimum: 0, Maximum: 59). The minute of an hour when backup for the service is started. New backup is only started if previous backup has already completed.
- [`binlog_retention_period`](#spec.userConfig.binlog_retention_period-property){: name='spec.userConfig.binlog_retention_period-property'} (integer, Minimum: 600, Maximum: 86400). The minimum amount of time in seconds to keep binlog entries before deletion. This may be extended for services that require binlog entries for longer than the default for example if using the MySQL Debezium Kafka connector.
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See below for [nested schema](#spec.userConfig.ip_filter).
- [`migration`](#spec.userConfig.migration-property){: name='spec.userConfig.migration-property'} (object). Migrate data from existing server. See below for [nested schema](#spec.userConfig.migration).
- [`mysql`](#spec.userConfig.mysql-property){: name='spec.userConfig.mysql-property'} (object). mysql.conf configuration values. See below for [nested schema](#spec.userConfig.mysql).
- [`mysql_version`](#spec.userConfig.mysql_version-property){: name='spec.userConfig.mysql_version-property'} (string, Enum: `8`). MySQL major version.
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`project_to_fork_from`](#spec.userConfig.project_to_fork_from-property){: name='spec.userConfig.project_to_fork_from-property'} (string, Immutable, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created.
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`recovery_target_time`](#spec.userConfig.recovery_target_time-property){: name='spec.userConfig.recovery_target_time-property'} (string, Immutable, MaxLength: 32). Recovery target time when forking a service. This has effect only when a new service is being created.
- [`service_to_fork_from`](#spec.userConfig.service_to_fork_from-property){: name='spec.userConfig.service_to_fork_from-property'} (string, Immutable, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created.
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'.

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
- [`ignore_dbs`](#spec.userConfig.migration.ignore_dbs-property){: name='spec.userConfig.migration.ignore_dbs-property'} (string, MaxLength: 2048). Comma-separated list of databases, which should be ignored during migration (supported by MySQL only at the moment).
- [`method`](#spec.userConfig.migration.method-property){: name='spec.userConfig.migration.method-property'} (string, Enum: `dump`, `replication`). The migration method to be used (currently supported only by Redis and MySQL service types).
- [`password`](#spec.userConfig.migration.password-property){: name='spec.userConfig.migration.password-property'} (string, MaxLength: 256). Password for authentication with the server where to migrate data from.
- [`ssl`](#spec.userConfig.migration.ssl-property){: name='spec.userConfig.migration.ssl-property'} (boolean). The server where to migrate data from is secured with SSL.
- [`username`](#spec.userConfig.migration.username-property){: name='spec.userConfig.migration.username-property'} (string, MaxLength: 256). User name for authentication with the server where to migrate data from.

### mysql {: #spec.userConfig.mysql }

_Appears on [`spec.userConfig`](#spec.userConfig)._

mysql.conf configuration values.

**Optional**

- [`connect_timeout`](#spec.userConfig.mysql.connect_timeout-property){: name='spec.userConfig.mysql.connect_timeout-property'} (integer, Minimum: 2, Maximum: 3600). The number of seconds that the mysqld server waits for a connect packet before responding with Bad handshake.
- [`default_time_zone`](#spec.userConfig.mysql.default_time_zone-property){: name='spec.userConfig.mysql.default_time_zone-property'} (string, MinLength: 2, MaxLength: 100). Default server time zone as an offset from UTC (from -12:00 to +12:00), a time zone name, or 'SYSTEM' to use the MySQL server default.
- [`group_concat_max_len`](#spec.userConfig.mysql.group_concat_max_len-property){: name='spec.userConfig.mysql.group_concat_max_len-property'} (integer, Minimum: 4). The maximum permitted result length in bytes for the GROUP_CONCAT() function.
- [`information_schema_stats_expiry`](#spec.userConfig.mysql.information_schema_stats_expiry-property){: name='spec.userConfig.mysql.information_schema_stats_expiry-property'} (integer, Minimum: 900, Maximum: 31536000). The time, in seconds, before cached statistics expire.
- [`innodb_change_buffer_max_size`](#spec.userConfig.mysql.innodb_change_buffer_max_size-property){: name='spec.userConfig.mysql.innodb_change_buffer_max_size-property'} (integer, Minimum: 0, Maximum: 50). Maximum size for the InnoDB change buffer, as a percentage of the total size of the buffer pool. Default is 25.
- [`innodb_flush_neighbors`](#spec.userConfig.mysql.innodb_flush_neighbors-property){: name='spec.userConfig.mysql.innodb_flush_neighbors-property'} (integer, Minimum: 0, Maximum: 2). Specifies whether flushing a page from the InnoDB buffer pool also flushes other dirty pages in the same extent (default is 1): 0 - dirty pages in the same extent are not flushed,  1 - flush contiguous dirty pages in the same extent,  2 - flush dirty pages in the same extent.
- [`innodb_ft_min_token_size`](#spec.userConfig.mysql.innodb_ft_min_token_size-property){: name='spec.userConfig.mysql.innodb_ft_min_token_size-property'} (integer, Minimum: 0, Maximum: 16). Minimum length of words that are stored in an InnoDB FULLTEXT index. Changing this parameter will lead to a restart of the MySQL service.
- [`innodb_ft_server_stopword_table`](#spec.userConfig.mysql.innodb_ft_server_stopword_table-property){: name='spec.userConfig.mysql.innodb_ft_server_stopword_table-property'} (string, Pattern: `^.+/.+$`, MaxLength: 1024). This option is used to specify your own InnoDB FULLTEXT index stopword list for all InnoDB tables.
- [`innodb_lock_wait_timeout`](#spec.userConfig.mysql.innodb_lock_wait_timeout-property){: name='spec.userConfig.mysql.innodb_lock_wait_timeout-property'} (integer, Minimum: 1, Maximum: 3600). The length of time in seconds an InnoDB transaction waits for a row lock before giving up.
- [`innodb_log_buffer_size`](#spec.userConfig.mysql.innodb_log_buffer_size-property){: name='spec.userConfig.mysql.innodb_log_buffer_size-property'} (integer, Minimum: 1048576, Maximum: 4294967295). The size in bytes of the buffer that InnoDB uses to write to the log files on disk.
- [`innodb_online_alter_log_max_size`](#spec.userConfig.mysql.innodb_online_alter_log_max_size-property){: name='spec.userConfig.mysql.innodb_online_alter_log_max_size-property'} (integer, Minimum: 65536, Maximum: 1099511627776). The upper limit in bytes on the size of the temporary log files used during online DDL operations for InnoDB tables.
- [`innodb_print_all_deadlocks`](#spec.userConfig.mysql.innodb_print_all_deadlocks-property){: name='spec.userConfig.mysql.innodb_print_all_deadlocks-property'} (boolean). When enabled, information about all deadlocks in InnoDB user transactions is recorded in the error log. Disabled by default.
- [`innodb_read_io_threads`](#spec.userConfig.mysql.innodb_read_io_threads-property){: name='spec.userConfig.mysql.innodb_read_io_threads-property'} (integer, Minimum: 1, Maximum: 64). The number of I/O threads for read operations in InnoDB. Default is 4. Changing this parameter will lead to a restart of the MySQL service.
- [`innodb_rollback_on_timeout`](#spec.userConfig.mysql.innodb_rollback_on_timeout-property){: name='spec.userConfig.mysql.innodb_rollback_on_timeout-property'} (boolean). When enabled a transaction timeout causes InnoDB to abort and roll back the entire transaction. Changing this parameter will lead to a restart of the MySQL service.
- [`innodb_thread_concurrency`](#spec.userConfig.mysql.innodb_thread_concurrency-property){: name='spec.userConfig.mysql.innodb_thread_concurrency-property'} (integer, Minimum: 0, Maximum: 1000). Defines the maximum number of threads permitted inside of InnoDB. Default is 0 (infinite concurrency - no limit).
- [`innodb_write_io_threads`](#spec.userConfig.mysql.innodb_write_io_threads-property){: name='spec.userConfig.mysql.innodb_write_io_threads-property'} (integer, Minimum: 1, Maximum: 64). The number of I/O threads for write operations in InnoDB. Default is 4. Changing this parameter will lead to a restart of the MySQL service.
- [`interactive_timeout`](#spec.userConfig.mysql.interactive_timeout-property){: name='spec.userConfig.mysql.interactive_timeout-property'} (integer, Minimum: 30, Maximum: 604800). The number of seconds the server waits for activity on an interactive connection before closing it.
- [`internal_tmp_mem_storage_engine`](#spec.userConfig.mysql.internal_tmp_mem_storage_engine-property){: name='spec.userConfig.mysql.internal_tmp_mem_storage_engine-property'} (string, Enum: `TempTable`, `MEMORY`). The storage engine for in-memory internal temporary tables.
- [`long_query_time`](#spec.userConfig.mysql.long_query_time-property){: name='spec.userConfig.mysql.long_query_time-property'} (number). The slow_query_logs work as SQL statements that take more than long_query_time seconds to execute. Default is 10s.
- [`max_allowed_packet`](#spec.userConfig.mysql.max_allowed_packet-property){: name='spec.userConfig.mysql.max_allowed_packet-property'} (integer, Minimum: 102400, Maximum: 1073741824). Size of the largest message in bytes that can be received by the server. Default is 67108864 (64M).
- [`max_heap_table_size`](#spec.userConfig.mysql.max_heap_table_size-property){: name='spec.userConfig.mysql.max_heap_table_size-property'} (integer, Minimum: 1048576, Maximum: 1073741824). Limits the size of internal in-memory tables. Also set tmp_table_size. Default is 16777216 (16M).
- [`net_buffer_length`](#spec.userConfig.mysql.net_buffer_length-property){: name='spec.userConfig.mysql.net_buffer_length-property'} (integer, Minimum: 1024, Maximum: 1048576). Start sizes of connection buffer and result buffer. Default is 16384 (16K). Changing this parameter will lead to a restart of the MySQL service.
- [`net_read_timeout`](#spec.userConfig.mysql.net_read_timeout-property){: name='spec.userConfig.mysql.net_read_timeout-property'} (integer, Minimum: 1, Maximum: 3600). The number of seconds to wait for more data from a connection before aborting the read.
- [`net_write_timeout`](#spec.userConfig.mysql.net_write_timeout-property){: name='spec.userConfig.mysql.net_write_timeout-property'} (integer, Minimum: 1, Maximum: 3600). The number of seconds to wait for a block to be written to a connection before aborting the write.
- [`slow_query_log`](#spec.userConfig.mysql.slow_query_log-property){: name='spec.userConfig.mysql.slow_query_log-property'} (boolean). Slow query log enables capturing of slow queries. Setting slow_query_log to false also truncates the mysql.slow_log table. Default is off.
- [`sort_buffer_size`](#spec.userConfig.mysql.sort_buffer_size-property){: name='spec.userConfig.mysql.sort_buffer_size-property'} (integer, Minimum: 32768, Maximum: 1073741824). Sort buffer size in bytes for ORDER BY optimization. Default is 262144 (256K).
- [`sql_mode`](#spec.userConfig.mysql.sql_mode-property){: name='spec.userConfig.mysql.sql_mode-property'} (string, Pattern: `^[A-Z_]*(,[A-Z_]+)*$`, MaxLength: 1024). Global SQL mode. Set to empty to use MySQL server defaults. When creating a new service and not setting this field Aiven default SQL mode (strict, SQL standard compliant) will be assigned.
- [`sql_require_primary_key`](#spec.userConfig.mysql.sql_require_primary_key-property){: name='spec.userConfig.mysql.sql_require_primary_key-property'} (boolean). Require primary key to be defined for new tables or old tables modified with ALTER TABLE and fail if missing. It is recommended to always have primary keys because various functionality may break if any large table is missing them.
- [`tmp_table_size`](#spec.userConfig.mysql.tmp_table_size-property){: name='spec.userConfig.mysql.tmp_table_size-property'} (integer, Minimum: 1048576, Maximum: 1073741824). Limits the size of internal in-memory tables. Also set max_heap_table_size. Default is 16777216 (16M).
- [`wait_timeout`](#spec.userConfig.mysql.wait_timeout-property){: name='spec.userConfig.mysql.wait_timeout-property'} (integer, Minimum: 1, Maximum: 2147483). The number of seconds the server waits for activity on a noninteractive connection before closing it.

### private_access {: #spec.userConfig.private_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from private networks.

**Optional**

- [`mysql`](#spec.userConfig.private_access.mysql-property){: name='spec.userConfig.private_access.mysql-property'} (boolean). Allow clients to connect to mysql with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`mysqlx`](#spec.userConfig.private_access.mysqlx-property){: name='spec.userConfig.private_access.mysqlx-property'} (boolean). Allow clients to connect to mysqlx with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.
- [`prometheus`](#spec.userConfig.private_access.prometheus-property){: name='spec.userConfig.private_access.prometheus-property'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service components through Privatelink.

**Optional**

- [`mysql`](#spec.userConfig.privatelink_access.mysql-property){: name='spec.userConfig.privatelink_access.mysql-property'} (boolean). Enable mysql.
- [`mysqlx`](#spec.userConfig.privatelink_access.mysqlx-property){: name='spec.userConfig.privatelink_access.mysqlx-property'} (boolean). Enable mysqlx.
- [`prometheus`](#spec.userConfig.privatelink_access.prometheus-property){: name='spec.userConfig.privatelink_access.prometheus-property'} (boolean). Enable prometheus.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Optional**

- [`mysql`](#spec.userConfig.public_access.mysql-property){: name='spec.userConfig.public_access.mysql-property'} (boolean). Allow clients to connect to mysql from the public internet for service nodes that are in a project VPC or another type of private network.
- [`mysqlx`](#spec.userConfig.public_access.mysqlx-property){: name='spec.userConfig.public_access.mysqlx-property'} (boolean). Allow clients to connect to mysqlx from the public internet for service nodes that are in a project VPC or another type of private network.
- [`prometheus`](#spec.userConfig.public_access.prometheus-property){: name='spec.userConfig.public_access.prometheus-property'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network.

