---
title: "PostgreSQL"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | PostgreSQL |

PostgreSQLSpec defines the desired state of postgres instance.

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
- [`userConfig`](#userConfig){: name='userConfig'} (object). PostgreSQL specific user configuration options. See [below for nested schema](#userConfig).

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

PostgreSQL specific user configuration options.

**Optional**

- [`additional_backup_regions`](#additional_backup_regions){: name='additional_backup_regions'} (array, MaxItems: 1). Additional Cloud Regions for Backup Replication. 
- [`admin_password`](#admin_password){: name='admin_password'} (string, Immutable, Pattern: `^[a-zA-Z0-9-_]+$`, MinLength: 8, MaxLength: 256). Custom password for admin user. Defaults to random string. This must be set only when a new service is being created. 
- [`admin_username`](#admin_username){: name='admin_username'} (string, Immutable, Pattern: `^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$`, MaxLength: 64). Custom username for admin user. This must be set only when a new service is being created. 
- [`backup_hour`](#backup_hour){: name='backup_hour'} (integer, Minimum: 0, Maximum: 23). The hour of day (in UTC) when backup for the service is started. New backup is only started if previous backup has already completed. 
- [`backup_minute`](#backup_minute){: name='backup_minute'} (integer, Minimum: 0, Maximum: 59). The minute of an hour when backup for the service is started. New backup is only started if previous backup has already completed. 
- [`enable_ipv6`](#enable_ipv6){: name='enable_ipv6'} (boolean). Register AAAA DNS records for the service, and allow IPv6 packets to service ports. 
- [`ip_filter`](#ip_filter){: name='ip_filter'} (array, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See [below for nested schema](#ip_filter).
- [`migration`](#migration){: name='migration'} (object). Migrate data from existing server. See [below for nested schema](#migration).
- [`pg`](#pg){: name='pg'} (object). postgresql.conf configuration values. See [below for nested schema](#pg).
- [`pg_read_replica`](#pg_read_replica){: name='pg_read_replica'} (boolean). Should the service which is being forked be a read replica (deprecated, use read_replica service integration instead). 
- [`pg_service_to_fork_from`](#pg_service_to_fork_from){: name='pg_service_to_fork_from'} (string, Immutable, MaxLength: 64). Name of the PG Service from which to fork (deprecated, use service_to_fork_from). This has effect only when a new service is being created. 
- [`pg_stat_monitor_enable`](#pg_stat_monitor_enable){: name='pg_stat_monitor_enable'} (boolean). Enable the pg_stat_monitor extension. Enabling this extension will cause the cluster to be restarted.When this extension is enabled, pg_stat_statements results for utility commands are unreliable. 
- [`pg_version`](#pg_version){: name='pg_version'} (string, Enum: `11`, `12`, `13`, `14`, `15`). PostgreSQL major version. 
- [`pgbouncer`](#pgbouncer){: name='pgbouncer'} (object). PGBouncer connection pooling settings. See [below for nested schema](#pgbouncer).
- [`pglookout`](#pglookout){: name='pglookout'} (object). PGLookout settings. See [below for nested schema](#pglookout).
- [`private_access`](#private_access){: name='private_access'} (object). Allow access to selected service ports from private networks. See [below for nested schema](#private_access).
- [`privatelink_access`](#privatelink_access){: name='privatelink_access'} (object). Allow access to selected service components through Privatelink. See [below for nested schema](#privatelink_access).
- [`project_to_fork_from`](#project_to_fork_from){: name='project_to_fork_from'} (string, Immutable, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created. 
- [`public_access`](#public_access){: name='public_access'} (object). Allow access to selected service ports from the public Internet. See [below for nested schema](#public_access).
- [`recovery_target_time`](#recovery_target_time){: name='recovery_target_time'} (string, Immutable, MaxLength: 32). Recovery target time when forking a service. This has effect only when a new service is being created. 
- [`service_to_fork_from`](#service_to_fork_from){: name='service_to_fork_from'} (string, Immutable, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created. 
- [`shared_buffers_percentage`](#shared_buffers_percentage){: name='shared_buffers_percentage'} (number). Percentage of total RAM that the database server uses for shared memory buffers. Valid range is 20-60 (float), which corresponds to 20% - 60%. This setting adjusts the shared_buffers configuration value. 
- [`static_ips`](#static_ips){: name='static_ips'} (boolean). Use static public IP addresses. 
- [`synchronous_replication`](#synchronous_replication){: name='synchronous_replication'} (string, Enum: `quorum`, `off`). Synchronous replication type. Note that the service plan also needs to support synchronous replication. 
- [`timescaledb`](#timescaledb){: name='timescaledb'} (object). TimescaleDB extension configuration values. See [below for nested schema](#timescaledb).
- [`variant`](#variant){: name='variant'} (string, Enum: `aiven`, `timescale`). Variant of the PostgreSQL service, may affect the features that are exposed by default. 
- [`work_mem`](#work_mem){: name='work_mem'} (integer, Minimum: 1, Maximum: 1024). Sets the maximum amount of memory to be used by a query operation (such as a sort or hash table) before writing to temporary disk files, in MB. Default is 1MB + 0.075% of total RAM (up to 32MB). 

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

### pg {: #pg }

postgresql.conf configuration values.

**Optional**

- [`autovacuum_analyze_scale_factor`](#autovacuum_analyze_scale_factor){: name='autovacuum_analyze_scale_factor'} (number). Specifies a fraction of the table size to add to autovacuum_analyze_threshold when deciding whether to trigger an ANALYZE. The default is 0.2 (20% of table size). 
- [`autovacuum_analyze_threshold`](#autovacuum_analyze_threshold){: name='autovacuum_analyze_threshold'} (integer, Minimum: 0, Maximum: 2147483647). Specifies the minimum number of inserted, updated or deleted tuples needed to trigger an  ANALYZE in any one table. The default is 50 tuples. 
- [`autovacuum_freeze_max_age`](#autovacuum_freeze_max_age){: name='autovacuum_freeze_max_age'} (integer, Minimum: 200000000, Maximum: 1500000000). Specifies the maximum age (in transactions) that a table's pg_class.relfrozenxid field can attain before a VACUUM operation is forced to prevent transaction ID wraparound within the table. Note that the system will launch autovacuum processes to prevent wraparound even when autovacuum is otherwise disabled. This parameter will cause the server to be restarted. 
- [`autovacuum_max_workers`](#autovacuum_max_workers){: name='autovacuum_max_workers'} (integer, Minimum: 1, Maximum: 20). Specifies the maximum number of autovacuum processes (other than the autovacuum launcher) that may be running at any one time. The default is three. This parameter can only be set at server start. 
- [`autovacuum_naptime`](#autovacuum_naptime){: name='autovacuum_naptime'} (integer, Minimum: 1, Maximum: 86400). Specifies the minimum delay between autovacuum runs on any given database. The delay is measured in seconds, and the default is one minute. 
- [`autovacuum_vacuum_cost_delay`](#autovacuum_vacuum_cost_delay){: name='autovacuum_vacuum_cost_delay'} (integer, Minimum: -1, Maximum: 100). Specifies the cost delay value that will be used in automatic VACUUM operations. If -1 is specified, the regular vacuum_cost_delay value will be used. The default value is 20 milliseconds. 
- [`autovacuum_vacuum_cost_limit`](#autovacuum_vacuum_cost_limit){: name='autovacuum_vacuum_cost_limit'} (integer, Minimum: -1, Maximum: 10000). Specifies the cost limit value that will be used in automatic VACUUM operations. If -1 is specified (which is the default), the regular vacuum_cost_limit value will be used. 
- [`autovacuum_vacuum_scale_factor`](#autovacuum_vacuum_scale_factor){: name='autovacuum_vacuum_scale_factor'} (number). Specifies a fraction of the table size to add to autovacuum_vacuum_threshold when deciding whether to trigger a VACUUM. The default is 0.2 (20% of table size). 
- [`autovacuum_vacuum_threshold`](#autovacuum_vacuum_threshold){: name='autovacuum_vacuum_threshold'} (integer, Minimum: 0, Maximum: 2147483647). Specifies the minimum number of updated or deleted tuples needed to trigger a VACUUM in any one table. The default is 50 tuples. 
- [`bgwriter_delay`](#bgwriter_delay){: name='bgwriter_delay'} (integer, Minimum: 10, Maximum: 10000). Specifies the delay between activity rounds for the background writer in milliseconds. Default is 200. 
- [`bgwriter_flush_after`](#bgwriter_flush_after){: name='bgwriter_flush_after'} (integer, Minimum: 0, Maximum: 2048). Whenever more than bgwriter_flush_after bytes have been written by the background writer, attempt to force the OS to issue these writes to the underlying storage. Specified in kilobytes, default is 512. Setting of 0 disables forced writeback. 
- [`bgwriter_lru_maxpages`](#bgwriter_lru_maxpages){: name='bgwriter_lru_maxpages'} (integer, Minimum: 0, Maximum: 1073741823). In each round, no more than this many buffers will be written by the background writer. Setting this to zero disables background writing. Default is 100. 
- [`bgwriter_lru_multiplier`](#bgwriter_lru_multiplier){: name='bgwriter_lru_multiplier'} (number). The average recent need for new buffers is multiplied by bgwriter_lru_multiplier to arrive at an estimate of the number that will be needed during the next round, (up to bgwriter_lru_maxpages). 1.0 represents a “just in time” policy of writing exactly the number of buffers predicted to be needed. Larger values provide some cushion against spikes in demand, while smaller values intentionally leave writes to be done by server processes. The default is 2.0. 
- [`deadlock_timeout`](#deadlock_timeout){: name='deadlock_timeout'} (integer, Minimum: 500, Maximum: 1800000). This is the amount of time, in milliseconds, to wait on a lock before checking to see if there is a deadlock condition. 
- [`default_toast_compression`](#default_toast_compression){: name='default_toast_compression'} (string, Enum: `lz4`, `pglz`). Specifies the default TOAST compression method for values of compressible columns (the default is lz4). 
- [`idle_in_transaction_session_timeout`](#idle_in_transaction_session_timeout){: name='idle_in_transaction_session_timeout'} (integer, Minimum: 0, Maximum: 604800000). Time out sessions with open transactions after this number of milliseconds. 
- [`jit`](#jit){: name='jit'} (boolean). Controls system-wide use of Just-in-Time Compilation (JIT). 
- [`log_autovacuum_min_duration`](#log_autovacuum_min_duration){: name='log_autovacuum_min_duration'} (integer, Minimum: -1, Maximum: 2147483647). Causes each action executed by autovacuum to be logged if it ran for at least the specified number of milliseconds. Setting this to zero logs all autovacuum actions. Minus-one (the default) disables logging autovacuum actions. 
- [`log_error_verbosity`](#log_error_verbosity){: name='log_error_verbosity'} (string, Enum: `TERSE`, `DEFAULT`, `VERBOSE`). Controls the amount of detail written in the server log for each message that is logged. 
- [`log_line_prefix`](#log_line_prefix){: name='log_line_prefix'} (string, Enum: `'pid=%p,user=%u,db=%d,app=%a,client=%h '`, `'%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h '`, `'%m [%p] %q[user=%u,db=%d,app=%a] '`). Choose from one of the available log-formats. These can support popular log analyzers like pgbadger, pganalyze etc. 
- [`log_min_duration_statement`](#log_min_duration_statement){: name='log_min_duration_statement'} (integer, Minimum: -1, Maximum: 86400000). Log statements that take more than this number of milliseconds to run, -1 disables. 
- [`log_temp_files`](#log_temp_files){: name='log_temp_files'} (integer, Minimum: -1, Maximum: 2147483647). Log statements for each temporary file created larger than this number of kilobytes, -1 disables. 
- [`max_files_per_process`](#max_files_per_process){: name='max_files_per_process'} (integer, Minimum: 1000, Maximum: 4096). PostgreSQL maximum number of files that can be open per process. 
- [`max_locks_per_transaction`](#max_locks_per_transaction){: name='max_locks_per_transaction'} (integer, Minimum: 64, Maximum: 6400). PostgreSQL maximum locks per transaction. 
- [`max_logical_replication_workers`](#max_logical_replication_workers){: name='max_logical_replication_workers'} (integer, Minimum: 4, Maximum: 64). PostgreSQL maximum logical replication workers (taken from the pool of max_parallel_workers). 
- [`max_parallel_workers`](#max_parallel_workers){: name='max_parallel_workers'} (integer, Minimum: 0, Maximum: 96). Sets the maximum number of workers that the system can support for parallel queries. 
- [`max_parallel_workers_per_gather`](#max_parallel_workers_per_gather){: name='max_parallel_workers_per_gather'} (integer, Minimum: 0, Maximum: 96). Sets the maximum number of workers that can be started by a single Gather or Gather Merge node. 
- [`max_pred_locks_per_transaction`](#max_pred_locks_per_transaction){: name='max_pred_locks_per_transaction'} (integer, Minimum: 64, Maximum: 5120). PostgreSQL maximum predicate locks per transaction. 
- [`max_prepared_transactions`](#max_prepared_transactions){: name='max_prepared_transactions'} (integer, Minimum: 0, Maximum: 10000). PostgreSQL maximum prepared transactions. 
- [`max_replication_slots`](#max_replication_slots){: name='max_replication_slots'} (integer, Minimum: 8, Maximum: 64). PostgreSQL maximum replication slots. 
- [`max_slot_wal_keep_size`](#max_slot_wal_keep_size){: name='max_slot_wal_keep_size'} (integer, Minimum: -1, Maximum: 2147483647). PostgreSQL maximum WAL size (MB) reserved for replication slots. Default is -1 (unlimited). wal_keep_size minimum WAL size setting takes precedence over this. 
- [`max_stack_depth`](#max_stack_depth){: name='max_stack_depth'} (integer, Minimum: 2097152, Maximum: 6291456). Maximum depth of the stack in bytes. 
- [`max_standby_archive_delay`](#max_standby_archive_delay){: name='max_standby_archive_delay'} (integer, Minimum: 1, Maximum: 43200000). Max standby archive delay in milliseconds. 
- [`max_standby_streaming_delay`](#max_standby_streaming_delay){: name='max_standby_streaming_delay'} (integer, Minimum: 1, Maximum: 43200000). Max standby streaming delay in milliseconds. 
- [`max_wal_senders`](#max_wal_senders){: name='max_wal_senders'} (integer, Minimum: 20, Maximum: 64). PostgreSQL maximum WAL senders. 
- [`max_worker_processes`](#max_worker_processes){: name='max_worker_processes'} (integer, Minimum: 8, Maximum: 96). Sets the maximum number of background processes that the system can support. 
- [`pg_partman_bgw.interval`](#pg_partman_bgw.interval){: name='pg_partman_bgw.interval'} (integer, Minimum: 3600, Maximum: 604800). Sets the time interval to run pg_partman's scheduled tasks. 
- [`pg_partman_bgw.role`](#pg_partman_bgw.role){: name='pg_partman_bgw.role'} (string, Pattern: `^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$`, MaxLength: 64). Controls which role to use for pg_partman's scheduled background tasks. 
- [`pg_stat_monitor.pgsm_enable_query_plan`](#pg_stat_monitor.pgsm_enable_query_plan){: name='pg_stat_monitor.pgsm_enable_query_plan'} (boolean). Enables or disables query plan monitoring. 
- [`pg_stat_monitor.pgsm_max_buckets`](#pg_stat_monitor.pgsm_max_buckets){: name='pg_stat_monitor.pgsm_max_buckets'} (integer, Minimum: 1, Maximum: 10). Sets the maximum number of buckets. 
- [`pg_stat_statements.track`](#pg_stat_statements.track){: name='pg_stat_statements.track'} (string, Enum: `all`, `top`, `none`). Controls which statements are counted. Specify top to track top-level statements (those issued directly by clients), all to also track nested statements (such as statements invoked within functions), or none to disable statement statistics collection. The default value is top. 
- [`temp_file_limit`](#temp_file_limit){: name='temp_file_limit'} (integer, Minimum: -1, Maximum: 2147483647). PostgreSQL temporary file limit in KiB, -1 for unlimited. 
- [`timezone`](#timezone){: name='timezone'} (string, MaxLength: 64). PostgreSQL service timezone. 
- [`track_activity_query_size`](#track_activity_query_size){: name='track_activity_query_size'} (integer, Minimum: 1024, Maximum: 10240). Specifies the number of bytes reserved to track the currently executing command for each active session. 
- [`track_commit_timestamp`](#track_commit_timestamp){: name='track_commit_timestamp'} (string, Enum: `off`, `on`). Record commit time of transactions. 
- [`track_functions`](#track_functions){: name='track_functions'} (string, Enum: `all`, `pl`, `none`). Enables tracking of function call counts and time used. 
- [`track_io_timing`](#track_io_timing){: name='track_io_timing'} (string, Enum: `off`, `on`). Enables timing of database I/O calls. This parameter is off by default, because it will repeatedly query the operating system for the current time, which may cause significant overhead on some platforms. 
- [`wal_sender_timeout`](#wal_sender_timeout){: name='wal_sender_timeout'} (integer). Terminate replication connections that are inactive for longer than this amount of time, in milliseconds. Setting this value to zero disables the timeout. 
- [`wal_writer_delay`](#wal_writer_delay){: name='wal_writer_delay'} (integer, Minimum: 10, Maximum: 200). WAL flush interval in milliseconds. Note that setting this value to lower than the default 200ms may negatively impact performance. 

### pgbouncer {: #pgbouncer }

PGBouncer connection pooling settings.

**Optional**

- [`autodb_idle_timeout`](#autodb_idle_timeout){: name='autodb_idle_timeout'} (integer, Minimum: 0, Maximum: 86400). If the automatically created database pools have been unused this many seconds, they are freed. If 0 then timeout is disabled. [seconds]. 
- [`autodb_max_db_connections`](#autodb_max_db_connections){: name='autodb_max_db_connections'} (integer, Minimum: 0, Maximum: 2147483647). Do not allow more than this many server connections per database (regardless of user). Setting it to 0 means unlimited. 
- [`autodb_pool_mode`](#autodb_pool_mode){: name='autodb_pool_mode'} (string, Enum: `session`, `transaction`, `statement`). PGBouncer pool mode. 
- [`autodb_pool_size`](#autodb_pool_size){: name='autodb_pool_size'} (integer, Minimum: 0, Maximum: 10000). If non-zero then create automatically a pool of that size per user when a pool doesn't exist. 
- [`ignore_startup_parameters`](#ignore_startup_parameters){: name='ignore_startup_parameters'} (array, MaxItems: 32). List of parameters to ignore when given in startup packet. 
- [`min_pool_size`](#min_pool_size){: name='min_pool_size'} (integer, Minimum: 0, Maximum: 10000). Add more server connections to pool if below this number. Improves behavior when usual load comes suddenly back after period of total inactivity. The value is effectively capped at the pool size. 
- [`server_idle_timeout`](#server_idle_timeout){: name='server_idle_timeout'} (integer, Minimum: 0, Maximum: 86400). If a server connection has been idle more than this many seconds it will be dropped. If 0 then timeout is disabled. [seconds]. 
- [`server_lifetime`](#server_lifetime){: name='server_lifetime'} (integer, Minimum: 60, Maximum: 86400). The pooler will close an unused server connection that has been connected longer than this. [seconds]. 
- [`server_reset_query_always`](#server_reset_query_always){: name='server_reset_query_always'} (boolean). Run server_reset_query (DISCARD ALL) in all pooling modes. 

### pglookout {: #pglookout }

PGLookout settings.

**Required**

- [`max_failover_replication_time_lag`](#max_failover_replication_time_lag){: name='max_failover_replication_time_lag'} (integer, Minimum: 10). Number of seconds of master unavailability before triggering database failover to standby. 

### private_access {: #private_access }

Allow access to selected service ports from private networks.

**Optional**

- [`pg`](#pg){: name='pg'} (boolean). Allow clients to connect to pg with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`pgbouncer`](#pgbouncer){: name='pgbouncer'} (boolean). Allow clients to connect to pgbouncer with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 

### privatelink_access {: #privatelink_access }

Allow access to selected service components through Privatelink.

**Optional**

- [`pg`](#pg){: name='pg'} (boolean). Enable pg. 
- [`pgbouncer`](#pgbouncer){: name='pgbouncer'} (boolean). Enable pgbouncer. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Enable prometheus. 

### public_access {: #public_access }

Allow access to selected service ports from the public Internet.

**Optional**

- [`pg`](#pg){: name='pg'} (boolean). Allow clients to connect to pg from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`pgbouncer`](#pgbouncer){: name='pgbouncer'} (boolean). Allow clients to connect to pgbouncer from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network. 

### timescaledb {: #timescaledb }

TimescaleDB extension configuration values.

**Required**

- [`max_background_workers`](#max_background_workers){: name='max_background_workers'} (integer, Minimum: 1, Maximum: 4096). The number of background workers for timescaledb operations. You should configure this setting to the sum of your number of databases and the total number of concurrent background workers you want running at any given point in time. 

