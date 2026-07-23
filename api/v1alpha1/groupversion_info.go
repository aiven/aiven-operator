// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

// Package v1alpha1 contains API Schema definitions for the  v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=aiven.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "aiven.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes registers every kind in this API group with the scheme.
// When adding a new resource type, add its object and list here.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&Clickhouse{}, &ClickhouseList{},
		&ClickhouseDatabase{}, &ClickhouseDatabaseList{},
		&ClickhouseGrant{}, &ClickhouseGrantList{},
		&ClickhouseRole{}, &ClickhouseRoleList{},
		&ClickhouseUser{}, &ClickhouseUserList{},
		&ConnectionPool{}, &ConnectionPoolList{},
		&Database{}, &DatabaseList{},
		&Flink{}, &FlinkList{},
		&Grafana{}, &GrafanaList{},
		&Kafka{}, &KafkaList{},
		&KafkaACL{}, &KafkaACLList{},
		&KafkaConnect{}, &KafkaConnectList{},
		&KafkaConnector{}, &KafkaConnectorList{},
		&KafkaNativeACL{}, &KafkaNativeACLList{},
		&KafkaQuota{}, &KafkaQuotaList{},
		&KafkaSchema{}, &KafkaSchemaList{},
		&KafkaSchemaRegistryACL{}, &KafkaSchemaRegistryACLList{},
		&KafkaTopic{}, &KafkaTopicList{},
		&MySQL{}, &MySQLList{},
		&OpenSearch{}, &OpenSearchList{},
		&OpenSearchACLConfig{}, &OpenSearchACLConfigList{},
		&PostgreSQL{}, &PostgreSQLList{},
		&Project{}, &ProjectList{},
		&ProjectVPC{}, &ProjectVPCList{},
		&ServiceIntegration{}, &ServiceIntegrationList{},
		&ServiceIntegrationEndpoint{}, &ServiceIntegrationEndpointList{},
		&ServiceUser{}, &ServiceUserList{},
		&UpgradePipelineStep{}, &UpgradePipelineStepList{},
		&Valkey{}, &ValkeyList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
