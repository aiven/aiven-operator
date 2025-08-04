// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	opensearchuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/opensearch"
)

// OpenSearchSpec defines the desired state of OpenSearch
type OpenSearchSpec struct {
	ServiceCommonSpec `json:",inline"`

	// OpenSearch specific user configuration options
	UserConfig *opensearchuserconfig.OpensearchUserConfig `json:"userConfig,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// OpenSearch is the Schema for the opensearches API.
// Info "Exposes secret keys": `OPENSEARCH_HOST`, `OPENSEARCH_PORT`, `OPENSEARCH_USER`, `OPENSEARCH_PASSWORD`, `OPENSEARCH_URI`
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
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

func (in *OpenSearch) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *OpenSearch) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
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
