//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

// Code generated by controller-gen. DO NOT EDIT.

package redisuserconfig

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
func (in *PrivateAccess) DeepCopyInto(out *PrivateAccess) {
	*out = *in
	if in.Prometheus != nil {
		in, out := &in.Prometheus, &out.Prometheus
		*out = new(bool)
		**out = **in
	}
	if in.Redis != nil {
		in, out := &in.Redis, &out.Redis
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
	if in.Prometheus != nil {
		in, out := &in.Prometheus, &out.Prometheus
		*out = new(bool)
		**out = **in
	}
	if in.Redis != nil {
		in, out := &in.Redis, &out.Redis
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
	if in.Prometheus != nil {
		in, out := &in.Prometheus, &out.Prometheus
		*out = new(bool)
		**out = **in
	}
	if in.Redis != nil {
		in, out := &in.Redis, &out.Redis
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
func (in *RedisUserConfig) DeepCopyInto(out *RedisUserConfig) {
	*out = *in
	if in.AdditionalBackupRegions != nil {
		in, out := &in.AdditionalBackupRegions, &out.AdditionalBackupRegions
		*out = make([]string, len(*in))
		copy(*out, *in)
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
	if in.RedisAclChannelsDefault != nil {
		in, out := &in.RedisAclChannelsDefault, &out.RedisAclChannelsDefault
		*out = new(string)
		**out = **in
	}
	if in.RedisIoThreads != nil {
		in, out := &in.RedisIoThreads, &out.RedisIoThreads
		*out = new(int)
		**out = **in
	}
	if in.RedisLfuDecayTime != nil {
		in, out := &in.RedisLfuDecayTime, &out.RedisLfuDecayTime
		*out = new(int)
		**out = **in
	}
	if in.RedisLfuLogFactor != nil {
		in, out := &in.RedisLfuLogFactor, &out.RedisLfuLogFactor
		*out = new(int)
		**out = **in
	}
	if in.RedisMaxmemoryPolicy != nil {
		in, out := &in.RedisMaxmemoryPolicy, &out.RedisMaxmemoryPolicy
		*out = new(string)
		**out = **in
	}
	if in.RedisNotifyKeyspaceEvents != nil {
		in, out := &in.RedisNotifyKeyspaceEvents, &out.RedisNotifyKeyspaceEvents
		*out = new(string)
		**out = **in
	}
	if in.RedisNumberOfDatabases != nil {
		in, out := &in.RedisNumberOfDatabases, &out.RedisNumberOfDatabases
		*out = new(int)
		**out = **in
	}
	if in.RedisPersistence != nil {
		in, out := &in.RedisPersistence, &out.RedisPersistence
		*out = new(string)
		**out = **in
	}
	if in.RedisPubsubClientOutputBufferLimit != nil {
		in, out := &in.RedisPubsubClientOutputBufferLimit, &out.RedisPubsubClientOutputBufferLimit
		*out = new(int)
		**out = **in
	}
	if in.RedisSsl != nil {
		in, out := &in.RedisSsl, &out.RedisSsl
		*out = new(bool)
		**out = **in
	}
	if in.RedisTimeout != nil {
		in, out := &in.RedisTimeout, &out.RedisTimeout
		*out = new(int)
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RedisUserConfig.
func (in *RedisUserConfig) DeepCopy() *RedisUserConfig {
	if in == nil {
		return nil
	}
	out := new(RedisUserConfig)
	in.DeepCopyInto(out)
	return out
}