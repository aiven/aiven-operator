// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pguserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfigs/pg"
)

// PostgreSQLSpec defines the desired state of postgres instance
type PostgreSQLSpec struct {
	ServiceCommonSpec `json:",inline"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// PostgreSQL specific user configuration options
	UserConfig *pguserconfig.PgUserConfig `json:"userConfig,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PostgreSQL is the Schema for the postgresql API
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type PostgreSQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLSpec `json:"spec,omitempty"`
	Status ServiceStatus  `json:"status,omitempty"`
}

func (in *PostgreSQL) AuthSecretRef() AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *PostgreSQL) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

// +kubebuilder:object:root=true

// PostgreSQLList contains a list of PostgreSQL instances
type PostgreSQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
}
