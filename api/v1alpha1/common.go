package v1alpha1

import (
	"github.com/docker/go-units"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuthSecretReference references a Secret containing an Aiven authentication token
type AuthSecretReference struct {
	Name string `json:"name"`
	Key string `json:"key"`
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

	// Service state
	State string `json:"state"`
}

type ServiceCommonSpec struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Format="^[a-zA-Z0-9_-]*$"
	// Target project.
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=128
	// Subscription plan.
	Plan string `json:"plan,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Cloud the service runs in.
	CloudName string `json:"cloudName,omitempty"`

	// +kubebuilder:validation:MaxLength=36
	// Identifier of the VPC the service should be in, if any.
	ProjectVPCID string `json:"projectVpcId,omitempty"`

	// +kubebuilder:validation:Enum=monday;tuesday;wednesday;thursday;friday;saturday;sunday;never
	// Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
	MaintenanceWindowDow string `json:"maintenanceWindowDow,omitempty"`

	// +kubebuilder:validation:MaxLength=8
	// Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
	MaintenanceWindowTime string `json:"maintenanceWindowTime,omitempty"`

	// Prevent service from being deleted. It is recommended to have this enabled for all services.
	TerminationProtection bool `json:"terminationProtection,omitempty"`
}

func ConvertDiscSpace(v string) int {
	diskSizeMB, _ := units.RAMInBytes(v)
	return int(diskSizeMB / units.MiB)
}
