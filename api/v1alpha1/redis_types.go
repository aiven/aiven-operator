// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	redisuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/redis"
)

// RedisSpec defines the desired state of Redis
type RedisSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Redis specific user configuration options
	UserConfig *redisuserconfig.RedisUserConfig `json:"userConfig,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:deprecatedversion:warning="EOL date **March 31, 2025**. Please follow [these](https://aiven.io/docs/products/caching/howto/upgrade-aiven-for-caching-to-valkey#upgrade-service) instructions to upgrade your service to Valkey."

// Redis is the Schema for the redis API.
// Info "Exposes secret keys": `REDIS_HOST`, `REDIS_PORT`, `REDIS_USER`, `REDIS_PASSWORD`, `REDIS_SSL`
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
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

func (in *Redis) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
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
