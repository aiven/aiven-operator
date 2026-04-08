// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"github.com/aiven/go-client-codegen/handler/opensearch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OpenSearchACLConfigSpec defines the desired state of OpenSearchACLConfig.
type OpenSearchACLConfigSpec struct {
	ServiceDependant `json:",inline"`

	// Enable OpenSearch ACLs. When disabled, authenticated service users have unrestricted access
	Enabled bool `json:"enabled"`

	// List of OpenSearch ACLs
	// +listType=map
	// +listMapKey=username
	Acls []OpenSearchACLConfigACL `json:"acls,omitempty"`
}

// OpenSearchACLConfigACL defines a single OpenSearch ACL entry.
type OpenSearchACLConfigACL struct {
	// +kubebuilder:validation:MinLength=1
	// Username
	Username string `json:"username"`

	// +kubebuilder:validation:Required
	// OpenSearch rules
	Rules []OpenSearchACLConfigRule `json:"rules"`
}

// OpenSearchACLConfigRule defines a single OpenSearch ACL rule.
type OpenSearchACLConfigRule struct {
	// +kubebuilder:validation:MinLength=1
	// OpenSearch index pattern
	Index string `json:"index"`

	// +kubebuilder:validation:Enum=admin;deny;read;readwrite;write
	// OpenSearch permission
	Permission opensearch.PermissionType `json:"permission"`
}

// OpenSearchACLConfigStatus defines the observed state of OpenSearchACLConfig
type OpenSearchACLConfigStatus struct {
	// Conditions represent the latest available observations of an OpenSearchACLConfig state
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OpenSearchACLConfig is the Schema for the opensearchaclconfigs API.
// Manages the full OpenSearch ACL configuration for one Aiven OpenSearch service.
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Enabled",type="boolean",JSONPath=".spec.enabled"
type OpenSearchACLConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenSearchACLConfigSpec   `json:"spec,omitempty"`
	Status OpenSearchACLConfigStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &OpenSearchACLConfig{}

func (in *OpenSearchACLConfig) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *OpenSearchACLConfig) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *OpenSearchACLConfig) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (*OpenSearchACLConfig) NoSecret() bool {
	return true
}

// +kubebuilder:object:root=true

// OpenSearchACLConfigList contains a list of OpenSearchACLConfig.
type OpenSearchACLConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenSearchACLConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenSearchACLConfig{}, &OpenSearchACLConfigList{})
}
