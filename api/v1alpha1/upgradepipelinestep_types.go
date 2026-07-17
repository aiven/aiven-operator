// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// UpgradePipelineStepSpec defines the desired state of UpgradePipelineStep.
type UpgradePipelineStepSpec struct {
	AuthSecretRefField `json:",inline"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// OrganizationID is the Aiven organization ID that owns the upgrade pipeline step.
	OrganizationID string `json:"organizationId"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9_-]+$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// SourceProjectName is the project name of the service that must be upgraded first.
	SourceProjectName string `json:"sourceProjectName"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-z][-a-z0-9]+$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// SourceServiceName is the service name that must be upgraded first.
	SourceServiceName string `json:"sourceServiceName"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9_-]+$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// DestinationProjectName is the project name of the service that waits for the source service.
	DestinationProjectName string `json:"destinationProjectName"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-z][-a-z0-9]+$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// DestinationServiceName is the service name that waits for the source service.
	DestinationServiceName string `json:"destinationServiceName"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=7
	// AutoValidationDelayDays is the number of days before Aiven can automatically validate the source service.
	AutoValidationDelayDays *int `json:"autoValidationDelayDays,omitempty"`
}

// UpgradePipelineStepLastValidationStatus describes the last validation observed for an upgrade pipeline step.
type UpgradePipelineStepLastValidationStatus struct {
	// Comment is the validation comment.
	Comment string `json:"comment,omitempty"`

	// ValidatedAt is the time the validation was created.
	ValidatedAt *metav1.Time `json:"validatedAt,omitempty"`

	// ValidatedByUser is the user who created the validation. It is empty for auto-validation.
	ValidatedByUser string `json:"validatedByUser,omitempty"`
}

// UpgradePipelineStepStatus defines the observed state of UpgradePipelineStep.
type UpgradePipelineStepStatus struct {
	// Conditions represent the latest available observations of an UpgradePipelineStep state.
	Conditions []metav1.Condition `json:"conditions"`

	// ID is the Aiven upgrade pipeline step ID.
	ID string `json:"id,omitempty"`

	// LastValidation contains the last validation information returned by the Aiven API.
	LastValidation *UpgradePipelineStepLastValidationStatus `json:"lastValidation,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// UpgradePipelineStep is the Schema for the upgradepipelinesteps API.
// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".spec.organizationId"
// +kubebuilder:printcolumn:name="Source Project",type="string",JSONPath=".spec.sourceProjectName"
// +kubebuilder:printcolumn:name="Source Service",type="string",JSONPath=".spec.sourceServiceName"
// +kubebuilder:printcolumn:name="Destination Project",type="string",JSONPath=".spec.destinationProjectName"
// +kubebuilder:printcolumn:name="Destination Service",type="string",JSONPath=".spec.destinationServiceName"
// +kubebuilder:printcolumn:name="Step ID",type="string",JSONPath=".status.id"
type UpgradePipelineStep struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UpgradePipelineStepSpec   `json:"spec,omitempty"`
	Status UpgradePipelineStepStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &UpgradePipelineStep{}

func (*UpgradePipelineStep) NoSecret() bool {
	return true
}

func (in *UpgradePipelineStep) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *UpgradePipelineStep) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *UpgradePipelineStep) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

// +kubebuilder:object:root=true

// UpgradePipelineStepList contains a list of UpgradePipelineStep.
type UpgradePipelineStepList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UpgradePipelineStep `json:"items"`
}
