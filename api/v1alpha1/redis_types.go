// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/redis"
)

// RedisSpec defines the desired state of Redis
// +kubebuilder:validation:XValidation:rule="has(oldSelf.connInfoSecretTargetDisabled) == has(self.connInfoSecretTargetDisabled)",message="connInfoSecretTargetDisabled can only be set during resource creation."
type RedisSpec struct {
	ServiceCommonSpec `json:",inline"`

	// +kubebuilder:validation:Format="^[1-9][0-9]*(GiB|G)*"
	// The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
	DiskSpace string `json:"disk_space,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`

	// Information regarding secret creation.
	// Exposed keys: `REDIS_HOST`, `REDIS_PORT`, `REDIS_USER`, `REDIS_PASSWORD`
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="connInfoSecretTargetDisabled is immutable."
	// When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
	ConnInfoSecretTargetDisabled *bool `json:"connInfoSecretTargetDisabled,omitempty"`

	// Redis specific user configuration options
	UserConfig *redisuserconfig.RedisUserConfig `json:"userConfig,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Redis is the Schema for the redis API
// +kubebuilder:subresource:status
type Redis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisSpec     `json:"spec,omitempty"`
	Status ServiceStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &Redis{}

func (in *Redis) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *Redis) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *Redis) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *Redis) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func (in *Redis) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

//+kubebuilder:object:root=true

// RedisList contains a list of Redis
type RedisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Redis `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Redis{}, &RedisList{})
}
