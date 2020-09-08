/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProjectSpec defines the desired state of Project
type ProjectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// x-kubernetes-immutable: false
	// Project name
	Name string `json:"name"`

	// +kubebuilder:validation:MaxLength=64
	// Credit card ID
	CardId string `json:"card_id,omitempty"`

	// +kubebuilder:validation:MaxLength=32
	// Account ID
	AccountId string `json:"account_id,omitempty"`

	// +kubebuilder:validation:MaxLength=1000
	// Billing name and address of the project
	BillingAddress string `json:"billing_address,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// Billing contact emails of the project
	BillingEmails []string `json:"billing_emails,omitempty"`

	// +kubebuilder:validation:Enum=AUD;CAD;CHF;DKK;EUR;GBP;NOK;SEK;USD
	// Billing currency
	BillingCurrency string `json:"billing_currency,omitempty"`

	// +kubebuilder:validation:MaxLength=1000
	// Extra text to be included in all project invoices, e.g. purchase order or cost center number
	BillingExtraText string `json:"billing_extra_text,omitempty"`

	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=2
	// Billing country code of the project
	CountryCode string `json:"country_code,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Target cloud, example: aws-eu-central-1
	Cloud string `json:"cloud,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Project name from which to copy settings to the new project
	CopyFromProject string `json:"copy_from_project,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// Technical contact emails of the project
	TechnicalEmails []string `json:"technical_emails,omitempty"`
}

//BillingAddress   *string          `json:"billing_address,omitempty"`
//BillingEmails    *[]*ContactEmail `json:"billing_emails,omitempty"`
//BillingExtraText *string          `json:"billing_extra_text,omitempty"`
//CardID           string           `json:"card_id,omitempty"`
//Cloud            string           `json:"cloud,omitempty"`
//CopyFromProject  string           `json:"copy_from_project,omitempty"`
//CountryCode      *string          `json:"country_code,omitempty"`
//Project          string           `json:"project"`
//AccountId        string           `json:"account_id,omitempty"`
//TechnicalEmails  *[]*ContactEmail `json:"tech_emails,omitempty"`
//BillingCurrency  string           `json:"billing_currency,omitempty"`

// ProjectStatus defines the observed state of Project
type ProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Project name
	Name string `json:"name"`

	// +kubebuilder:validation:MaxLength=64
	// Credit card ID
	CardId string `json:"card_id,omitempty"`

	// +kubebuilder:validation:MaxLength=32
	// Account ID
	AccountId string `json:"account_id,omitempty"`

	// +kubebuilder:validation:MaxLength=1000
	// Billing name and address of the project
	BillingAddress string `json:"billing_address,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// Billing contact emails of the project
	BillingEmails []string `json:"billing_emails,omitempty"`

	// +kubebuilder:validation:Enum=AUD;CAD;CHF;DKK;EUR;GBP;NOK;SEK;USD
	// Billing currency
	BillingCurrency string `json:"billing_currency,omitempty"`

	// +kubebuilder:validation:MaxLength=1000
	// Extra text to be included in all project invoices, e.g. purchase order or cost center number
	BillingExtraText string `json:"billing_extra_text,omitempty"`

	// +kubebuilder:validation:MinLength=2
	// +kubebuilder:validation:MaxLength=2
	// Billing country code of the project
	CountryCode string `json:"country_code,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Target cloud, example: aws-eu-central-1
	Cloud string `json:"cloud,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// Project name from which to copy settings to the new project
	CopyFromProject string `json:"copy_from_project,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// Technical contact emails of the project
	TechnicalEmails []string `json:"technical_emails,omitempty"`

	// +kubebuilder:validation:MaxLength=64
	// EU VAT Identification Number
	VatId string `json:"vat_id,omitempty"`

	AvailableCredits string `json:"available_credits,omitempty"`

	// Country name
	Country string `json:"country,omitempty"`

	// Estimated balance
	EstimatedBalance string `json:"estimated_balance,omitempty"`

	// Payment method name
	PaymentMethod string `json:"payment_method,omitempty"`
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
