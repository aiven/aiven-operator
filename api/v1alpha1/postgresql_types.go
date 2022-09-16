// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgreSQLSpec defines the desired state of postgres instance
type PostgreSQLSpec struct {
	ServiceCommonSpec `json:",inline"`

	// +kubebuilder:validation:Format="^[1-9][0-9]*(GiB|G)*"
	// The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
	DiskSpace string `json:"disk_space,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef AuthSecretReference `json:"authSecretRef,omitempty"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// PostgreSQL specific user configuration options
	UserConfig PostgreSQLUserconfig `json:"userConfig,omitempty"`
}

type PostgreSQLUserconfig struct {
	// PostgreSQL major version
	PgVersion string `json:"pg_version,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=59
	// The minute of an hour when backup for the service is started. New backup is only started if previous backup has already completed.
	BackupMinute *int64 `json:"backup_minute,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Name of the PostgreSQL Service from which to fork (deprecated, use service_to_fork_from). This has effect only when a new service is being created.
	PgServiceToForkFrom string `json:"pg_service_to_fork_from,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=23
	// The hour of day (in UTC) when backup for the service is started. New backup is only started if previous backup has already completed.
	BackupHour *int64 `json:"backup_hour,omitempty"`

	// PGLookout settings
	Pglookout PgLookoutUserConfig `json:"pglookout,omitempty"`

	// +kubebuilder:validation:Minimum=20
	// +kubebuilder:validation:Maximum=60
	// shared_buffers_percentage Percentage of total RAM that the database server uses for shared memory buffers. Valid range is 20-60 (float), which corresponds to 20% - 60%. This setting adjusts the shared_buffers configuration value. The absolute maximum is 12 GB.
	SharedBuffersPercentage *int64 `json:"shared_buffers_percentage,omitempty"`

	// +kubebuilder:validation:Enum=quorum;off
	// Synchronous replication type. Note that the service plan also needs to support synchronous replication.
	SynchronousReplication string `json:"synchronous_replication,omitempty"`

	// TimescaleDB extension configuration values
	Timescaledb TimescaledbUserConfig `json:"timescaledb,omitempty"`

	// +kubebuilder:validation:Format="^[a-zA-Z0-9-_]+$"
	// +kubebuilder:validation:MaxLength=256
	// Custom password for admin user. Defaults to random string. This must be set only when a new service is being created.
	AdminPassword string `json:"admin_password,omitempty"`

	// IP filter Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'
	IPFilter []string `json:"ip_filter,omitempty"`

	// PGBouncer connection pooling settings
	Pgbouncer PgbouncerUserConfig `json:"pgbouncer,omitempty"`

	// +kubebuilder:validation:MaxLength=32
	// Recovery target time when forking a service. This has effect only when a new service is being created.
	RecoveryTargetTime string `json:"recovery_target_time,omitempty"`

	// +kubebuilder:validation:Format="^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$"
	// +kubebuilder:validation:MaxLength=64
	// Custom username for admin user. This must be set only when a new service is being created.
	AdminUsername string `json:"admin_username,omitempty"`

	// Migrate data from existing server
	Migration MigrationUserConfig `json:"migration,omitempty"`

	// Allow access to selected service ports from private networks
	PrivateAccess PrivateAccessUserConfig `json:"private_access,omitempty"`

	// Allow access to selected service ports from the public Internet
	PublicAccess PublicAccessUserConfig `json:"public_access,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Name of another service to fork from. This has effect only when a new service is being created.
	ServiceToForkFrom string `json:"service_to_fork_from,omitempty"`

	// +kubebuilder:validation:Enum=aiven;timescale
	// Variant of the PostgreSQL service, may affect the features that are exposed by default
	Variant string `json:"variant,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1024
	// work_mem Sets the maximum amount of memory to be used by a query operation (such as a sort or hash table) before writing to temporary disk files, in MB. Default is 1MB + 0.075% of total RAM (up to 32MB).
	WorkMem *int64 `json:"work_mem,omitempty"`

	// postgresql.conf configuration values
	Pg PostgreSQLSubUserConfig `json:"pg,omitempty"`
}

type PostgreSQLSubUserConfig struct {
	// +kubebuilder:validation:Maximum=86400000
	// log_min_duration_statement Log statements that take more than this number of milliseconds to run, -1 disables
	LogMinDurationStatement *int64 `json:"log_min_duration_statement,omitempty"`

	// +kubebuilder:validation:Minimum=8
	// +kubebuilder:validation:Maximum=64
	// max_replication_slots PostgreSQL maximum replication slots
	MaxReplicationSlots *int64 `json:"max_replication_slots,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=43200000
	// max_standby_streaming_delay Max standby streaming delay in milliseconds
	MaxStandbyStreamingDelay *int64 `json:"max_standby_streaming_delay,omitempty"`

	// +kubebuilder:validation:Minimum=3600
	// +kubebuilder:validation:Maximum=604800
	// pg_partman_bgw.interval Sets the time interval to run pg_partman's scheduled tasks
	PgPartmanBgwInterval *int64 `json:"pg_partman_bgw.interval,omitempty"`

	// +kubebuilder:validation:Enum=all;top;none
	// pg_stat_statements.track Controls which statements are counted. Specify top to track top-level statements (those issued directly by clients), all to also track nested statements (such as statements invoked within functions), or none to disable statement statistics collection. The default value is top.
	PgStatStatementsTrack string `json:"pg_stat_statements.track,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2147483647
	// autovacuum_vacuum_threshold Specifies the minimum number of updated or deleted tuples needed to trigger a VACUUM in any one table. The default is 50 tuples
	AutovacuumVacuumThreshold *int64 `json:"autovacuum_vacuum_threshold,omitempty"`

	// jit Controls system-wide use of Just-in-Time Compilation (JIT).
	Jit *bool `json:"jit,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=10000
	// max_prepared_transactions PostgreSQL maximum prepared transactions
	MaxPreparedTransactions *int64 `json:"max_prepared_transactions,omitempty"`

	// +kubebuilder:validation:Minimum=200000000
	// +kubebuilder:validation:Maximum=1500000000
	// autovacuum_freeze_max_age Specifies the maximum age (in transactions) that a table's pg_class.relfrozenxid field can attain before a VACUUM operation is forced to prevent transaction ID wraparound within the table. Note that the system will launch autovacuum processes to prevent wraparound even when autovacuum is otherwise disabled. This parameter will cause the server to be restarted.
	AutovacuumFreezeMaxAge *int64 `json:"autovacuum_freeze_max_age,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=604800000
	// idle_in_transaction_session_timeout Time out sessions with open transactions after this number of milliseconds
	IdleInTransactionSessionTimeout *int64 `json:"idle_in_transaction_session_timeout,omitempty"`

	// +kubebuilder:validation:Minimum=5000
	// +kubebuilder:validation:Maximum=600000
	// wal_sender_timeout Terminate replication connections that are inactive for longer than this amount of time, in milliseconds.
	WalSenderTimeout *int64 `json:"wal_sender_timeout,omitempty"`

	// +kubebuilder:validation:Minimum=64
	// +kubebuilder:validation:Maximum=640
	// max_pred_locks_per_transaction PostgreSQL maximum predicate locks per transaction
	MaxPredLocksPerTransaction *int64 `json:"max_pred_locks_per_transaction,omitempty"`

	// +kubebuilder:validation:MaxLength=64
	// timezone PostgreSQL service timezone
	Timezone string `json:"timezone,omitempty"`

	// +kubebuilder:validation:Minimum=8
	// +kubebuilder:validation:Maximum=64
	// max_wal_senders PostgreSQL maximum WAL senders
	MaxWalSenders *int64 `json:"max_wal_senders,omitempty"`

	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=10240
	// track_activity_query_size Specifies the number of bytes reserved to track the currently executing command for each active session.
	TrackActivityQuerySize *int64 `json:"track_activity_query_size,omitempty"`

	// +kubebuilder:validation:Minimum=1000
	// +kubebuilder:validation:Maximum=4096
	// max_files_per_process PostgreSQL maximum number of files that can be open per process
	MaxFilesPerProcess *int64 `json:"max_files_per_process,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=96
	// max_parallel_workers_per_gather Sets the maximum number of workers that can be started by a single Gather or Gather Merge node
	MaxParallelWorkersPerGather *int64 `json:"max_parallel_workers_per_gather,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1
	// autovacuum_vacuum_scale_factor Specifies a fraction of the table size to add to autovacuum_vacuum_threshold when deciding whether to trigger a VACUUM. The default is 0.2 (20% of table size)
	AutovacuumVacuumScaleFactor *int64 `json:"autovacuum_vacuum_scale_factor,omitempty"`

	// +kubebuilder:validation:Maximum=2147483647
	// log_autovacuum_min_duration Causes each action executed by autovacuum to be logged if it ran for at least the specified number of milliseconds. Setting this to zero logs all autovacuum actions. Minus-one (the default) disables logging autovacuum actions.
	LogAutovacuumMinDuration *int64 `json:"log_autovacuum_min_duration,omitempty"`

	// +kubebuilder:validation:Minimum=64
	// +kubebuilder:validation:Maximum=640
	// max_locks_per_transaction PostgreSQL maximum locks per transaction
	MaxLocksPerTransaction *int64 `json:"max_locks_per_transaction,omitempty"`

	// +kubebuilder:validation:Minimum=2097152
	// +kubebuilder:validation:Maximum=6291456
	// max_stack_depth Maximum depth of the stack in bytes
	MaxStackDepth *int64 `json:"max_stack_depth,omitempty"`

	// +kubebuilder:validation:Minimum=8
	// +kubebuilder:validation:Maximum=96
	// max_worker_processes Sets the maximum number of background processes that the system can support
	MaxWorkerProcesses *int64 `json:"max_worker_processes,omitempty"`

	// +kubebuilder:validation:Format="^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$"
	// +kubebuilder:validation:MaxLength=64
	// pg_partman_bgw.role Controls which role to use for pg_partman's scheduled background tasks.
	PgPartmanBgwRole string `json:"pg_partman_bgw.role,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1
	// autovacuum_analyze_scale_factor Specifies a fraction of the table size to add to autovacuum_analyze_threshold when deciding whether to trigger an ANALYZE. The default is 0.2 (20% of table size)
	AutovacuumAnalyzeScaleFactor *int64 `json:"autovacuum_analyze_scale_factor,omitempty"`

	// +kubebuilder:validation:Maximum=10000
	// autovacuum_vacuum_cost_limit Specifies the cost limit value that will be used in automatic VACUUM operations. If -1 is specified (which is the default), the regular vacuum_cost_limit value will be used.
	AutovacuumVacuumCostLimit *int64 `json:"autovacuum_vacuum_cost_limit,omitempty"`

	// +kubebuilder:validation:Maximum=2147483647
	// temp_file_limit PostgreSQL temporary file limit in KiB, -1 for unlimited
	TempFileLimit *int64 `json:"temp_file_limit,omitempty"`

	// +kubebuilder:validation:Enum=all;pl;none
	// track_functions Enables tracking of function call counts and time used.
	TrackFunctions string `json:"track_functions,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=96
	// max_parallel_workers Sets the maximum number of workers that the system can support for parallel queries
	MaxParallelWorkers *int64 `json:"max_parallel_workers,omitempty"`

	// +kubebuilder:validation:Enum=off;on
	// track_commit_timestamp Record commit time of transactions.
	TrackCommitTimestamp string `json:"track_commit_timestamp,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=43200000
	// max_standby_archive_delay Max standby archive delay in milliseconds
	MaxStandbyArchiveDelay *int64 `json:"max_standby_archive_delay,omitempty"`

	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=200
	// wal_writer_delay WAL flush interval in milliseconds. Note that setting this value to lower than the default 200ms may negatively impact performance
	WalWriterDelay *int64 `json:"wal_writer_delay,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2147483647
	// autovacuum_analyze_threshold Specifies the minimum number of inserted, updated or deleted tuples needed to trigger an  ANALYZE in any one table. The default is 50 tuples.
	AutovacuumAnalyzeThreshold *int64 `json:"autovacuum_analyze_threshold,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=86400
	// autovacuum_naptime Specifies the minimum delay between autovacuum runs on any given database. The delay is measured in seconds, and the default is one minute
	AutovacuumNaptime *int64 `json:"autovacuum_naptime,omitempty"`

	// +kubebuilder:validation:Minimum=500
	// +kubebuilder:validation:Maximum=1800000
	// deadlock_timeout This is the amount of time, in milliseconds, to wait on a lock before checking to see if there is a deadlock condition.
	DeadlockTimeout *int64 `json:"deadlock_timeout,omitempty"`

	// +kubebuilder:validation:Enum=TERSE;DEFAULT;VERBOSE
	// log_error_verbosity Controls the amount of detail written in the server log for each message that is logged.
	LogErrorVerbosity string `json:"log_error_verbosity,omitempty"`

	// +kubebuilder:validation:Minimum=4
	// +kubebuilder:validation:Maximum=64
	// max_logical_replication_workers PostgreSQL maximum logical replication workers (taken from the pool of max_parallel_workers)
	MaxLogicalReplicationWorkers *int64 `json:"max_logical_replication_workers,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=20
	// autovacuum_max_workers Specifies the maximum number of autovacuum processes (other than the autovacuum launcher) that may be running at any one time. The default is three. This parameter can only be set at server start.
	AutovacuumMaxWorkers *int64 `json:"autovacuum_max_workers,omitempty"`

	// +kubebuilder:validation:Maximum=100
	// autovacuum_vacuum_cost_delay Specifies the cost delay value that will be used in automatic VACUUM operations. If -1 is specified, the regular vacuum_cost_delay value will be used. The default value is 20 milliseconds
	AutovacuumVacuumCostDelay *int64 `json:"autovacuum_vacuum_cost_delay,omitempty"`
}

type PgbouncerUserConfig struct {
	// List of parameters to ignore when given in startup packet
	IgnoreStartupParameters []string `json:"ignore_startup_parameters,omitempty"`

	// Run server_reset_query (DISCARD ALL) in all pooling modes
	ServerResetQueryAlways *bool `json:"server_reset_query_always,omitempty"`
}

type PgLookoutUserConfig struct {
	// +kubebuilder:validation:Minimum=10
	// max_failover_replication_time_lag Number of seconds of master unavailability before triggering database failover to standby
	MaxFailoverReplicationTimeLag *int64 `json:"max_failover_replication_time_lag,omitempty"`
}

type TimescaledbUserConfig struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4096
	// timescaledb.max_background_workers The number of background workers for timescaledb operations. You should configure this setting to the sum of your number of databases and the total number of concurrent background workers you want running at any given point in time.
	MaxBackgroundWorkers *int64 `json:"max_background_workers,omitempty"`
}

type MigrationUserConfig struct {
	// +kubebuilder:validation:MaxLength=255
	// Hostname or IP address of the server where to migrate data from
	Host string `json:"host,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Password for authentication with the server where to migrate data from
	Password string `json:"password,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// Port number of the server where to migrate data from
	Port *int64 `json:"port,omitempty"`

	// The server where to migrate data from is secured with SSL
	Ssl *bool `json:"ssl,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// User name for authentication with the server where to migrate data from
	Username string `json:"username,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Database name for bootstrapping the initial connection
	Dbname string `json:"dbname,omitempty"`
}

type PrivateAccessUserConfig struct {
	// Allow clients to connect to pg with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Pg *bool `json:"pg,omitempty"`

	// Allow clients to connect to pgbouncer with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Pgbouncer *bool `json:"pgbouncer,omitempty"`

	// Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Prometheus *bool `json:"prometheus,omitempty"`
}

type PublicAccessUserConfig struct {
	// Allow clients to connect to pg from the public internet for service nodes that are in a project VPC or another type of private network
	Pg *bool `json:"pg,omitempty"`

	// Allow clients to connect to pgbouncer from the public internet for service nodes that are in a project VPC or another type of private network
	Pgbouncer *bool `json:"pgbouncer,omitempty"`

	// Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network
	Prometheus *bool `json:"prometheus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PostgreSQL is the Schema for the postgresql API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type PostgreSQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLSpec `json:"spec,omitempty"`
	Status ServiceStatus  `json:"status,omitempty"`
}

func (pg PostgreSQL) AuthSecretRef() AuthSecretReference {
	return pg.Spec.AuthSecretRef
}

// +kubebuilder:object:root=true

// PostgreSQLList contains a list of PostgreSQL instances
type PostgreSQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
}
