// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// OrganizationProjectSpec defines the desired state of OrganizationProject.
type OrganizationProjectSpec struct {
	AuthSecretRefField `json:",inline"`
	SecretFields       `json:",inline"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// OrganizationID is the Aiven organization ID that owns the project.
	// It is the addressing key for the project and cannot be changed (moving a project
	// between organizations is not supported).
	OrganizationID string `json:"organizationId"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9_-]+$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// ProjectID is the name of the project. It is immutable once set.
	ProjectID string `json:"projectId"`

	// +kubebuilder:validation:MinLength=1
	// BillingGroupID is the ID of the billing group the project is assigned to.
	BillingGroupID string `json:"billingGroupId"`

	// +kubebuilder:validation:MinLength=1
	// ParentID is the ID of the organization or organizational unit the project belongs to.
	// Moving a project between organizational units within the same organization is supported.
	ParentID string `json:"parentId"`

	// +kubebuilder:validation:Minimum=10000
	// +kubebuilder:validation:Maximum=30000
	// BasePort is the valid port number range for the project, from 10000 to 30000.
	// When omitted, the field is unmanaged: Aiven assigns the value and changes made
	// outside Kubernetes are left as is.
	BasePort *int `json:"basePort,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:validation:items:MaxLength=254
	// +kubebuilder:validation:XValidation:rule="self.all(x, self.exists_one(y, y == x))",message="Emails must be unique"
	// TechnicalEmails are the technical contact emails of the project.
	// This list is authoritative: when omitted, emails added outside Kubernetes are removed.
	// Duplicates are rejected.
	TechnicalEmails []string `json:"technicalEmails,omitempty"`

	// Tags are key-value pairs that allow you to categorize projects.
	// This map is authoritative: when omitted, tags added outside
	// Kubernetes are removed.
	Tags map[string]string `json:"tags,omitempty"`
}

// OrganizationProjectStatus defines the observed state of OrganizationProject.
type OrganizationProjectStatus struct {
	// Conditions represent the latest available observations of an OrganizationProject state.
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OrganizationProject is the Schema for the organizationprojects API.
// Warning "Adoption of existing projects":
// If `projectId` refers to a project that already exists in the organization, the operator adopts it:
// the remote state is overwritten to match the spec (billing group, parent, tags, technical emails, base port),
// and deleting the resource deletes the project in Aiven.
//
// Info "Exposes secret keys": `ORGANIZATIONPROJECT_CA_CERT`
// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".spec.organizationId"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.projectId"
// +kubebuilder:printcolumn:name="Parent",type="string",JSONPath=".spec.parentId"
type OrganizationProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationProjectSpec   `json:"spec,omitempty"`
	Status OrganizationProjectStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &OrganizationProject{}

func (in *OrganizationProject) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *OrganizationProject) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *OrganizationProject) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *OrganizationProject) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *OrganizationProject) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

// +kubebuilder:object:root=true

// OrganizationProjectList contains a list of OrganizationProject.
type OrganizationProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationProject `json:"items"`
}
