package v1alpha1

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/go-units"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrDeleteDependencies = errors.New("object has dependencies and cannot be deleted")

// AuthSecretReference references a Secret containing an Aiven authentication token
type AuthSecretReference struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
}

// ConnInfoSecretTarget contains information secret name
type ConnInfoSecretTarget struct {
	// Name of the secret resource to be created. By default, is equal to the resource name
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
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Target project.
	Project string `json:"project"`

	// +kubebuilder:validation:MaxLength=128
	// Subscription plan.
	Plan string `json:"plan,omitempty"`

	// +kubebuilder:validation:MaxLength=256
	// Cloud the service runs in.
	CloudName string `json:"cloudName,omitempty"`

	// +kubebuilder:validation:MaxLength=36
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Identifier of the VPC the service should be in, if any.
	ProjectVPCID string `json:"projectVpcId,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically
	ProjectVPCRef *ResourceReference `json:"projectVPCRef,omitempty"`

	// +kubebuilder:validation:Enum=monday;tuesday;wednesday;thursday;friday;saturday;sunday
	// Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
	MaintenanceWindowDow string `json:"maintenanceWindowDow,omitempty"`

	// +kubebuilder:validation:MaxLength=8
	// Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
	MaintenanceWindowTime string `json:"maintenanceWindowTime,omitempty"`

	// Prevent service from being deleted. It is recommended to have this enabled for all services.
	TerminationProtection bool `json:"terminationProtection,omitempty"`

	// Tags are key-value pairs that allow you to categorize services.
	Tags map[string]string `json:"tags,omitempty"`

	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Service integrations to specify when creating a service. Not applied after initial service creation
	ServiceIntegrations []*ServiceIntegrationItem `json:"serviceIntegrations,omitempty"`
}

// Validate runs complex validation on ServiceCommonSpec
func (in *ServiceCommonSpec) Validate() error {
	// todo: remove when resolved https://github.com/kubernetes-sigs/controller-tools/issues/461
	if in.ProjectVPCID != "" && in.ProjectVPCRef != nil {
		return fmt.Errorf("please set ProjectVPCID or ProjectVPCRef, not both")
	}
	return nil
}

// GetRefs is inherited by kafka, pg, os, etc
func (in *ServiceCommonSpec) GetRefs(namespace string) (refs []*ResourceReferenceObject) {
	if in.ProjectVPCRef != nil {
		refs = append(refs, in.ProjectVPCRef.ProjectVPC(namespace))
	}
	return refs
}

// ResourceReference is a generic reference to another resource.
// Resource referring to another (dependency) won't start reconciliation until
// dependency is not ready
type ResourceReference struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace,omitempty"`
}

func (in *ResourceReference) ref(kind string, objNamespace string) *ResourceReferenceObject {
	ns := in.Namespace
	if !strings.Contains(in.Name, string(types.Separator)) {
		// If no namespace is provided and in.Name doesn't contain it already
		// sets to default namespace
		if ns == "" {
			ns = objNamespace
		}

		// falls to default
		if ns == "" {
			ns = "default"
		}
	}

	gvk := GroupVersion.WithKind(kind)
	return &ResourceReferenceObject{
		GroupVersionKind: gvk,
		NamespacedName: types.NamespacedName{
			Namespace: ns,
			Name:      in.Name,
		},
	}
}

// ProjectVPC returns reference ProjectVPC kind
func (in *ResourceReference) ProjectVPC(objNamespace string) *ResourceReferenceObject {
	return in.ref("ProjectVPC", objNamespace)
}

// ResourceReferenceObject is a composite "key" to resource
// GroupVersionKind is for resource "type": GroupVersionKind{Group: "aiven.io", Version: "v1alpha1", Kind: "Kafka"}
// NamespacedName is for specific instance: NamespacedName{Name: "my-kafka", Namespace: "default"}
type ResourceReferenceObject struct {
	GroupVersionKind schema.GroupVersionKind
	NamespacedName   types.NamespacedName
}

func ConvertDiscSpace(v string) int {
	if v == "" {
		return 0
	}
	diskSizeMB, _ := units.RAMInBytes(v)
	return int(diskSizeMB / units.MiB)
}

// FindProjectVPC returns ProjectVPC from reference list
func FindProjectVPC(refs []client.Object) *ProjectVPC {
	for _, o := range refs {
		if p, ok := o.(*ProjectVPC); ok {
			return p
		}
	}
	return nil
}

// ErrorSubstrChecker returns error checker for containing given substrings
func ErrorSubstrChecker(substrings ...string) func(error) bool {
	return func(err error) bool {
		if err == nil {
			return false
		}

		errStr := err.Error()
		for _, s := range substrings {
			if strings.Contains(errStr, s) {
				return true
			}
		}
		return false
	}
}

// Service integrations to specify when creating a service. Not applied after initial service creation
type ServiceIntegrationItem struct {
	// +kubebuilder:validation:Enum=read_replica
	IntegrationType string `json:"integrationType"`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	SourceServiceName string `json:"sourceServiceName"`
}
