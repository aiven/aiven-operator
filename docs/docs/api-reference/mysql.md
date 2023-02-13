---
title: "MySQL"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | MySQL |

MySQLSpec defines the desired state of MySQL.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`cloudName`](#cloudName){: name='cloudName'} (string, MaxLength: 256). Cloud the service runs in. 
- [`connInfoSecretTarget`](#connInfoSecretTarget){: name='connInfoSecretTarget'} (object). Information regarding secret creation. See [below for nested schema](#connInfoSecretTarget).
- [`disk_space`](#disk_space){: name='disk_space'} (string). The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing. 
- [`maintenanceWindowDow`](#maintenanceWindowDow){: name='maintenanceWindowDow'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc. 
- [`maintenanceWindowTime`](#maintenanceWindowTime){: name='maintenanceWindowTime'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format. 
- [`plan`](#plan){: name='plan'} (string, MaxLength: 128). Subscription plan. 
- [`project`](#project){: name='project'} (string, Immutable, MaxLength: 63). Target project. 
- [`projectVPCRef`](#projectVPCRef){: name='projectVPCRef'} (object, Immutable). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See [below for nested schema](#projectVPCRef).
- [`projectVpcId`](#projectVpcId){: name='projectVpcId'} (string, Immutable, MaxLength: 36). Identifier of the VPC the service should be in, if any. 
- [`serviceIntegrations`](#serviceIntegrations){: name='serviceIntegrations'} (array, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See [below for nested schema](#serviceIntegrations).
- [`tags`](#tags){: name='tags'} (object). Tags are key-value pairs that allow you to categorize services. 
- [`terminationProtection`](#terminationProtection){: name='terminationProtection'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services. 
- [`userConfig`](#userConfig){: name='userConfig'} (object). MySQL specific user configuration options. See [below for nested schema](#userConfig).

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

## connInfoSecretTarget {: #connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#name){: name='name'} (string). Name of the Secret resource to be created. 

## projectVPCRef {: #projectVPCRef }

ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically.

**Required**

- [`name`](#name){: name='name'} (string, MinLength: 1).  

**Optional**

- [`namespace`](#namespace){: name='namespace'} (string, MinLength: 1).  

## serviceIntegrations {: #serviceIntegrations }

Service integrations to specify when creating a service. Not applied after initial service creation.

**Required**

- [`integrationType`](#integrationType){: name='integrationType'} (string, Enum: `read_replica`).  
- [`sourceServiceName`](#sourceServiceName){: name='sourceServiceName'} (string, MinLength: 1, MaxLength: 64).  

## userConfig {: #userConfig }

MySQL specific user configuration options.

**Optional**

- [`additional_backup_regions`](#additional_backup_regions){: name='additional_backup_regions'} (array, MaxItems: 1). Additional Cloud Regions for Backup Replication. 
- [`admin_password`](#admin_password){: name='admin_password'} (string, Immutable, Pattern: `^[a-zA-Z0-9-_]+$`, MinLength: 8, MaxLength: 256). Custom password for admin user. Defaults to random string. This must be set only when a new service is being created. 
- [`admin_username`](#admin_username){: name='admin_username'} (string, Immutable, Pattern: `^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$`, MaxLength: 64). Custom username for admin user. This must be set only when a new service is being created. 
- [`backup_hour`](#backup_hour){: name='backup_hour'} (integer, Minimum: 0, Maximum: 23). The hour of day (in UTC) when backup for the service is started. New backup is only started if previous backup has already completed. 
- [`backup_minute`](#backup_minute){: name='backup_minute'} (integer, Minimum: 0, Maximum: 59). The minute of an hour when backup for the service is started. New backup is only started if previous backup has already completed. 
- [`binlog_retention_period`](#binlog_retention_period){: name='binlog_retention_period'} (integer, Minimum: 600, Maximum: 86400). The minimum amount of time in seconds to keep binlog entries before deletion. This may be extended for services that require binlog entries for longer than the default for example if using the MySQL Debezium Kafka connector. 
- [`ip_filter`](#ip_filter){: name='ip_filter'} (array, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See [below for nested schema](#ip_filter).
- [`migration`](#migration){: name='migration'} (object). Migrate data from existing server. See [below for nested schema](#migration).
- [`mysql`](#mysql){: name='mysql'} (object). mysql.conf configuration values. See [below for nested schema](#mysql).
- [`mysql_version`](#mysql_version){: name='mysql_version'} (string, Enum: `8`). MySQL major version. 
- [`private_access`](#private_access){: name='private_access'} (object). Allow access to selected service ports from private networks. See [below for nested schema](#private_access).
- [`privatelink_access`](#privatelink_access){: name='privatelink_access'} (object). Allow access to selected service components through Privatelink. See [below for nested schema](#privatelink_access).
- [`project_to_fork_from`](#project_to_fork_from){: name='project_to_fork_from'} (string, Immutable, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created. 
- [`public_access`](#public_access){: name='public_access'} (object). Allow access to selected service ports from the public Internet. See [below for nested schema](#public_access).
- [`recovery_target_time`](#recovery_target_time){: name='recovery_target_time'} (string, Immutable, MaxLength: 32). Recovery target time when forking a service. This has effect only when a new service is being created. 
- [`service_to_fork_from`](#service_to_fork_from){: name='service_to_fork_from'} (string, Immutable, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created. 
- [`static_ips`](#static_ips){: name='static_ips'} (boolean). Use static public IP addresses. 

### ip_filter {: #ip_filter }

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#network){: name='network'} (string, MaxLength: 43). CIDR address block. 

**Optional**

- [`description`](#description){: name='description'} (string, MaxLength: 1024). Description for IP filter list entry. 

### migration {: #migration }

Migrate data from existing server.

**Required**

- [`host`](#host){: name='host'} (string, MaxLength: 255). Hostname or IP address of the server where to migrate data from. 
- [`port`](#port){: name='port'} (integer, Minimum: 1, Maximum: 65535). Port number of the server where to migrate data from. 

**Optional**

- [`dbname`](#dbname){: name='dbname'} (string, MaxLength: 63). Database name for bootstrapping the initial connection. 
- [`ignore_dbs`](#ignore_dbs){: name='ignore_dbs'} (string, MaxLength: 2048). Comma-separated list of databases, which should be ignored during migration (supported by MySQL only at the moment). 
- [`method`](#method){: name='method'} (string, Enum: `dump`, `replication`). The migration method to be used (currently supported only by Redis and MySQL service types). 
- [`password`](#password){: name='password'} (string, MaxLength: 256). Password for authentication with the server where to migrate data from. 
- [`ssl`](#ssl){: name='ssl'} (boolean). The server where to migrate data from is secured with SSL. 
- [`username`](#username){: name='username'} (string, MaxLength: 256). User name for authentication with the server where to migrate data from. 

### mysql {: #mysql }

mysql.conf configuration values.

**Optional**

- [`connect_timeout`](#connect_timeout){: name='connect_timeout'} (integer, Minimum: 2, Maximum: 3600). The number of seconds that the mysqld server waits for a connect packet before responding with Bad handshake. 
- [`default_time_zone`](#default_time_zone){: name='default_time_zone'} (string, MinLength: 2, MaxLength: 100). Default server time zone as an offset from UTC (from -12:00 to +12:00), a time zone name, or 'SYSTEM' to use the MySQL server default. 
- [`group_concat_max_len`](#group_concat_max_len){: name='group_concat_max_len'} (integer, Minimum: 4). The maximum permitted result length in bytes for the GROUP_CONCAT() function. 
- [`information_schema_stats_expiry`](#information_schema_stats_expiry){: name='information_schema_stats_expiry'} (integer, Minimum: 900, Maximum: 31536000). The time, in seconds, before cached statistics expire. 
- [`innodb_change_buffer_max_size`](#innodb_change_buffer_max_size){: name='innodb_change_buffer_max_size'} (integer, Minimum: 0, Maximum: 50). Maximum size for the InnoDB change buffer, as a percentage of the total size of the buffer pool. Default is 25. 
- [`innodb_flush_neighbors`](#innodb_flush_neighbors){: name='innodb_flush_neighbors'} (integer, Minimum: 0, Maximum: 2). Specifies whether flushing a page from the InnoDB buffer pool also flushes other dirty pages in the same extent (default is 1): 0 - dirty pages in the same extent are not flushed,  1 - flush contiguous dirty pages in the same extent,  2 - flush dirty pages in the same extent. 
- [`innodb_ft_min_token_size`](#innodb_ft_min_token_size){: name='innodb_ft_min_token_size'} (integer, Minimum: 0, Maximum: 16). Minimum length of words that are stored in an InnoDB FULLTEXT index. Changing this parameter will lead to a restart of the MySQL service. 
- [`innodb_ft_server_stopword_table`](#innodb_ft_server_stopword_table){: name='innodb_ft_server_stopword_table'} (string, Pattern: `^.+/.+$`, MaxLength: 1024). This option is used to specify your own InnoDB FULLTEXT index stopword list for all InnoDB tables. 
- [`innodb_lock_wait_timeout`](#innodb_lock_wait_timeout){: name='innodb_lock_wait_timeout'} (integer, Minimum: 1, Maximum: 3600). The length of time in seconds an InnoDB transaction waits for a row lock before giving up. 
- [`innodb_log_buffer_size`](#innodb_log_buffer_size){: name='innodb_log_buffer_size'} (integer, Minimum: 1048576, Maximum: 4294967295). The size in bytes of the buffer that InnoDB uses to write to the log files on disk. 
- [`innodb_online_alter_log_max_size`](#innodb_online_alter_log_max_size){: name='innodb_online_alter_log_max_size'} (integer, Minimum: 65536, Maximum: 1099511627776). The upper limit in bytes on the size of the temporary log files used during online DDL operations for InnoDB tables. 
- [`innodb_print_all_deadlocks`](#innodb_print_all_deadlocks){: name='innodb_print_all_deadlocks'} (boolean). When enabled, information about all deadlocks in InnoDB user transactions is recorded in the error log. Disabled by default. 
- [`innodb_read_io_threads`](#innodb_read_io_threads){: name='innodb_read_io_threads'} (integer, Minimum: 1, Maximum: 64). The number of I/O threads for read operations in InnoDB. Default is 4. Changing this parameter will lead to a restart of the MySQL service. 
- [`innodb_rollback_on_timeout`](#innodb_rollback_on_timeout){: name='innodb_rollback_on_timeout'} (boolean). When enabled a transaction timeout causes InnoDB to abort and roll back the entire transaction. Changing this parameter will lead to a restart of the MySQL service. 
- [`innodb_thread_concurrency`](#innodb_thread_concurrency){: name='innodb_thread_concurrency'} (integer, Minimum: 0, Maximum: 1000). Defines the maximum number of threads permitted inside of InnoDB. Default is 0 (infinite concurrency - no limit). 
- [`innodb_write_io_threads`](#innodb_write_io_threads){: name='innodb_write_io_threads'} (integer, Minimum: 1, Maximum: 64). The number of I/O threads for write operations in InnoDB. Default is 4. Changing this parameter will lead to a restart of the MySQL service. 
- [`interactive_timeout`](#interactive_timeout){: name='interactive_timeout'} (integer, Minimum: 30, Maximum: 604800). The number of seconds the server waits for activity on an interactive connection before closing it. 
- [`internal_tmp_mem_storage_engine`](#internal_tmp_mem_storage_engine){: name='internal_tmp_mem_storage_engine'} (string, Enum: `TempTable`, `MEMORY`). The storage engine for in-memory internal temporary tables. 
- [`long_query_time`](#long_query_time){: name='long_query_time'} (number). The slow_query_logs work as SQL statements that take more than long_query_time seconds to execute. Default is 10s. 
- [`max_allowed_packet`](#max_allowed_packet){: name='max_allowed_packet'} (integer, Minimum: 102400, Maximum: 1073741824). Size of the largest message in bytes that can be received by the server. Default is 67108864 (64M). 
- [`max_heap_table_size`](#max_heap_table_size){: name='max_heap_table_size'} (integer, Minimum: 1048576, Maximum: 1073741824). Limits the size of internal in-memory tables. Also set tmp_table_size. Default is 16777216 (16M). 
- [`net_buffer_length`](#net_buffer_length){: name='net_buffer_length'} (integer, Minimum: 1024, Maximum: 1048576). Start sizes of connection buffer and result buffer. Default is 16384 (16K). Changing this parameter will lead to a restart of the MySQL service. 
- [`net_read_timeout`](#net_read_timeout){: name='net_read_timeout'} (integer, Minimum: 1, Maximum: 3600). The number of seconds to wait for more data from a connection before aborting the read. 
- [`net_write_timeout`](#net_write_timeout){: name='net_write_timeout'} (integer, Minimum: 1, Maximum: 3600). The number of seconds to wait for a block to be written to a connection before aborting the write. 
- [`slow_query_log`](#slow_query_log){: name='slow_query_log'} (boolean). Slow query log enables capturing of slow queries. Setting slow_query_log to false also truncates the mysql.slow_log table. Default is off. 
- [`sort_buffer_size`](#sort_buffer_size){: name='sort_buffer_size'} (integer, Minimum: 32768, Maximum: 1073741824). Sort buffer size in bytes for ORDER BY optimization. Default is 262144 (256K). 
- [`sql_mode`](#sql_mode){: name='sql_mode'} (string, Pattern: `^[A-Z_]*(,[A-Z_]+)*$`, MaxLength: 1024). Global SQL mode. Set to empty to use MySQL server defaults. When creating a new service and not setting this field Aiven default SQL mode (strict, SQL standard compliant) will be assigned. 
- [`sql_require_primary_key`](#sql_require_primary_key){: name='sql_require_primary_key'} (boolean). Require primary key to be defined for new tables or old tables modified with ALTER TABLE and fail if missing. It is recommended to always have primary keys because various functionality may break if any large table is missing them. 
- [`tmp_table_size`](#tmp_table_size){: name='tmp_table_size'} (integer, Minimum: 1048576, Maximum: 1073741824). Limits the size of internal in-memory tables. Also set max_heap_table_size. Default is 16777216 (16M). 
- [`wait_timeout`](#wait_timeout){: name='wait_timeout'} (integer, Minimum: 1, Maximum: 2147483). The number of seconds the server waits for activity on a noninteractive connection before closing it. 

### private_access {: #private_access }

Allow access to selected service ports from private networks.

**Optional**

- [`mysql`](#mysql){: name='mysql'} (boolean). Allow clients to connect to mysql with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`mysqlx`](#mysqlx){: name='mysqlx'} (boolean). Allow clients to connect to mysqlx with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 

### privatelink_access {: #privatelink_access }

Allow access to selected service components through Privatelink.

**Optional**

- [`mysql`](#mysql){: name='mysql'} (boolean). Enable mysql. 
- [`mysqlx`](#mysqlx){: name='mysqlx'} (boolean). Enable mysqlx. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Enable prometheus. 

### public_access {: #public_access }

Allow access to selected service ports from the public Internet.

**Optional**

- [`mysql`](#mysql){: name='mysql'} (boolean). Allow clients to connect to mysql from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`mysqlx`](#mysqlx){: name='mysqlx'} (boolean). Allow clients to connect to mysqlx from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network. 

