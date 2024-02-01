//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

// Code generated by controller-gen. DO NOT EDIT.

package externalawscloudwatchmetricsuserconfig

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DroppedMetrics) DeepCopyInto(out *DroppedMetrics) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DroppedMetrics.
func (in *DroppedMetrics) DeepCopy() *DroppedMetrics {
	if in == nil {
		return nil
	}
	out := new(DroppedMetrics)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalAwsCloudwatchMetricsUserConfig) DeepCopyInto(out *ExternalAwsCloudwatchMetricsUserConfig) {
	*out = *in
	if in.DroppedMetrics != nil {
		in, out := &in.DroppedMetrics, &out.DroppedMetrics
		*out = make([]*DroppedMetrics, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(DroppedMetrics)
				**out = **in
			}
		}
	}
	if in.ExtraMetrics != nil {
		in, out := &in.ExtraMetrics, &out.ExtraMetrics
		*out = make([]*ExtraMetrics, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(ExtraMetrics)
				**out = **in
			}
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalAwsCloudwatchMetricsUserConfig.
func (in *ExternalAwsCloudwatchMetricsUserConfig) DeepCopy() *ExternalAwsCloudwatchMetricsUserConfig {
	if in == nil {
		return nil
	}
	out := new(ExternalAwsCloudwatchMetricsUserConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExtraMetrics) DeepCopyInto(out *ExtraMetrics) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExtraMetrics.
func (in *ExtraMetrics) DeepCopy() *ExtraMetrics {
	if in == nil {
		return nil
	}
	out := new(ExtraMetrics)
	in.DeepCopyInto(out)
	return out
}
