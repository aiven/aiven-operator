// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// +kubebuilder:validation:MaxLength=64
	// Credit card ID; The ID may be either last 4 digits of the card or the actual ID
	CardID string `json:"cardId,omitempty"`

	// +kubebuilder:validation:MaxLength=32
	// Account ID
	AccountID string `json:"accountId,omitempty"`

	// +kubebuilder:validation:MaxLength=1000
	// Billing name and address of the project
	BillingAddress string `json:"billingAddress,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// Billing contact emails of the project
	BillingEmails []string `json:"billingEmails,omitempty"`

	// +kubebuilder:validation:Enum=AUD;CAD;CHF;DKK;EUR;GBP;NOK;SEK;USD
	// Billing currency
	BillingCurrency string `json:"billingCurrency,omitempty"`

	// +kubebuilder:validation:MaxLength=1000
	// Extra text to be included in all project invoices, e.g. purchase order or cost center number
	BillingExtraText string `json:"billingExtraText,omitempty"`

	// +kubebuilder:validation:MaxLength=36
	// +kubebuilder:validation:MinLength=36
	// BillingGroup ID
	BillingGroupID string `json:"billingGroupId,omitempty"`

	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=2
	// Billing country code of the project
	CountryCode string `json:"countryCode,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Target cloud, example: aws-eu-central-1
	Cloud string `json:"cloud,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Project name from which to copy settings to the new project
	CopyFromProject string `json:"copyFromProject,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// Technical contact emails of the project
	TechnicalEmails []string `json:"technicalEmails,omitempty"`

	// Information regarding secret creation
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`

	// Tags are key-value pairs that allow you to categorize projects
	Tags map[string]string `json:"tags,omitempty"`

	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	// Conditions represent the latest available observations of an Project state
	Conditions []metav1.Condition `json:"conditions"`

	// +kubebuilder:validation:MaxLength=64
	// EU VAT Identification Number
	VatID string `json:"vatId,omitempty"`

	// Available credirs
	AvailableCredits string `json:"availableCredits,omitempty"`

	// Country name
	Country string `json:"country,omitempty"`

	// Estimated balance
	EstimatedBalance string `json:"estimatedBalance,omitempty"`

	// Payment method name
	PaymentMethod string `json:"paymentMethod,omitempty"`
}

// +kubebuilder:object:root=true

// Project is the Schema for the projects API
// +kubebuilder:subresource:status
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

func (proj Project) AuthSecretRef() *AuthSecretReference {
	return proj.Spec.AuthSecretRef
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
