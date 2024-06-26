//go:build !ignore_autogenerated

// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

// Code generated by controller-gen. DO NOT EDIT.

package externalschemaregistryuserconfig

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExternalSchemaRegistryUserConfig) DeepCopyInto(out *ExternalSchemaRegistryUserConfig) {
	*out = *in
	if in.BasicAuthPassword != nil {
		in, out := &in.BasicAuthPassword, &out.BasicAuthPassword
		*out = new(string)
		**out = **in
	}
	if in.BasicAuthUsername != nil {
		in, out := &in.BasicAuthUsername, &out.BasicAuthUsername
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExternalSchemaRegistryUserConfig.
func (in *ExternalSchemaRegistryUserConfig) DeepCopy() *ExternalSchemaRegistryUserConfig {
	if in == nil {
		return nil
	}
	out := new(ExternalSchemaRegistryUserConfig)
	in.DeepCopyInto(out)
	return out
}
