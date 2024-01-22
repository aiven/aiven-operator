// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	opensearchuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/opensearch"
)

// OpenSearchSpec defines the desired state of OpenSearch
// +kubebuilder:validation:XValidation:rule="has(oldSelf.connInfoSecretTargetDisabled) == has(self.connInfoSecretTargetDisabled)",message="connInfoSecretTargetDisabled can only be set during resource creation."
type OpenSearchSpec struct {
	ServiceCommonSpec `json:",inline"`

	// +kubebuilder:validation:Format="^[1-9][0-9]*(GiB|G)*"
	// The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing.
	DiskSpace string `json:"disk_space,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`

	// Information regarding secret creation.
	// Exposed keys: `OPENSEARCH_HOST`, `OPENSEARCH_PORT`, `OPENSEARCH_USER`, `OPENSEARCH_PASSWORD`
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="connInfoSecretTargetDisabled is immutable."
	// When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
	ConnInfoSecretTargetDisabled *bool `json:"connInfoSecretTargetDisabled,omitempty"`

	// OpenSearch specific user configuration options
	UserConfig *opensearchuserconfig.OpensearchUserConfig `json:"userConfig,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenSearch is the Schema for the opensearches API
type OpenSearch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenSearchSpec `json:"spec,omitempty"`
	Status ServiceStatus  `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenSearchList contains a list of OpenSearch
type OpenSearchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenSearch `json:"items"`
}

var _ AivenManagedObject = &OpenSearch{}

func (in *OpenSearch) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *OpenSearch) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *OpenSearch) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func (in *OpenSearch) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

func init() {
	SchemeBuilder.Register(&OpenSearch{}, &OpenSearchList{})
}
