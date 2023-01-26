//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

// Code generated by controller-gen. DO NOT EDIT.

package pguserconfig

import ()

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IpFilter) DeepCopyInto(out *IpFilter) {
	*out = *in
	if in.Description != nil {
		in, out := &in.Description, &out.Description
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IpFilter.
func (in *IpFilter) DeepCopy() *IpFilter {
	if in == nil {
		return nil
	}
	out := new(IpFilter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Migration) DeepCopyInto(out *Migration) {
	*out = *in
	if in.Dbname != nil {
		in, out := &in.Dbname, &out.Dbname
		*out = new(string)
		**out = **in
	}
	if in.IgnoreDbs != nil {
		in, out := &in.IgnoreDbs, &out.IgnoreDbs
		*out = new(string)
		**out = **in
	}
	if in.Method != nil {
		in, out := &in.Method, &out.Method
		*out = new(string)
		**out = **in
	}
	if in.Password != nil {
		in, out := &in.Password, &out.Password
		*out = new(string)
		**out = **in
	}
	if in.Ssl != nil {
		in, out := &in.Ssl, &out.Ssl
		*out = new(bool)
		**out = **in
	}
	if in.Username != nil {
		in, out := &in.Username, &out.Username
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Migration.
func (in *Migration) DeepCopy() *Migration {
	if in == nil {
		return nil
	}
	out := new(Migration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Pg) DeepCopyInto(out *Pg) {
	*out = *in
	if in.AutovacuumAnalyzeScaleFactor != nil {
		in, out := &in.AutovacuumAnalyzeScaleFactor, &out.AutovacuumAnalyzeScaleFactor
		*out = new(int)
		**out = **in
	}
	if in.AutovacuumAnalyzeThreshold != nil {
		in, out := &in.AutovacuumAnalyzeThreshold, &out.AutovacuumAnalyzeThreshold
		*out = new(int)
		**out = **in
	}
	if in.AutovacuumFreezeMaxAge != nil {
		in, out := &in.AutovacuumFreezeMaxAge, &out.AutovacuumFreezeMaxAge
		*out = new(int)
		**out = **in
	}
	if in.AutovacuumMaxWorkers != nil {
		in, out := &in.AutovacuumMaxWorkers, &out.AutovacuumMaxWorkers
		*out = new(int)
		**out = **in
	}
	if in.AutovacuumNaptime != nil {
		in, out := &in.AutovacuumNaptime, &out.AutovacuumNaptime
		*out = new(int)
		**out = **in
	}
	if in.AutovacuumVacuumCostDelay != nil {
		in, out := &in.AutovacuumVacuumCostDelay, &out.AutovacuumVacuumCostDelay
		*out = new(int)
		**out = **in
	}
	if in.AutovacuumVacuumCostLimit != nil {
		in, out := &in.AutovacuumVacuumCostLimit, &out.AutovacuumVacuumCostLimit
		*out = new(int)
		**out = **in
	}
	if in.AutovacuumVacuumScaleFactor != nil {
		in, out := &in.AutovacuumVacuumScaleFactor, &out.AutovacuumVacuumScaleFactor
		*out = new(int)
		**out = **in
	}
	if in.AutovacuumVacuumThreshold != nil {
		in, out := &in.AutovacuumVacuumThreshold, &out.AutovacuumVacuumThreshold
		*out = new(int)
		**out = **in
	}
	if in.BgwriterDelay != nil {
		in, out := &in.BgwriterDelay, &out.BgwriterDelay
		*out = new(int)
		**out = **in
	}
	if in.BgwriterFlushAfter != nil {
		in, out := &in.BgwriterFlushAfter, &out.BgwriterFlushAfter
		*out = new(int)
		**out = **in
	}
	if in.BgwriterLruMaxpages != nil {
		in, out := &in.BgwriterLruMaxpages, &out.BgwriterLruMaxpages
		*out = new(int)
		**out = **in
	}
	if in.BgwriterLruMultiplier != nil {
		in, out := &in.BgwriterLruMultiplier, &out.BgwriterLruMultiplier
		*out = new(int)
		**out = **in
	}
	if in.DeadlockTimeout != nil {
		in, out := &in.DeadlockTimeout, &out.DeadlockTimeout
		*out = new(int)
		**out = **in
	}
	if in.DefaultToastCompression != nil {
		in, out := &in.DefaultToastCompression, &out.DefaultToastCompression
		*out = new(string)
		**out = **in
	}
	if in.IdleInTransactionSessionTimeout != nil {
		in, out := &in.IdleInTransactionSessionTimeout, &out.IdleInTransactionSessionTimeout
		*out = new(int)
		**out = **in
	}
	if in.Jit != nil {
		in, out := &in.Jit, &out.Jit
		*out = new(bool)
		**out = **in
	}
	if in.LogAutovacuumMinDuration != nil {
		in, out := &in.LogAutovacuumMinDuration, &out.LogAutovacuumMinDuration
		*out = new(int)
		**out = **in
	}
	if in.LogErrorVerbosity != nil {
		in, out := &in.LogErrorVerbosity, &out.LogErrorVerbosity
		*out = new(string)
		**out = **in
	}
	if in.LogLinePrefix != nil {
		in, out := &in.LogLinePrefix, &out.LogLinePrefix
		*out = new(string)
		**out = **in
	}
	if in.LogMinDurationStatement != nil {
		in, out := &in.LogMinDurationStatement, &out.LogMinDurationStatement
		*out = new(int)
		**out = **in
	}
	if in.LogTempFiles != nil {
		in, out := &in.LogTempFiles, &out.LogTempFiles
		*out = new(int)
		**out = **in
	}
	if in.MaxFilesPerProcess != nil {
		in, out := &in.MaxFilesPerProcess, &out.MaxFilesPerProcess
		*out = new(int)
		**out = **in
	}
	if in.MaxLocksPerTransaction != nil {
		in, out := &in.MaxLocksPerTransaction, &out.MaxLocksPerTransaction
		*out = new(int)
		**out = **in
	}
	if in.MaxLogicalReplicationWorkers != nil {
		in, out := &in.MaxLogicalReplicationWorkers, &out.MaxLogicalReplicationWorkers
		*out = new(int)
		**out = **in
	}
	if in.MaxParallelWorkers != nil {
		in, out := &in.MaxParallelWorkers, &out.MaxParallelWorkers
		*out = new(int)
		**out = **in
	}
	if in.MaxParallelWorkersPerGather != nil {
		in, out := &in.MaxParallelWorkersPerGather, &out.MaxParallelWorkersPerGather
		*out = new(int)
		**out = **in
	}
	if in.MaxPredLocksPerTransaction != nil {
		in, out := &in.MaxPredLocksPerTransaction, &out.MaxPredLocksPerTransaction
		*out = new(int)
		**out = **in
	}
	if in.MaxPreparedTransactions != nil {
		in, out := &in.MaxPreparedTransactions, &out.MaxPreparedTransactions
		*out = new(int)
		**out = **in
	}
	if in.MaxReplicationSlots != nil {
		in, out := &in.MaxReplicationSlots, &out.MaxReplicationSlots
		*out = new(int)
		**out = **in
	}
	if in.MaxSlotWalKeepSize != nil {
		in, out := &in.MaxSlotWalKeepSize, &out.MaxSlotWalKeepSize
		*out = new(int)
		**out = **in
	}
	if in.MaxStackDepth != nil {
		in, out := &in.MaxStackDepth, &out.MaxStackDepth
		*out = new(int)
		**out = **in
	}
	if in.MaxStandbyArchiveDelay != nil {
		in, out := &in.MaxStandbyArchiveDelay, &out.MaxStandbyArchiveDelay
		*out = new(int)
		**out = **in
	}
	if in.MaxStandbyStreamingDelay != nil {
		in, out := &in.MaxStandbyStreamingDelay, &out.MaxStandbyStreamingDelay
		*out = new(int)
		**out = **in
	}
	if in.MaxWalSenders != nil {
		in, out := &in.MaxWalSenders, &out.MaxWalSenders
		*out = new(int)
		**out = **in
	}
	if in.MaxWorkerProcesses != nil {
		in, out := &in.MaxWorkerProcesses, &out.MaxWorkerProcesses
		*out = new(int)
		**out = **in
	}
	if in.PgPartmanBgwInterval != nil {
		in, out := &in.PgPartmanBgwInterval, &out.PgPartmanBgwInterval
		*out = new(int)
		**out = **in
	}
	if in.PgPartmanBgwRole != nil {
		in, out := &in.PgPartmanBgwRole, &out.PgPartmanBgwRole
		*out = new(string)
		**out = **in
	}
	if in.PgStatMonitorPgsmEnableQueryPlan != nil {
		in, out := &in.PgStatMonitorPgsmEnableQueryPlan, &out.PgStatMonitorPgsmEnableQueryPlan
		*out = new(bool)
		**out = **in
	}
	if in.PgStatMonitorPgsmMaxBuckets != nil {
		in, out := &in.PgStatMonitorPgsmMaxBuckets, &out.PgStatMonitorPgsmMaxBuckets
		*out = new(int)
		**out = **in
	}
	if in.PgStatStatementsTrack != nil {
		in, out := &in.PgStatStatementsTrack, &out.PgStatStatementsTrack
		*out = new(string)
		**out = **in
	}
	if in.TempFileLimit != nil {
		in, out := &in.TempFileLimit, &out.TempFileLimit
		*out = new(int)
		**out = **in
	}
	if in.Timezone != nil {
		in, out := &in.Timezone, &out.Timezone
		*out = new(string)
		**out = **in
	}
	if in.TrackActivityQuerySize != nil {
		in, out := &in.TrackActivityQuerySize, &out.TrackActivityQuerySize
		*out = new(int)
		**out = **in
	}
	if in.TrackCommitTimestamp != nil {
		in, out := &in.TrackCommitTimestamp, &out.TrackCommitTimestamp
		*out = new(string)
		**out = **in
	}
	if in.TrackFunctions != nil {
		in, out := &in.TrackFunctions, &out.TrackFunctions
		*out = new(string)
		**out = **in
	}
	if in.TrackIoTiming != nil {
		in, out := &in.TrackIoTiming, &out.TrackIoTiming
		*out = new(string)
		**out = **in
	}
	if in.WalSenderTimeout != nil {
		in, out := &in.WalSenderTimeout, &out.WalSenderTimeout
		*out = new(int)
		**out = **in
	}
	if in.WalWriterDelay != nil {
		in, out := &in.WalWriterDelay, &out.WalWriterDelay
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Pg.
func (in *Pg) DeepCopy() *Pg {
	if in == nil {
		return nil
	}
	out := new(Pg)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PgUserConfig) DeepCopyInto(out *PgUserConfig) {
	*out = *in
	if in.AdditionalBackupRegions != nil {
		in, out := &in.AdditionalBackupRegions, &out.AdditionalBackupRegions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.AdminPassword != nil {
		in, out := &in.AdminPassword, &out.AdminPassword
		*out = new(string)
		**out = **in
	}
	if in.AdminUsername != nil {
		in, out := &in.AdminUsername, &out.AdminUsername
		*out = new(string)
		**out = **in
	}
	if in.BackupHour != nil {
		in, out := &in.BackupHour, &out.BackupHour
		*out = new(int)
		**out = **in
	}
	if in.BackupMinute != nil {
		in, out := &in.BackupMinute, &out.BackupMinute
		*out = new(int)
		**out = **in
	}
	if in.EnableIpv6 != nil {
		in, out := &in.EnableIpv6, &out.EnableIpv6
		*out = new(bool)
		**out = **in
	}
	if in.IpFilter != nil {
		in, out := &in.IpFilter, &out.IpFilter
		*out = make([]*IpFilter, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(IpFilter)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.Migration != nil {
		in, out := &in.Migration, &out.Migration
		*out = new(Migration)
		(*in).DeepCopyInto(*out)
	}
	if in.Pg != nil {
		in, out := &in.Pg, &out.Pg
		*out = new(Pg)
		(*in).DeepCopyInto(*out)
	}
	if in.PgReadReplica != nil {
		in, out := &in.PgReadReplica, &out.PgReadReplica
		*out = new(bool)
		**out = **in
	}
	if in.PgServiceToForkFrom != nil {
		in, out := &in.PgServiceToForkFrom, &out.PgServiceToForkFrom
		*out = new(string)
		**out = **in
	}
	if in.PgStatMonitorEnable != nil {
		in, out := &in.PgStatMonitorEnable, &out.PgStatMonitorEnable
		*out = new(bool)
		**out = **in
	}
	if in.PgVersion != nil {
		in, out := &in.PgVersion, &out.PgVersion
		*out = new(string)
		**out = **in
	}
	if in.Pgbouncer != nil {
		in, out := &in.Pgbouncer, &out.Pgbouncer
		*out = new(Pgbouncer)
		(*in).DeepCopyInto(*out)
	}
	if in.Pglookout != nil {
		in, out := &in.Pglookout, &out.Pglookout
		*out = new(Pglookout)
		(*in).DeepCopyInto(*out)
	}
	if in.PrivateAccess != nil {
		in, out := &in.PrivateAccess, &out.PrivateAccess
		*out = new(PrivateAccess)
		(*in).DeepCopyInto(*out)
	}
	if in.PrivatelinkAccess != nil {
		in, out := &in.PrivatelinkAccess, &out.PrivatelinkAccess
		*out = new(PrivatelinkAccess)
		(*in).DeepCopyInto(*out)
	}
	if in.ProjectToForkFrom != nil {
		in, out := &in.ProjectToForkFrom, &out.ProjectToForkFrom
		*out = new(string)
		**out = **in
	}
	if in.PublicAccess != nil {
		in, out := &in.PublicAccess, &out.PublicAccess
		*out = new(PublicAccess)
		(*in).DeepCopyInto(*out)
	}
	if in.RecoveryTargetTime != nil {
		in, out := &in.RecoveryTargetTime, &out.RecoveryTargetTime
		*out = new(string)
		**out = **in
	}
	if in.ServiceToForkFrom != nil {
		in, out := &in.ServiceToForkFrom, &out.ServiceToForkFrom
		*out = new(string)
		**out = **in
	}
	if in.SharedBuffersPercentage != nil {
		in, out := &in.SharedBuffersPercentage, &out.SharedBuffersPercentage
		*out = new(int)
		**out = **in
	}
	if in.StaticIps != nil {
		in, out := &in.StaticIps, &out.StaticIps
		*out = new(bool)
		**out = **in
	}
	if in.SynchronousReplication != nil {
		in, out := &in.SynchronousReplication, &out.SynchronousReplication
		*out = new(string)
		**out = **in
	}
	if in.Timescaledb != nil {
		in, out := &in.Timescaledb, &out.Timescaledb
		*out = new(Timescaledb)
		(*in).DeepCopyInto(*out)
	}
	if in.Variant != nil {
		in, out := &in.Variant, &out.Variant
		*out = new(string)
		**out = **in
	}
	if in.WorkMem != nil {
		in, out := &in.WorkMem, &out.WorkMem
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PgUserConfig.
func (in *PgUserConfig) DeepCopy() *PgUserConfig {
	if in == nil {
		return nil
	}
	out := new(PgUserConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Pgbouncer) DeepCopyInto(out *Pgbouncer) {
	*out = *in
	if in.AutodbIdleTimeout != nil {
		in, out := &in.AutodbIdleTimeout, &out.AutodbIdleTimeout
		*out = new(int)
		**out = **in
	}
	if in.AutodbMaxDbConnections != nil {
		in, out := &in.AutodbMaxDbConnections, &out.AutodbMaxDbConnections
		*out = new(int)
		**out = **in
	}
	if in.AutodbPoolMode != nil {
		in, out := &in.AutodbPoolMode, &out.AutodbPoolMode
		*out = new(string)
		**out = **in
	}
	if in.AutodbPoolSize != nil {
		in, out := &in.AutodbPoolSize, &out.AutodbPoolSize
		*out = new(int)
		**out = **in
	}
	if in.IgnoreStartupParameters != nil {
		in, out := &in.IgnoreStartupParameters, &out.IgnoreStartupParameters
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.MinPoolSize != nil {
		in, out := &in.MinPoolSize, &out.MinPoolSize
		*out = new(int)
		**out = **in
	}
	if in.ServerIdleTimeout != nil {
		in, out := &in.ServerIdleTimeout, &out.ServerIdleTimeout
		*out = new(int)
		**out = **in
	}
	if in.ServerLifetime != nil {
		in, out := &in.ServerLifetime, &out.ServerLifetime
		*out = new(int)
		**out = **in
	}
	if in.ServerResetQueryAlways != nil {
		in, out := &in.ServerResetQueryAlways, &out.ServerResetQueryAlways
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Pgbouncer.
func (in *Pgbouncer) DeepCopy() *Pgbouncer {
	if in == nil {
		return nil
	}
	out := new(Pgbouncer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Pglookout) DeepCopyInto(out *Pglookout) {
	*out = *in
	if in.MaxFailoverReplicationTimeLag != nil {
		in, out := &in.MaxFailoverReplicationTimeLag, &out.MaxFailoverReplicationTimeLag
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Pglookout.
func (in *Pglookout) DeepCopy() *Pglookout {
	if in == nil {
		return nil
	}
	out := new(Pglookout)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrivateAccess) DeepCopyInto(out *PrivateAccess) {
	*out = *in
	if in.Pg != nil {
		in, out := &in.Pg, &out.Pg
		*out = new(bool)
		**out = **in
	}
	if in.Pgbouncer != nil {
		in, out := &in.Pgbouncer, &out.Pgbouncer
		*out = new(bool)
		**out = **in
	}
	if in.Prometheus != nil {
		in, out := &in.Prometheus, &out.Prometheus
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrivateAccess.
func (in *PrivateAccess) DeepCopy() *PrivateAccess {
	if in == nil {
		return nil
	}
	out := new(PrivateAccess)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrivatelinkAccess) DeepCopyInto(out *PrivatelinkAccess) {
	*out = *in
	if in.Pg != nil {
		in, out := &in.Pg, &out.Pg
		*out = new(bool)
		**out = **in
	}
	if in.Pgbouncer != nil {
		in, out := &in.Pgbouncer, &out.Pgbouncer
		*out = new(bool)
		**out = **in
	}
	if in.Prometheus != nil {
		in, out := &in.Prometheus, &out.Prometheus
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PrivatelinkAccess.
func (in *PrivatelinkAccess) DeepCopy() *PrivatelinkAccess {
	if in == nil {
		return nil
	}
	out := new(PrivatelinkAccess)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PublicAccess) DeepCopyInto(out *PublicAccess) {
	*out = *in
	if in.Pg != nil {
		in, out := &in.Pg, &out.Pg
		*out = new(bool)
		**out = **in
	}
	if in.Pgbouncer != nil {
		in, out := &in.Pgbouncer, &out.Pgbouncer
		*out = new(bool)
		**out = **in
	}
	if in.Prometheus != nil {
		in, out := &in.Prometheus, &out.Prometheus
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PublicAccess.
func (in *PublicAccess) DeepCopy() *PublicAccess {
	if in == nil {
		return nil
	}
	out := new(PublicAccess)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Timescaledb) DeepCopyInto(out *Timescaledb) {
	*out = *in
	if in.MaxBackgroundWorkers != nil {
		in, out := &in.MaxBackgroundWorkers, &out.MaxBackgroundWorkers
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Timescaledb.
func (in *Timescaledb) DeepCopy() *Timescaledb {
	if in == nil {
		return nil
	}
	out := new(Timescaledb)
	in.DeepCopyInto(out)
	return out
}
