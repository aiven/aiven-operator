//go:build !ignore_autogenerated

// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

// Code generated by controller-gen. DO NOT EDIT.

package clickhouseuserconfig

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClickhouseUserConfig) DeepCopyInto(out *ClickhouseUserConfig) {
	*out = *in
	if in.AdditionalBackupRegions != nil {
		in, out := &in.AdditionalBackupRegions, &out.AdditionalBackupRegions
		*out = make([]string, len(*in))
		copy(*out, *in)
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

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClickhouseUserConfig.
func (in *ClickhouseUserConfig) DeepCopy() *ClickhouseUserConfig {
	if in == nil {
		return nil
	}
	out := new(ClickhouseUserConfig)
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
func (in *PrivateAccess) DeepCopyInto(out *PrivateAccess) {
	*out = *in
	if in.Clickhouse != nil {
		in, out := &in.Clickhouse, &out.Clickhouse
		*out = new(bool)
		**out = **in
	}
	if in.ClickhouseHttps != nil {
		in, out := &in.ClickhouseHttps, &out.ClickhouseHttps
		*out = new(bool)
		**out = **in
	}
	if in.ClickhouseMysql != nil {
		in, out := &in.ClickhouseMysql, &out.ClickhouseMysql
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
	if in.Clickhouse != nil {
		in, out := &in.Clickhouse, &out.Clickhouse
		*out = new(bool)
		**out = **in
	}
	if in.ClickhouseHttps != nil {
		in, out := &in.ClickhouseHttps, &out.ClickhouseHttps
		*out = new(bool)
		**out = **in
	}
	if in.ClickhouseMysql != nil {
		in, out := &in.ClickhouseMysql, &out.ClickhouseMysql
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
	if in.Clickhouse != nil {
		in, out := &in.Clickhouse, &out.Clickhouse
		*out = new(bool)
		**out = **in
	}
	if in.ClickhouseHttps != nil {
		in, out := &in.ClickhouseHttps, &out.ClickhouseHttps
		*out = new(bool)
		**out = **in
	}
	if in.ClickhouseMysql != nil {
		in, out := &in.ClickhouseMysql, &out.ClickhouseMysql
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
