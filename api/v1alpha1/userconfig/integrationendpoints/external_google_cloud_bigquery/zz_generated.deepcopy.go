//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

// Code generated by controller-gen. DO NOT EDIT.

package externalgooglecloudbigqueryuserconfig

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalGoogleCloudBigqueryUserConfig) DeepCopyInto(out *ExternalGoogleCloudBigqueryUserConfig) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalGoogleCloudBigqueryUserConfig.
func (in *ExternalGoogleCloudBigqueryUserConfig) DeepCopy() *ExternalGoogleCloudBigqueryUserConfig {
	if in == nil {
		return nil
	}
	out := new(ExternalGoogleCloudBigqueryUserConfig)
	in.DeepCopyInto(out)
	return out
}
