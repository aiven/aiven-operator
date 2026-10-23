// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"github.com/aiven/go-client-codegen/handler/service"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// defaultDriftFields lists the ServiceCommonSpec fields compared on every poll.
func defaultDriftFields() []driftField {
	return []driftField{
		planField{},
		cloudField{},
		vpcField{},
		maintenanceDowField{},
		maintenanceTimeField{},
		terminationProtectionField{},
		poweredField{},
		diskSpaceField{},
		tagsField{},
		techEmailsField{},
	}
}

type planField struct{}

func (planField) name() string { return "plan" }
func (planField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	return compareScalar(NilIfZero(d.spec.Plan), l.Plan)
}

type cloudField struct{}

func (cloudField) name() string { return "cloudName" }
func (cloudField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	return compareScalar(NilIfZero(d.spec.CloudName), l.CloudName)
}

type vpcField struct{}

func (vpcField) name() string { return "projectVpcId" }
func (vpcField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	return compareScalar(d.vpcID, l.ProjectVpcId)
}

type maintenanceDowField struct{}

func (maintenanceDowField) name() string { return "maintenanceWindowDow" }
func (maintenanceDowField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	dow, _ := liveMaintenance(l)
	return compareScalar(NilIfZero(string(d.spec.MaintenanceWindowDow)), dow)
}

type maintenanceTimeField struct{}

func (maintenanceTimeField) name() string { return "maintenanceWindowTime" }
func (maintenanceTimeField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	_, time := liveMaintenance(l)
	return compareScalar(NilIfZero(d.spec.MaintenanceWindowTime), time)
}

// liveMaintenance is the live window; a service without one reads as "", "".
func liveMaintenance(l *service.ServiceGetOut) (dow, time string) {
	if l.Maintenance == nil {
		return "", ""
	}
	return string(l.Maintenance.Dow), l.Maintenance.Time
}

type terminationProtectionField struct{}

func (terminationProtectionField) name() string { return "terminationProtection" }
func (terminationProtectionField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	return compareScalar(d.spec.TerminationProtection, l.TerminationProtection)
}

// poweredField: only POWEROFF is off; a transitional state (REBALANCING, REBUILDING) is still powered.
type poweredField struct{}

func (poweredField) name() string { return "powered" }
func (poweredField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	return compareScalar(new(isPowered(d.spec)), l.State != service.ServiceStateTypePoweroff)
}

type diskSpaceField struct{}

func (diskSpaceField) name() string { return "disk_space" }
func (diskSpaceField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	return compareScalar(NilIfZero(v1alpha1.ConvertDiskSpace(d.spec.DiskSpace)), fromAnyPointer(l.DiskSpaceMb))
}

type tagsField struct{}

func (tagsField) name() string { return "tags" }
func (tagsField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	if d.spec.Tags == nil {
		return "", "", false
	}
	spec, live := tagSet(d.spec.Tags), tagSet(l.Tags)
	return spec.String(), live.String(), !spec.Equal(live)
}

type techEmailsField struct{}

func (techEmailsField) name() string { return "technicalEmails" }
func (techEmailsField) compare(d desiredService, l *service.ServiceGetOut) (string, string, bool) {
	if d.spec.TechnicalEmails == nil {
		return "", "", false
	}
	spec, live := newEmailSet(d.spec.TechnicalEmails), newEmailSet(l.TechEmails)
	return spec.String(), live.String(), !spec.Equal(live)
}
