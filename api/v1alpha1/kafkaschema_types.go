// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// KafkaSchemaSpec defines the desired state of KafkaSchema
// +kubebuilder:validation:XValidation:rule="!has(self.references) || size(self.references) == 0 || self.schemaType in ['PROTOBUF', 'JSON']",message="references are only supported for PROTOBUF and JSON schema types"
type KafkaSchemaSpec struct {
	ServiceDependant `json:",inline"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Kafka Schema Subject name
	SubjectName string `json:"subjectName"`

	// Kafka Schema definition. Format depends on schemaType (AVRO/JSON/PROTOBUF)
	Schema string `json:"schema"`

	// +kubebuilder:validation:Enum=AVRO;JSON;PROTOBUF
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Schema type
	SchemaType kafkaschemaregistry.SchemaType `json:"schemaType,omitempty"`

	// +kubebuilder:validation:Enum=BACKWARD;BACKWARD_TRANSITIVE;FORWARD;FORWARD_TRANSITIVE;FULL;FULL_TRANSITIVE;NONE
	// Kafka Schemas compatibility level
	CompatibilityLevel kafkaschemaregistry.CompatibilityType `json:"compatibilityLevel,omitempty"`

	// +kubebuilder:validation:MaxItems=100
	// +listType=map
	// +listMapKey=name
	// Schema references for Protobuf or JSON schemas that import other schemas.
	// References must form a directed acyclic graph (DAG); cycles are not allowed.
	References []SchemaReference `json:"references,omitempty"`
}

// SchemaReference is a reference to another schema in the registry.
// Exactly one of {subject+version} or kafkaSchemaRef must be set.
// +kubebuilder:validation:XValidation:rule="(has(self.subject) && has(self.version) && !has(self.kafkaSchemaRef)) || (!has(self.subject) && !has(self.version) && has(self.kafkaSchemaRef))",message="set both subject and version, or set kafkaSchemaRef, but not both"
type SchemaReference struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	// Name used to reference the schema (e.g., the import path in Protobuf)
	Name string `json:"name"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=512
	// Subject name of the referenced schema in the registry. Mutually exclusive with kafkaSchemaRef.
	// +optional
	Subject string `json:"subject,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// Version of the referenced schema. Mutually exclusive with kafkaSchemaRef.
	// +optional
	Version int `json:"version,omitempty"`

	// Reference to another KafkaSchema resource in the same namespace.
	// Mutually exclusive with subject/version.
	//
	// Cleanup order matters: delete the dependent before the referent.
	// +optional
	KafkaSchemaRef *LocalKafkaSchemaRef `json:"kafkaSchemaRef,omitempty"`
}

// LocalKafkaSchemaRef references another KafkaSchema in the same namespace as the owner.
// Cross-namespace references are not supported to avoid confused-deputy situations in multi-tenant clusters.
type LocalKafkaSchemaRef struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// Name of the KafkaSchema resource in the same namespace.
	Name string `json:"name"`
}

// KafkaSchemaStatus defines the observed state of KafkaSchema
type KafkaSchemaStatus struct {
	// Conditions represent the latest available observations of an KafkaSchema state
	Conditions []metav1.Condition `json:"conditions"`

	// Schema ID
	ID int `json:"id"`

	// Kafka Schema configuration version
	Version int `json:"version"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// KafkaSchema is the Schema for the kafkaschemas API.
//
// Self-references (A -> A) are blocked at admission; transitive cycles
// (A -> B -> A) are not detected at admission time.
//
// Deletion: the operator performs a soft delete followed by a hard delete on
// the subject. The subject disappears from the registry's listing, re-applying a KafkaSchema with the same subjectName
// after deletion starts a brand-new subject at version 1.
// +kubebuilder:printcolumn:name="Service Name",type="string",JSONPath=".spec.serviceName"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Subject",type="string",JSONPath=".spec.subjectName"
// +kubebuilder:printcolumn:name="Compatibility Level",type="string",JSONPath=".spec.compatibilityLevel"
// +kubebuilder:printcolumn:name="Version",type="number",JSONPath=".status.version"
// +kubebuilder:validation:XValidation:rule="!has(self.spec.references) || self.spec.references.all(r, !has(r.kafkaSchemaRef) || r.kafkaSchemaRef.name != self.metadata.name)",message="kafkaSchemaRef cannot point to the KafkaSchema itself"
type KafkaSchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSchemaSpec   `json:"spec,omitempty"`
	Status KafkaSchemaStatus `json:"status,omitempty"`
}

var _ AivenManagedObject = &KafkaSchema{}

func (*KafkaSchema) NoSecret() bool {
	return true
}

func (in *KafkaSchema) AuthSecretRef() *AuthSecretReference {
	return in.Spec.AuthSecretRef
}

func (in *KafkaSchema) Conditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *KafkaSchema) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

// GetRefs returns ResourceReferenceObjects for any kafkaSchemaRef entries in Spec.References.
// The namespace is always the owner's namespace; refs are same-namespace only by design.
func (in *KafkaSchema) GetRefs() []*ResourceReferenceObject {
	refs := make([]*ResourceReferenceObject, 0, len(in.Spec.References))
	for _, ref := range in.Spec.References {
		if ref.KafkaSchemaRef == nil {
			continue
		}
		refs = append(refs, &ResourceReferenceObject{
			GroupVersionKind: GroupVersion.WithKind("KafkaSchema"),
			NamespacedName: types.NamespacedName{
				Namespace: in.GetNamespace(),
				Name:      ref.KafkaSchemaRef.Name,
			},
		})
	}
	return refs
}

// +kubebuilder:object:root=true

// KafkaSchemaList contains a list of KafkaSchema
type KafkaSchemaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaSchema `json:"items"`
}
