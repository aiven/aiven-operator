//go:build !ignore_autogenerated

// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

// Code generated by controller-gen. DO NOT EDIT.

package externalawscloudwatchmetricsuserconfig

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalAwsCloudwatchMetricsUserConfig) DeepCopyInto(out *ExternalAwsCloudwatchMetricsUserConfig) {
	*out = *in
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
