package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// AuthSecretReference references a Secret containing an Aiven authentication token
type AuthSecretReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// ConnInfoSecretTarget contains information secret name
type ConnInfoSecretTarget struct {
	// Name of the Secret resource to be created
	Name string `json:"name"`
}

// ServiceStatus defines the observed state of service
type ServiceStatus struct {
	// Conditions represent the latest available observations of a service state
	Conditions []metav1.Condition `json:"conditions"`
}
