package v1alpha1

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aiven/go-client-codegen/handler/service"
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
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Name of the secret resource to be created. By default, it is equal to the resource name
	Name string `json:"name"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// Annotations added to the secret
	Annotations map[string]string `json:"annotations,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// Labels added to the secret
	Labels map[string]string `json:"labels,omitempty"`
	// Prefix for the secret's keys.
	// Added "as is" without any transformations.
	// By default, is equal to the kind name in uppercase + underscore, e.g. `KAFKA_`, `REDIS_`, etc.
	Prefix string `json:"prefix,omitempty"`
}

// ConnInfoSecretSource contains information about existing secret to read connection parameters from.
// IMPORTANT: The source secret is not watched for changes. If you update the password in the source secret,
// you must trigger a reconciliation (e.g., by adding an annotation) to apply the new password.
type ConnInfoSecretSource struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// Name of the secret resource to read connection parameters from
	Name string `json:"name"`
	// Namespace of the source secret. If not specified, defaults to the same namespace as the resource
	Namespace string `json:"namespace,omitempty"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// Key in the secret containing the password to use for authentication
	PasswordKey string `json:"passwordKey"`
}

// ServiceStatus defines the observed state of service
type ServiceStatus struct {
	// Conditions represent the latest available observations of a service state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Service state
	State service.ServiceStateType `json:"state,omitempty"`
}

type ServiceTechEmail struct {
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
	// Email address.
	Email string `json:"email"`
}

type ProjectField struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-zA-Z0-9_-]+$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Identifies the project this resource belongs to
	Project string `json:"project"`
}

type ServiceField struct {
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern="^[a-z][-a-z0-9]+$"
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Specifies the name of the service that this resource belongs to
	ServiceName string `json:"serviceName"`
}

type AuthSecretRefField struct {
	// Authentication reference to Aiven token in a secret
	AuthSecretRef *AuthSecretReference `json:"authSecretRef,omitempty"`
}

type ProjectDependant struct {
	ProjectField       `json:",inline"`
	AuthSecretRefField `json:",inline"`
}

type ServiceDependant struct {
	ProjectDependant `json:",inline"`
	ServiceField     `json:",inline"`
}

type BaseServiceFields struct {
	ProjectDependant `json:",inline"`

	// +kubebuilder:validation:MaxLength=128
	// Subscription plan.
	Plan string `json:"plan"`

	// +kubebuilder:validation:MaxLength=256
	// Cloud the service runs in.
	CloudName string `json:"cloudName,omitempty"`

	// +kubebuilder:validation:MaxLength=36
	// Identifier of the VPC the service should be in, if any.
	ProjectVPCID string `json:"projectVpcId,omitempty"`

	// ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically
	ProjectVPCRef *ResourceReference `json:"projectVPCRef,omitempty"`

	// +kubebuilder:validation:Enum=monday;tuesday;wednesday;thursday;friday;saturday;sunday
	// Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
	MaintenanceWindowDow service.DowType `json:"maintenanceWindowDow,omitempty"`

	// +kubebuilder:validation:MaxLength=8
	// Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
	MaintenanceWindowTime string `json:"maintenanceWindowTime,omitempty"`

	// Prevent service from being deleted. It is recommended to have this enabled for all services.
	TerminationProtection *bool `json:"terminationProtection,omitempty"`

	// Tags are key-value pairs that allow you to categorize services.
	Tags map[string]string `json:"tags,omitempty"`

	// +kubebuilder:validation:MaxItems=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Service integrations to specify when creating a service. Not applied after initial service creation
	ServiceIntegrations []*ServiceIntegrationItem `json:"serviceIntegrations,omitempty"`

	// +kubebuilder:validation:MaxItems=10
	// Defines the email addresses that will receive alerts about upcoming maintenance updates or warnings about service instability.
	TechnicalEmails []ServiceTechEmail `json:"technicalEmails,omitempty"`

	// +kubebuilder:default=true
	// Determines the power state of the service. When `true` (default), the service is running.
	// When `false`, the service is powered off.
	// For more information please see [Aiven documentation](https://aiven.io/docs/platform/concepts/service-power-cycle).
	// Note that:
	// - When set to `false` the annotation `controllers.aiven.io/instance-is-running` is also set to `false`.
	// - Services cannot be created in a powered off state. The value is ignored during creation.
	// - It is highly recommended to not run dependent resources when the service is powered off.
	//   Creating a new resource or updating an existing resource that depends on a powered off service will result in an error.
	//   Existing resources will need to be manually recreated after the service is powered on.
	// - Existing secrets will not be updated or removed when the service is powered off.
	// - For Kafka services with backups: Topic configuration, schemas and connectors are all backed up, but not the data in topics. All topic data is lost on power off.
	// - For Kafka services without backups: Topic configurations including all topic data is lost on power off.
	Powered *bool `json:"powered,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(oldSelf.connInfoSecretTargetDisabled) == has(self.connInfoSecretTargetDisabled)",message="connInfoSecretTargetDisabled can only be set during resource creation."
type SecretFields struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="connInfoSecretTargetDisabled is immutable."
	// When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
	ConnInfoSecretTargetDisabled *bool `json:"connInfoSecretTargetDisabled,omitempty"`

	// Secret configuration.
	ConnInfoSecretTarget ConnInfoSecretTarget `json:"connInfoSecretTarget,omitempty"`
}

// Validate runs complex validation on ServiceCommonSpec
func (in *BaseServiceFields) Validate() error {
	// todo: remove when resolved https://github.com/kubernetes-sigs/controller-tools/issues/461
	if in.ProjectVPCID != "" && in.ProjectVPCRef != nil {
		return fmt.Errorf("please set ProjectVPCID or ProjectVPCRef, not both")
	}
	return nil
}

// GetRefs is inherited by kafka, pg, os, etc
func (in *BaseServiceFields) GetRefs(namespace string) (refs []*ResourceReferenceObject) {
	if in.ProjectVPCRef != nil {
		refs = append(refs, in.ProjectVPCRef.ProjectVPC(namespace))
	}
	return refs
}

type ServiceCommonSpec struct {
	BaseServiceFields `json:",inline"`
	SecretFields      `json:",inline"`

	// +kubebuilder:validation:Pattern="(?i)^[1-9][0-9]*(GiB|G)?$"
	// The disk space of the service, possible values depend on the service type, the cloud provider and the project.
	// Reducing will result in the service re-balancing.
	// The removal of this field does not change the value.
	DiskSpace string `json:"disk_space,omitempty"`
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

func ConvertDiskSpace(v string) int {
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
	IntegrationType service.IntegrationType `json:"integrationType"`
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	SourceServiceName string `json:"sourceServiceName"`
}

// +k8s:deepcopy-gen=false
type AivenManagedObject interface {
	client.Object

	AuthSecretRef() *AuthSecretReference
	Conditions() *[]metav1.Condition
	GetObjectMeta() *metav1.ObjectMeta
	NoSecret() bool
}
