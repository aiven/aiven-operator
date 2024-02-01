//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

// Code generated by controller-gen. DO NOT EDIT.

package opensearchuserconfig

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuthFailureListeners) DeepCopyInto(out *AuthFailureListeners) {
	*out = *in
	if in.InternalAuthenticationBackendLimiting != nil {
		in, out := &in.InternalAuthenticationBackendLimiting, &out.InternalAuthenticationBackendLimiting
		*out = new(InternalAuthenticationBackendLimiting)
		(*in).DeepCopyInto(*out)
	}
	if in.IpRateLimiting != nil {
		in, out := &in.IpRateLimiting, &out.IpRateLimiting
		*out = new(IpRateLimiting)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuthFailureListeners.
func (in *AuthFailureListeners) DeepCopy() *AuthFailureListeners {
	if in == nil {
		return nil
	}
	out := new(AuthFailureListeners)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IndexPatterns) DeepCopyInto(out *IndexPatterns) {
	*out = *in
	if in.SortingAlgorithm != nil {
		in, out := &in.SortingAlgorithm, &out.SortingAlgorithm
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IndexPatterns.
func (in *IndexPatterns) DeepCopy() *IndexPatterns {
	if in == nil {
		return nil
	}
	out := new(IndexPatterns)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IndexTemplate) DeepCopyInto(out *IndexTemplate) {
	*out = *in
	if in.MappingNestedObjectsLimit != nil {
		in, out := &in.MappingNestedObjectsLimit, &out.MappingNestedObjectsLimit
		*out = new(int)
		**out = **in
	}
	if in.NumberOfReplicas != nil {
		in, out := &in.NumberOfReplicas, &out.NumberOfReplicas
		*out = new(int)
		**out = **in
	}
	if in.NumberOfShards != nil {
		in, out := &in.NumberOfShards, &out.NumberOfShards
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IndexTemplate.
func (in *IndexTemplate) DeepCopy() *IndexTemplate {
	if in == nil {
		return nil
	}
	out := new(IndexTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InternalAuthenticationBackendLimiting) DeepCopyInto(out *InternalAuthenticationBackendLimiting) {
	*out = *in
	if in.AllowedTries != nil {
		in, out := &in.AllowedTries, &out.AllowedTries
		*out = new(int)
		**out = **in
	}
	if in.AuthenticationBackend != nil {
		in, out := &in.AuthenticationBackend, &out.AuthenticationBackend
		*out = new(string)
		**out = **in
	}
	if in.BlockExpirySeconds != nil {
		in, out := &in.BlockExpirySeconds, &out.BlockExpirySeconds
		*out = new(int)
		**out = **in
	}
	if in.MaxBlockedClients != nil {
		in, out := &in.MaxBlockedClients, &out.MaxBlockedClients
		*out = new(int)
		**out = **in
	}
	if in.MaxTrackedClients != nil {
		in, out := &in.MaxTrackedClients, &out.MaxTrackedClients
		*out = new(int)
		**out = **in
	}
	if in.TimeWindowSeconds != nil {
		in, out := &in.TimeWindowSeconds, &out.TimeWindowSeconds
		*out = new(int)
		**out = **in
	}
	if in.Type != nil {
		in, out := &in.Type, &out.Type
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InternalAuthenticationBackendLimiting.
func (in *InternalAuthenticationBackendLimiting) DeepCopy() *InternalAuthenticationBackendLimiting {
	if in == nil {
		return nil
	}
	out := new(InternalAuthenticationBackendLimiting)
	in.DeepCopyInto(out)
	return out
}

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
func (in *IpRateLimiting) DeepCopyInto(out *IpRateLimiting) {
	*out = *in
	if in.AllowedTries != nil {
		in, out := &in.AllowedTries, &out.AllowedTries
		*out = new(int)
		**out = **in
	}
	if in.BlockExpirySeconds != nil {
		in, out := &in.BlockExpirySeconds, &out.BlockExpirySeconds
		*out = new(int)
		**out = **in
	}
	if in.MaxBlockedClients != nil {
		in, out := &in.MaxBlockedClients, &out.MaxBlockedClients
		*out = new(int)
		**out = **in
	}
	if in.MaxTrackedClients != nil {
		in, out := &in.MaxTrackedClients, &out.MaxTrackedClients
		*out = new(int)
		**out = **in
	}
	if in.TimeWindowSeconds != nil {
		in, out := &in.TimeWindowSeconds, &out.TimeWindowSeconds
		*out = new(int)
		**out = **in
	}
	if in.Type != nil {
		in, out := &in.Type, &out.Type
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IpRateLimiting.
func (in *IpRateLimiting) DeepCopy() *IpRateLimiting {
	if in == nil {
		return nil
	}
	out := new(IpRateLimiting)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Openid) DeepCopyInto(out *Openid) {
	*out = *in
	if in.Header != nil {
		in, out := &in.Header, &out.Header
		*out = new(string)
		**out = **in
	}
	if in.JwtHeader != nil {
		in, out := &in.JwtHeader, &out.JwtHeader
		*out = new(string)
		**out = **in
	}
	if in.JwtUrlParameter != nil {
		in, out := &in.JwtUrlParameter, &out.JwtUrlParameter
		*out = new(string)
		**out = **in
	}
	if in.RefreshRateLimitCount != nil {
		in, out := &in.RefreshRateLimitCount, &out.RefreshRateLimitCount
		*out = new(int)
		**out = **in
	}
	if in.RefreshRateLimitTimeWindowMs != nil {
		in, out := &in.RefreshRateLimitTimeWindowMs, &out.RefreshRateLimitTimeWindowMs
		*out = new(int)
		**out = **in
	}
	if in.RolesKey != nil {
		in, out := &in.RolesKey, &out.RolesKey
		*out = new(string)
		**out = **in
	}
	if in.Scope != nil {
		in, out := &in.Scope, &out.Scope
		*out = new(string)
		**out = **in
	}
	if in.SubjectKey != nil {
		in, out := &in.SubjectKey, &out.SubjectKey
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Openid.
func (in *Openid) DeepCopy() *Openid {
	if in == nil {
		return nil
	}
	out := new(Openid)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Opensearch) DeepCopyInto(out *Opensearch) {
	*out = *in
	if in.ActionAutoCreateIndexEnabled != nil {
		in, out := &in.ActionAutoCreateIndexEnabled, &out.ActionAutoCreateIndexEnabled
		*out = new(bool)
		**out = **in
	}
	if in.ActionDestructiveRequiresName != nil {
		in, out := &in.ActionDestructiveRequiresName, &out.ActionDestructiveRequiresName
		*out = new(bool)
		**out = **in
	}
	if in.AuthFailureListeners != nil {
		in, out := &in.AuthFailureListeners, &out.AuthFailureListeners
		*out = new(AuthFailureListeners)
		(*in).DeepCopyInto(*out)
	}
	if in.ClusterMaxShardsPerNode != nil {
		in, out := &in.ClusterMaxShardsPerNode, &out.ClusterMaxShardsPerNode
		*out = new(int)
		**out = **in
	}
	if in.ClusterRoutingAllocationNodeConcurrentRecoveries != nil {
		in, out := &in.ClusterRoutingAllocationNodeConcurrentRecoveries, &out.ClusterRoutingAllocationNodeConcurrentRecoveries
		*out = new(int)
		**out = **in
	}
	if in.EmailSenderName != nil {
		in, out := &in.EmailSenderName, &out.EmailSenderName
		*out = new(string)
		**out = **in
	}
	if in.EmailSenderPassword != nil {
		in, out := &in.EmailSenderPassword, &out.EmailSenderPassword
		*out = new(string)
		**out = **in
	}
	if in.EmailSenderUsername != nil {
		in, out := &in.EmailSenderUsername, &out.EmailSenderUsername
		*out = new(string)
		**out = **in
	}
	if in.EnableSecurityAudit != nil {
		in, out := &in.EnableSecurityAudit, &out.EnableSecurityAudit
		*out = new(bool)
		**out = **in
	}
	if in.HttpMaxContentLength != nil {
		in, out := &in.HttpMaxContentLength, &out.HttpMaxContentLength
		*out = new(int)
		**out = **in
	}
	if in.HttpMaxHeaderSize != nil {
		in, out := &in.HttpMaxHeaderSize, &out.HttpMaxHeaderSize
		*out = new(int)
		**out = **in
	}
	if in.HttpMaxInitialLineLength != nil {
		in, out := &in.HttpMaxInitialLineLength, &out.HttpMaxInitialLineLength
		*out = new(int)
		**out = **in
	}
	if in.IndicesFielddataCacheSize != nil {
		in, out := &in.IndicesFielddataCacheSize, &out.IndicesFielddataCacheSize
		*out = new(int)
		**out = **in
	}
	if in.IndicesMemoryIndexBufferSize != nil {
		in, out := &in.IndicesMemoryIndexBufferSize, &out.IndicesMemoryIndexBufferSize
		*out = new(int)
		**out = **in
	}
	if in.IndicesMemoryMaxIndexBufferSize != nil {
		in, out := &in.IndicesMemoryMaxIndexBufferSize, &out.IndicesMemoryMaxIndexBufferSize
		*out = new(int)
		**out = **in
	}
	if in.IndicesMemoryMinIndexBufferSize != nil {
		in, out := &in.IndicesMemoryMinIndexBufferSize, &out.IndicesMemoryMinIndexBufferSize
		*out = new(int)
		**out = **in
	}
	if in.IndicesQueriesCacheSize != nil {
		in, out := &in.IndicesQueriesCacheSize, &out.IndicesQueriesCacheSize
		*out = new(int)
		**out = **in
	}
	if in.IndicesQueryBoolMaxClauseCount != nil {
		in, out := &in.IndicesQueryBoolMaxClauseCount, &out.IndicesQueryBoolMaxClauseCount
		*out = new(int)
		**out = **in
	}
	if in.IndicesRecoveryMaxBytesPerSec != nil {
		in, out := &in.IndicesRecoveryMaxBytesPerSec, &out.IndicesRecoveryMaxBytesPerSec
		*out = new(int)
		**out = **in
	}
	if in.IndicesRecoveryMaxConcurrentFileChunks != nil {
		in, out := &in.IndicesRecoveryMaxConcurrentFileChunks, &out.IndicesRecoveryMaxConcurrentFileChunks
		*out = new(int)
		**out = **in
	}
	if in.IsmEnabled != nil {
		in, out := &in.IsmEnabled, &out.IsmEnabled
		*out = new(bool)
		**out = **in
	}
	if in.IsmHistoryEnabled != nil {
		in, out := &in.IsmHistoryEnabled, &out.IsmHistoryEnabled
		*out = new(bool)
		**out = **in
	}
	if in.IsmHistoryMaxAge != nil {
		in, out := &in.IsmHistoryMaxAge, &out.IsmHistoryMaxAge
		*out = new(int)
		**out = **in
	}
	if in.IsmHistoryMaxDocs != nil {
		in, out := &in.IsmHistoryMaxDocs, &out.IsmHistoryMaxDocs
		*out = new(int)
		**out = **in
	}
	if in.IsmHistoryRolloverCheckPeriod != nil {
		in, out := &in.IsmHistoryRolloverCheckPeriod, &out.IsmHistoryRolloverCheckPeriod
		*out = new(int)
		**out = **in
	}
	if in.IsmHistoryRolloverRetentionPeriod != nil {
		in, out := &in.IsmHistoryRolloverRetentionPeriod, &out.IsmHistoryRolloverRetentionPeriod
		*out = new(int)
		**out = **in
	}
	if in.OverrideMainResponseVersion != nil {
		in, out := &in.OverrideMainResponseVersion, &out.OverrideMainResponseVersion
		*out = new(bool)
		**out = **in
	}
	if in.ReindexRemoteWhitelist != nil {
		in, out := &in.ReindexRemoteWhitelist, &out.ReindexRemoteWhitelist
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ScriptMaxCompilationsRate != nil {
		in, out := &in.ScriptMaxCompilationsRate, &out.ScriptMaxCompilationsRate
		*out = new(string)
		**out = **in
	}
	if in.SearchMaxBuckets != nil {
		in, out := &in.SearchMaxBuckets, &out.SearchMaxBuckets
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolAnalyzeQueueSize != nil {
		in, out := &in.ThreadPoolAnalyzeQueueSize, &out.ThreadPoolAnalyzeQueueSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolAnalyzeSize != nil {
		in, out := &in.ThreadPoolAnalyzeSize, &out.ThreadPoolAnalyzeSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolForceMergeSize != nil {
		in, out := &in.ThreadPoolForceMergeSize, &out.ThreadPoolForceMergeSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolGetQueueSize != nil {
		in, out := &in.ThreadPoolGetQueueSize, &out.ThreadPoolGetQueueSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolGetSize != nil {
		in, out := &in.ThreadPoolGetSize, &out.ThreadPoolGetSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolSearchQueueSize != nil {
		in, out := &in.ThreadPoolSearchQueueSize, &out.ThreadPoolSearchQueueSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolSearchSize != nil {
		in, out := &in.ThreadPoolSearchSize, &out.ThreadPoolSearchSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolSearchThrottledQueueSize != nil {
		in, out := &in.ThreadPoolSearchThrottledQueueSize, &out.ThreadPoolSearchThrottledQueueSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolSearchThrottledSize != nil {
		in, out := &in.ThreadPoolSearchThrottledSize, &out.ThreadPoolSearchThrottledSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolWriteQueueSize != nil {
		in, out := &in.ThreadPoolWriteQueueSize, &out.ThreadPoolWriteQueueSize
		*out = new(int)
		**out = **in
	}
	if in.ThreadPoolWriteSize != nil {
		in, out := &in.ThreadPoolWriteSize, &out.ThreadPoolWriteSize
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Opensearch.
func (in *Opensearch) DeepCopy() *Opensearch {
	if in == nil {
		return nil
	}
	out := new(Opensearch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpensearchDashboards) DeepCopyInto(out *OpensearchDashboards) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.MaxOldSpaceSize != nil {
		in, out := &in.MaxOldSpaceSize, &out.MaxOldSpaceSize
		*out = new(int)
		**out = **in
	}
	if in.OpensearchRequestTimeout != nil {
		in, out := &in.OpensearchRequestTimeout, &out.OpensearchRequestTimeout
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpensearchDashboards.
func (in *OpensearchDashboards) DeepCopy() *OpensearchDashboards {
	if in == nil {
		return nil
	}
	out := new(OpensearchDashboards)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpensearchUserConfig) DeepCopyInto(out *OpensearchUserConfig) {
	*out = *in
	if in.AdditionalBackupRegions != nil {
		in, out := &in.AdditionalBackupRegions, &out.AdditionalBackupRegions
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.CustomDomain != nil {
		in, out := &in.CustomDomain, &out.CustomDomain
		*out = new(string)
		**out = **in
	}
	if in.DisableReplicationFactorAdjustment != nil {
		in, out := &in.DisableReplicationFactorAdjustment, &out.DisableReplicationFactorAdjustment
		*out = new(bool)
		**out = **in
	}
	if in.IndexPatterns != nil {
		in, out := &in.IndexPatterns, &out.IndexPatterns
		*out = make([]*IndexPatterns, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(IndexPatterns)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	if in.IndexTemplate != nil {
		in, out := &in.IndexTemplate, &out.IndexTemplate
		*out = new(IndexTemplate)
		(*in).DeepCopyInto(*out)
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
	if in.KeepIndexRefreshInterval != nil {
		in, out := &in.KeepIndexRefreshInterval, &out.KeepIndexRefreshInterval
		*out = new(bool)
		**out = **in
	}
	if in.MaxIndexCount != nil {
		in, out := &in.MaxIndexCount, &out.MaxIndexCount
		*out = new(int)
		**out = **in
	}
	if in.Openid != nil {
		in, out := &in.Openid, &out.Openid
		*out = new(Openid)
		(*in).DeepCopyInto(*out)
	}
	if in.Opensearch != nil {
		in, out := &in.Opensearch, &out.Opensearch
		*out = new(Opensearch)
		(*in).DeepCopyInto(*out)
	}
	if in.OpensearchDashboards != nil {
		in, out := &in.OpensearchDashboards, &out.OpensearchDashboards
		*out = new(OpensearchDashboards)
		(*in).DeepCopyInto(*out)
	}
	if in.OpensearchVersion != nil {
		in, out := &in.OpensearchVersion, &out.OpensearchVersion
		*out = new(string)
		**out = **in
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
	if in.RecoveryBasebackupName != nil {
		in, out := &in.RecoveryBasebackupName, &out.RecoveryBasebackupName
		*out = new(string)
		**out = **in
	}
	if in.Saml != nil {
		in, out := &in.Saml, &out.Saml
		*out = new(Saml)
		(*in).DeepCopyInto(*out)
	}
	if in.ServiceLog != nil {
		in, out := &in.ServiceLog, &out.ServiceLog
		*out = new(bool)
		**out = **in
	}
	if in.ServiceToForkFrom != nil {
		in, out := &in.ServiceToForkFrom, &out.ServiceToForkFrom
		*out = new(string)
		**out = **in
	}
	if in.StaticIps != nil {
		in, out := &in.StaticIps, &out.StaticIps
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpensearchUserConfig.
func (in *OpensearchUserConfig) DeepCopy() *OpensearchUserConfig {
	if in == nil {
		return nil
	}
	out := new(OpensearchUserConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PrivateAccess) DeepCopyInto(out *PrivateAccess) {
	*out = *in
	if in.Opensearch != nil {
		in, out := &in.Opensearch, &out.Opensearch
		*out = new(bool)
		**out = **in
	}
	if in.OpensearchDashboards != nil {
		in, out := &in.OpensearchDashboards, &out.OpensearchDashboards
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
	if in.Opensearch != nil {
		in, out := &in.Opensearch, &out.Opensearch
		*out = new(bool)
		**out = **in
	}
	if in.OpensearchDashboards != nil {
		in, out := &in.OpensearchDashboards, &out.OpensearchDashboards
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
	if in.Opensearch != nil {
		in, out := &in.Opensearch, &out.Opensearch
		*out = new(bool)
		**out = **in
	}
	if in.OpensearchDashboards != nil {
		in, out := &in.OpensearchDashboards, &out.OpensearchDashboards
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
func (in *Saml) DeepCopyInto(out *Saml) {
	*out = *in
	if in.IdpPemtrustedcasContent != nil {
		in, out := &in.IdpPemtrustedcasContent, &out.IdpPemtrustedcasContent
		*out = new(string)
		**out = **in
	}
	if in.RolesKey != nil {
		in, out := &in.RolesKey, &out.RolesKey
		*out = new(string)
		**out = **in
	}
	if in.SubjectKey != nil {
		in, out := &in.SubjectKey, &out.SubjectKey
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Saml.
func (in *Saml) DeepCopy() *Saml {
	if in == nil {
		return nil
	}
	out := new(Saml)
	in.DeepCopyInto(out)
	return out
}
