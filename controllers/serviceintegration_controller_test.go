package controllers

import (
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestServiceIntegrationHandler_integrationMatches(t *testing.T) {
	handler := ServiceIntegrationHandler{}

	tests := []struct {
		name      string
		existing  *service.ServiceIntegrationOut
		desired   *v1alpha1.ServiceIntegration
		wantMatch bool
	}{
		{
			name: "exact match",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src",
				SourceProject:   "project-a",
				DestService:     ptr("kafka-dest"),
				DestProject:     "project-b",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:        "kafka_logs",
					SourceServiceName:      "kafka-src",
					SourceProjectName:      "project-a",
					DestinationServiceName: "kafka-dest",
					DestinationProjectName: "project-b",
				},
			},
			wantMatch: true,
		},
		{
			name: "match with endpoints",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "datadog",
				SourceService:   "pg-src",
				SourceProject:   "project-a",
				DestEndpointId:  ptr("endpoint-123"),
				DestProject:     "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:       "datadog",
					SourceServiceName:     "pg-src",
					DestinationEndpointID: "endpoint-123",
				},
			},
			wantMatch: true,
		},
		{
			name: "no match - different integration type",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src",
				SourceProject:   "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:   "metrics",
					SourceServiceName: "kafka-src",
				},
			},
			wantMatch: false,
		},
		{
			name: "no match - different source service",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src-1",
				SourceProject:   "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:   "kafka_logs",
					SourceServiceName: "kafka-src-2",
				},
			},
			wantMatch: false,
		},
		{
			name: "no match - different source project",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src",
				SourceProject:   "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:   "kafka_logs",
					SourceServiceName: "kafka-src",
					SourceProjectName: "project-b",
				},
			},
			wantMatch: false,
		},
		{
			name: "no match - different destination service",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "kafka_logs",
				SourceService:   "kafka-src",
				SourceProject:   "project-a",
				DestService:     ptr("kafka-dest-1"),
				DestProject:     "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:        "kafka_logs",
					SourceServiceName:      "kafka-src",
					DestinationServiceName: "kafka-dest-2",
				},
			},
			wantMatch: false,
		},
		{
			name: "no match - missing endpoint in existing",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "datadog",
				SourceService:   "pg-src",
				SourceProject:   "project-a",
				DestProject:     "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:       "datadog",
					SourceServiceName:     "pg-src",
					DestinationEndpointID: "endpoint-123",
				},
			},
			wantMatch: false,
		},
		{
			name: "match - both have nil destination service",
			existing: &service.ServiceIntegrationOut{
				IntegrationType: "autoscaler",
				SourceService:   "pg-src",
				SourceProject:   "project-a",
				DestService:     nil,
				DestProject:     "project-a",
			},
			desired: &v1alpha1.ServiceIntegration{
				Spec: v1alpha1.ServiceIntegrationSpec{
					ProjectDependant: v1alpha1.ProjectDependant{
						ProjectField: v1alpha1.ProjectField{
							Project: "project-a",
						},
					},
					IntegrationType:        "autoscaler",
					SourceServiceName:      "pg-src",
					DestinationServiceName: "",
				},
			},
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.integrationMatches(tt.existing, tt.desired)
			assert.Equal(t, tt.wantMatch, got)
		})
	}
}

func ptr(s string) *string {
	return &s
}
