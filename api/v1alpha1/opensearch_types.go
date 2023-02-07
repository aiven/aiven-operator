// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	opensearchuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/opensearch"
)

// OpenSearchSpec defines the desired state of OpenSearch
type OpenSearchSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

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

func (in *OpenSearch) AuthSecretRef() AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *OpenSearch) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func init() {
	SchemeBuilder.Register(&OpenSearch{}, &OpenSearchList{})
}
