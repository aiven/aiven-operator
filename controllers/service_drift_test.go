// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestDetectDrift(t *testing.T) {
	t.Parallel()

	fields := defaultDriftFields()
	require.Equal(t, defaultDriftFieldNames, fieldNames(fields))

	t.Run("round trip: a spec applied and read back reports nothing", func(t *testing.T) {
		t.Parallel()
		spec := fullSpec()
		require.Empty(t, detectDrift(fields, desired(spec), liveFor(spec)))

		spec.Powered = new(false)
		spec.Tags = map[string]string{}
		spec.TechnicalEmails = []v1alpha1.ServiceTechEmail{}
		require.Empty(t, detectDrift(fields, desired(spec), liveFor(spec)))
	})

	t.Run("omitted fields never drift", func(t *testing.T) {
		t.Parallel()
		live := liveFor(fullSpec())
		live.Tags = map[string]string{"console": "yes"}
		live.State = service.ServiceStateTypePoweroff
		require.Equal(t, []fieldDrift{{name: "powered", spec: "true", live: "false"}}, detectDrift(fields, desired(minimalSpec()), live))
	})

	t.Run("spec values in API terms", func(t *testing.T) {
		t.Parallel()
		spec := fullSpec()
		live := liveFor(spec)

		spec.DiskSpace = "80G"
		require.Equal(t, []fieldDrift{{"disk_space", "81920", "102400"}}, detectDrift(fields, desired(spec), live))
		spec.DiskSpace = "lots"
		require.Empty(t, detectDrift(fields, desired(spec), live), "an unparsable size is 0, so omitted")

		spec.ProjectVPCID = nil
		require.Equal(t, []fieldDrift{{"projectVpcId", "vpc-2", "vpc-1"}},
			detectDrift(fields, desiredService{spec: spec, vpcID: new("vpc-2")}, live), "the resolved vpcID is compared, not the spec field")
	})

	t.Run("projectVpcId tri-state", func(t *testing.T) {
		t.Parallel()
		spec := fullSpec()
		live := liveFor(spec) // in vpc-1

		spec.ProjectVPCID = nil
		require.Empty(t, detectDrift(fields, desired(spec), live), "omitted: the live VPC is unmanaged")
		spec.ProjectVPCID = new("")
		require.Equal(t, []fieldDrift{{"projectVpcId", "", "vpc-1"}}, detectDrift(fields, desired(spec), live), `"" asks for no VPC`)
		live.ProjectVpcId = ""
		require.Empty(t, detectDrift(fields, desired(spec), live))
	})

	// Aiven stores values as sent, so only absence is normalized.
	t.Run("shape normalizers", func(t *testing.T) {
		t.Parallel()

		t.Run("email order", func(t *testing.T) {
			spec := fullSpec()
			live := liveFor(spec)
			live.TechEmails = []service.TechEmailOut{{Email: "b@example.com"}, {Email: "a@example.com"}}
			require.Empty(t, detectDrift(fields, desired(spec), live))
			live.TechEmails[0].Email = "B@example.com"
			require.Len(t, detectDrift(fields, desired(spec), live), 1, "case is kept by Aiven, so it is compared")
		})

		t.Run("tags nil equals empty map", func(t *testing.T) {
			spec := fullSpec()
			spec.Tags = map[string]string{}
			live := liveFor(spec)
			require.Nil(t, live.Tags)
			require.Empty(t, detectDrift(fields, desired(spec), live))
			live.Tags = map[string]string{}
			require.Empty(t, detectDrift(fields, desired(spec), live))
		})

		t.Run("powered: only POWEROFF is off", func(t *testing.T) {
			spec := fullSpec()
			live := liveFor(spec)
			live.State = service.ServiceStateTypeRebalancing
			require.Empty(t, detectDrift(fields, desired(spec), live))
			spec.Powered = new(false)
			require.Equal(t, []fieldDrift{{"powered", "false", "true"}}, detectDrift(fields, desired(spec), live))
		})

		t.Run("disk nil is zero", func(t *testing.T) {
			spec := fullSpec()
			live := liveFor(spec)
			live.DiskSpaceMb = nil
			require.Equal(t, []fieldDrift{{"disk_space", "102400", "0"}}, detectDrift(fields, desired(spec), live))
		})

		t.Run("values Aiven rejects on apply are plain drift", func(t *testing.T) {
			spec := fullSpec()
			live := liveFor(spec)
			live.CloudName = "Google-Europe-West1"
			live.Maintenance.Time = "10:00"
			require.Equal(t, []fieldDrift{
				{"cloudName", "google-europe-west1", "Google-Europe-West1"},
				{"maintenanceWindowTime", "10:00:00", "10:00"},
			}, detectDrift(fields, desired(spec), live))
		})
	})

	t.Run("drift per field", func(t *testing.T) {
		t.Parallel()
		spec := fullSpec()
		for _, tc := range []struct {
			mutate func(*service.ServiceGetOut)
			drift  fieldDrift
		}{
			{func(s *service.ServiceGetOut) { s.Plan = "business-4" }, fieldDrift{"plan", "startup-4", "business-4"}},
			{func(s *service.ServiceGetOut) { s.CloudName = "aws-eu-west-1" }, fieldDrift{"cloudName", "google-europe-west1", "aws-eu-west-1"}},
			{func(s *service.ServiceGetOut) { s.ProjectVpcId = "" }, fieldDrift{"projectVpcId", "vpc-1", ""}},
			{func(s *service.ServiceGetOut) { s.Maintenance.Dow = service.MaintenanceDowTypeNever }, fieldDrift{"maintenanceWindowDow", "monday", "never"}},
			{func(s *service.ServiceGetOut) { s.Maintenance.Time = "11:30:00" }, fieldDrift{"maintenanceWindowTime", "10:00:00", "11:30:00"}},
			{func(s *service.ServiceGetOut) { s.TerminationProtection = false }, fieldDrift{"terminationProtection", "true", "false"}},
			{func(s *service.ServiceGetOut) { s.State = service.ServiceStateTypePoweroff }, fieldDrift{"powered", "true", "false"}},
			{func(s *service.ServiceGetOut) { s.DiskSpaceMb = new(204800) }, fieldDrift{"disk_space", "102400", "204800"}},
			{func(s *service.ServiceGetOut) { s.Tags["env"] = "prod" }, fieldDrift{"tags", "{env=test}", "{env=prod}"}},
			{func(s *service.ServiceGetOut) { s.Tags = nil }, fieldDrift{"tags", "{env=test}", "{}"}},
			{func(s *service.ServiceGetOut) { s.TechEmails = s.TechEmails[:1] }, fieldDrift{"technicalEmails", "[a@example.com, b@example.com]", "[a@example.com]"}},
			{func(s *service.ServiceGetOut) { s.TechEmails = nil }, fieldDrift{"technicalEmails", "[a@example.com, b@example.com]", "[]"}},
		} {
			t.Run(tc.drift.name+" "+tc.drift.live, func(t *testing.T) {
				live := liveFor(spec)
				tc.mutate(live)
				require.Equal(t, []fieldDrift{tc.drift}, detectDrift(fields, desired(spec), live))
			})
		}

		t.Run("maintenance missing on the live side", func(t *testing.T) {
			live := liveFor(spec)
			live.Maintenance = nil
			require.Equal(t, []fieldDrift{
				{"maintenanceWindowDow", "monday", ""},
				{"maintenanceWindowTime", "10:00:00", ""},
			}, detectDrift(fields, desired(spec), live))
		})
	})

	t.Run("message format", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "plan: business-8 (live) -> business-4 (spec)",
			fieldDrift{name: "plan", spec: "business-4", live: "business-8"}.String())
	})
}
