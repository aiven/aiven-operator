// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"github.com/aiven/go-client-codegen/handler/service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceUserSpec defines the desired state of ServiceUser
// +kubebuilder:validation:XValidation:rule="has(self.username) == has(oldSelf.username)",message="username can only be set during resource creation."
type ServiceUserSpec struct {
	ServiceDependant `json:",inline"`
	SecretFields     `json:",inline"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="username is immutable."
	// Username of the service user on Aiven.
	// Defaults to the K8S resource name. Aiven accepts usernames that are not valid
	// Kubernetes object names (e.g. containing underscores or uppercase characters);
	// set this field to manage such users.
	// Can only be set at creation and is immutable afterward.
	// To use a different username, delete the resource and create a new one.
	// Multiple ServiceUser resources referencing the same username are not supported.
	// Ensure each Aiven service user is managed by at most one resource.
	Username string `json:"username,omitempty"`

	// ConnInfoSecretSource declares the password the operator should enforce on the user.
	// Direct password changes in the database will be reverted on the next reconcile cycle.
	// To rotate, update the source secret.
	// When unset, the Operator does not manage the password:
	// the target secret holds whatever password the Aiven API returns.
	// If the password is later changed directly in the database,
	// the API returns an empty value and the target secret is emptied on the next reconcile.
	// Password must be 8-256 characters long.
	ConnInfoSecretSource *ConnInfoSecretSource `json:"connInfoSecretSource,omitempty"`

	// AccessControl Service type specific access control rules for user.
	// When this block is present, the operator manages the full access-control scope it contains.
	AccessControl *ServiceUserAccessControl `json:"accessControl,omitempty"`

	// +kubebuilder:validation:Enum=caching_sha2_password;mysql_native_password
	// Authentication details
	Authentication service.AuthenticationType `json:"authentication,omitempty"`
}

// ServiceUserAccessControl defines the full desired Valkey ACL snapshot managed by the operator.
//
// When this block is present, omitted inner fields are treated as empty lists.
//
// Valkey command and category rules are order-sensitive because ACL SETUSER
// applies rules from left to right:
// https://valkey.io/commands/acl-setuser/
//
// Key and channel rules are treated as unordered collections for drift detection.
// The operator intentionally ignores their remote ordering to avoid false drift
// if the backend returns an equivalent canonicalized ACL via ACL GETUSER:
// https://valkey.io/commands/acl-getuser/
type ServiceUserAccessControl struct {
	// Key access rules.
	ValkeyACLKeys []string `json:"valkeyAclKeys,omitempty"`

	// Rules for individual commands. Order matters.
	ValkeyACLCommands []string `json:"valkeyAclCommands,omitempty"`

	// Command category rules. Order matters.
	ValkeyACLCategories []string `json:"valkeyAclCategories,omitempty"`

	// Glob-style patterns defining which pub/sub channels can be accessed.
	ValkeyACLChannels []string `json:"valkeyAclChannels,omitempty"`
}

// ServiceUserStatus defines the observed state of ServiceUser
type ServiceUserStatus struct {
	// Conditions represent the latest available observations of an ServiceUser state
	Conditions []metav1.Condition `json:"conditions"`

	// Type of the user account
	Type string `json:"type,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceUser is the Schema for the serviceusers API.
// Creates a service user for accessing Aiven services. The ServiceUser resource name becomes the username in Aiven, unless spec.username overrides it.
// Built-in users like 'avnadmin' cannot be deleted but their passwords can be modified using connInfoSecretSource.
// Info "Exposes secret keys": `SERVICEUSER_HOST`, `SERVICEUSER_PORT`, `SERVICEUSER_USERNAME`, `SERVICEUSER_PASSWORD`, `SERVICEUSER_CA_CERT`, `SERVICEUSER_ACCESS_CERT`, `SERVICEUSER_ACCESS_KEY`, `SERVICEUSER_SASL_HOST`, `SERVICEUSER_SASL_PORT`, `SERVICEUSER_SCHEMA_REGISTRY_HOST`, `SERVICEUSER_SCHEMA_REGISTRY_PORT`
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Connection Information Secret",type="string",JSONPath=".spec.connInfoSecretTarget.name"
type ServiceUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceUserSpec   `json:"spec,omitempty"`
	Status ServiceUserStatus `json:"status,omitempty"`
}

func (in *ServiceUser) GetConnInfoSecretSource() *ConnInfoSecretSource {
	return in.Spec.ConnInfoSecretSource
}

// GetUsername returns the Aiven username for the ServiceUser.
// Defaults to Spec.Username and falls back to ObjectMeta.Name when empty.
func (in *ServiceUser) GetUsername() string {
	if in.Spec.Username != "" {
		return in.Spec.Username
	}
	return in.Name
}

var _ AivenManagedObject = &ServiceUser{}

func (in *ServiceUser) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *ServiceUser) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *ServiceUser) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *ServiceUser) NoSecret() bool {
	return in.Spec.ConnInfoSecretTargetDisabled != nil && *in.Spec.ConnInfoSecretTargetDisabled
}

func (in *ServiceUser) GetConnInfoSecretTarget() ConnInfoSecretTarget {
	return in.Spec.ConnInfoSecretTarget
}

// +kubebuilder:object:root=true

// ServiceUserList contains a list of ServiceUser
type ServiceUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceUser `json:"items"`
}
