// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// fullSpec sets every owned field.
func fullSpec() *v1alpha1.ServiceCommonSpec {
	return &v1alpha1.ServiceCommonSpec{
		BaseServiceFields: v1alpha1.BaseServiceFields{
			Plan:                  "startup-4",
			CloudName:             "google-europe-west1",
			ProjectVPCID:          new("vpc-1"),
			MaintenanceWindowDow:  service.DowTypeMonday,
			MaintenanceWindowTime: "10:00:00",
			TerminationProtection: new(true),
			Powered:               new(true),
			Tags:                  map[string]string{"env": "test"},
			TechnicalEmails:       []v1alpha1.ServiceTechEmail{{Email: "a@example.com"}, {Email: "b@example.com"}},
		},
		DiskSpace: "100GiB",
	}
}

// minimalSpec has only the required plan; powered is CRD-defaulted, everything else is unmanaged.
func minimalSpec() *v1alpha1.ServiceCommonSpec {
	return &v1alpha1.ServiceCommonSpec{BaseServiceFields: v1alpha1.BaseServiceFields{Plan: "startup-4"}}
}

// desired wraps spec with its projectVpcId as the resolved VPC ID.
func desired(spec *v1alpha1.ServiceCommonSpec) desiredService {
	return desiredService{spec: spec, vpcID: spec.ProjectVPCID}
}

// liveFor is the ServiceGetOut Aiven returns after spec was applied: values as sent, no tags key
// when there are none.
func liveFor(spec *v1alpha1.ServiceCommonSpec) *service.ServiceGetOut {
	s := &service.ServiceGetOut{
		Plan:                  spec.Plan,
		CloudName:             spec.CloudName,
		ProjectVpcId:          fromAnyPointer(spec.ProjectVPCID),
		TerminationProtection: fromAnyPointer(spec.TerminationProtection),
		State:                 service.ServiceStateTypePoweroff,
		DiskSpaceMb:           NilIfZero(v1alpha1.ConvertDiskSpace(spec.DiskSpace)),
	}
	if isPowered(spec) {
		s.State = service.ServiceStateTypeRunning
	}
	if spec.MaintenanceWindowDow != "" || spec.MaintenanceWindowTime != "" {
		s.Maintenance = &service.MaintenanceOut{
			Dow:  service.MaintenanceDowType(spec.MaintenanceWindowDow),
			Time: spec.MaintenanceWindowTime,
		}
	}
	if len(spec.Tags) > 0 {
		s.Tags = maps.Clone(spec.Tags)
	}
	for _, e := range spec.TechnicalEmails {
		s.TechEmails = append(s.TechEmails, service.TechEmailOut(e))
	}
	return s
}

// TestDriftFieldsCoverSpec fails when a field is added to ServiceCommonSpec without deciding
// whether it is compared (a driftField type) or not (listed here with the reason).
func TestDriftFieldsCoverSpec(t *testing.T) {
	t.Parallel()

	excluded := map[string]string{
		"project":                      "identity, immutable",
		"authSecretRef":                "operator-side",
		"connInfoSecretTarget":         "operator-side",
		"connInfoSecretTargetDisabled": "operator-side",
		"projectVPCRef":                "resolved into projectVpcId",
		"serviceIntegrations":          "create-only by contract",
	}

	var specFields []string
	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		for f := range typ.Fields() {
			if f.Anonymous {
				walk(f.Type)
				continue
			}
			specFields = append(specFields, strings.Split(f.Tag.Get("json"), ",")[0])
		}
	}
	walk(reflect.TypeFor[v1alpha1.ServiceCommonSpec]())

	compared := fieldNames(defaultDriftFields())
	for _, name := range specFields {
		_, isExcluded := excluded[name]
		require.True(t, slices.Contains(compared, name) || isExcluded, "spec field %q: add a driftField type or list it as excluded", name)
		require.False(t, slices.Contains(compared, name) && isExcluded, "spec field %q is both compared and excluded", name)
	}
	for _, name := range compared {
		require.Contains(t, specFields, name, "drift field %q is not a spec field", name)
	}
	for name := range excluded {
		require.Contains(t, specFields, name, "excluded field %q is not a spec field", name)
	}
}
