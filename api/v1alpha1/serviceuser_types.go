// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//"project": {
//		Type:        schema.TypeString,
//		Required:    true,
//		Description: "Project to link the user to",
//		ForceNew:    true,
//	},
//	"service_name": {
//		Type:        schema.TypeString,
//		Required:    true,
//		Description: "Service to link the user to",
//		ForceNew:    true,
//	},
//	"username": {
//		Type:        schema.TypeString,
//		Required:    true,
//		Description: "Name of the user account",
//		ForceNew:    true,
//	},
//	"password": {
//		Type:             schema.TypeString,
//		Sensitive:        true,
//		Computed:         true,
//		Optional:         true,
//		Description:      "Password of the user",
//		DiffSuppressFunc: emptyObjectDiffSuppressFunc,
//	},
//	"authentication": {
//		Type:             schema.TypeString,
//		Optional:         true,
//		Description:      "Authentication details",
//		DiffSuppressFunc: emptyObjectDiffSuppressFunc,
//		ValidateFunc:     validation.StringInSlice([]string{"caching_sha2_password", "mysql_native_password"}, false),
//	},
//	"type": {
//		Type:        schema.TypeString,
//		Computed:    true,
//		Description: "Type of the user account",
//	},
//	"access_cert": {
//		Type:        schema.TypeString,
//		Sensitive:   true,
//		Computed:    true,
//		Description: "Access certificate for the user if applicable for the service in question",
//	},
//	"access_key": {
//		Type:        schema.TypeString,
//		Sensitive:   true,
//		Computed:    true,
//		Description: "Access certificate key for the user if applicable for the service in question",
//	},

// ServiceUserSpec defines the desired state of ServiceUser
type ServiceUserSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// x-kubernetes-immutable: true
	// Project to link the user to
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// x-kubernetes-immutable: true
	// Service to link the user to
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=63
	// x-kubernetes-immutable: true
	// Name of the user account
	Username string `json:"username"`

	// +kubebuilder:validation:Enum=caching_sha2_password;mysql_native_password
	// Authentication details
	Authentication string `json:"authentication,omitempty"`
}

// ServiceUserStatus defines the observed state of ServiceUser
type ServiceUserStatus struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Project to link the user to
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=63
	// Service to link the user to
	ServiceName string `json:"service_name"`

	// +kubebuilder:validation:MaxLength=63
	// Name of the user account
	Username string `json:"username"`

	// +kubebuilder:validation:Enum=caching_sha2_password;mysql_native_password
	// Authentication details
	Authentication string `json:"authentication,omitempty"`

	// Type of the user account
	Type string `json:"type,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceUser is the Schema for the serviceusers API
type ServiceUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceUserSpec   `json:"spec,omitempty"`
	Status ServiceUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceUserList contains a list of ServiceUser
type ServiceUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceUser{}, &ServiceUserList{})
}
