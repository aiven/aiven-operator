// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	alloydbomni "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/alloydbomni"
)

// AlloyDBOmniSpec defines the desired state of AlloyDB Omni instance
type AlloyDBOmniSpec struct {
	ServiceCommonSpec `json:",inline"`

	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=string
	// Your [Google service account key](https://cloud.google.com/iam/docs/service-account-creds#key-types) in JSON format.
	ServiceAccountCredentials string `json:"serviceAccountCredentials,omitempty"`

	// AlloyDBOmni specific user configuration options
	UserConfig *alloydbomni.AlloydbomniUserConfig `json:"userConfig,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion:warning="AlloyDBOmni is deprecated and will be removed in a future version"

// AlloyDBOmni is the Schema for the alloydbomni API.
// Info "Exposes secret keys": `ALLOYDBOMNI_HOST`, `ALLOYDBOMNI_PORT`, `ALLOYDBOMNI_DATABASE`, `ALLOYDBOMNI_USER`, `ALLOYDBOMNI_PASSWORD`, `ALLOYDBOMNI_SSLMODE`, `ALLOYDBOMNI_DATABASE_URI`
// Deprecated: End of life notice - Aiven for AlloyDB Omni is entering its end-of-life cycle. See https://aiven.io/docs/platform/reference/end-of-life for details. From 5 September 2025, you can no longer create new Aiven for AlloyDB Omni services. Existing services continue to operate until the end of life (EOL) date but you cannot change plans for these services. On 5 December 2025, all active Aiven for AlloyDB Omni services are powered off and deleted, making data from these services inaccessible. The recommended alternatives are Aiven for PostgreSQL®, Aiven for ClickHouse®, and Aiven for MySQL®. To ensure uninterrupted service, complete your migration before December 5, 2025. For further assistance, contact the Aiven support team at support@aiven.io or your account team.
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Region",type="string",JSONPath=".spec.cloudName"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.plan"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
type AlloyDBOmni struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlloyDBOmniSpec `json:"spec,omitempty"`
	Status ServiceStatus   `json:"status,omitempty"`
}

var _ AivenManagedObject = &AlloyDBOmni{}

func (in *AlloyDBOmni) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *AlloyDBOmni) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *AlloyDBOmni) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *AlloyDBOmni) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *AlloyDBOmni) GetRefs() []*ResourceReferenceObject {
	return in.Spec.GetRefs(in.GetNamespace())
}

func (in *AlloyDBOmni) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

// +kubebuilder:object:root=true

// AlloyDBOmniList contains a list of AlloyDBOmni instances
type AlloyDBOmniList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AlloyDBOmni `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AlloyDBOmni{}, &AlloyDBOmniList{})
}
