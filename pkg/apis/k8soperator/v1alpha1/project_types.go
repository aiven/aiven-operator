package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	CACertificate string `json:"ca_certificate"`
}

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format=^[a-zA-Z0-9_-]*$
	Name string `json:"name"`

	// +kubebuilder:validation:MaxLength=64
	CardId string `json:"card_id,omitempty"`

	// +kubebuilder:validation:MaxLength=32
	AccountId string `json:"account_id,omitempty"`

	// +kubebuilder:validation:MaxLength=1000
	BillingAddress string `json:"billing_address,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:UniqueItems=true
	BillingEmails []string `json:"billing_emails,omitempty"`

	// +kubebuilder:validation:Enum=AUD,CAD,CHF,DKK,EUR,GBP,NOK,SEK,USD
	BillingCurrency string `json:"billing_currency,omitempty"`

	// +kubebuilder:validation:MaxLength=1000
	BillingExtraText string `json:"billing_extra_text,omitempty"`

	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=2
	CountryCode string `json:"country_code,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	Cloud string `json:"cloud,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	CopyFromProject string `json:"copy_from_project,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:UniqueItems=true
	TechnicalEmails []string `json:"technical_emails,omitempty"`

	// +kubebuilder:validation:MaxLength=64
	VatId string `json:"vat_id,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Project is the Schema for the projects API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=projects,scope=Namespaced
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectList contains a list of Project
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
