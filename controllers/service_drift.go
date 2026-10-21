// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/aiven/go-client-codegen/handler/service"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// desiredService is what the live service is compared against.
type desiredService struct {
	spec *v1alpha1.ServiceCommonSpec
	// vpcID is the Aiven VPC ID; the spec can name it through projectVPCRef.
	// "" when the spec names no VPC.
	vpcID string
}

// driftField compares one ServiceCommonSpec field with the live service. A field left out of
// the spec is unmanaged and never drifts.
type driftField interface {
	// name is the field's JSON name in the spec.
	name() string
	compare(d desiredService, live *service.ServiceGetOut) (specValue, liveValue string, drifted bool)
}

// fieldDrift is a field whose live value differs from the spec.
type fieldDrift struct{ name, spec, live string }

// String formats the drift for the DriftDetected event.
func (f fieldDrift) String() string {
	return fmt.Sprintf("%s: %s (live) -> %s (spec)", f.name, f.live, f.spec)
}

// detectDrift returns the fields whose live value differs from the desired service, in table order.
func detectDrift(fields []driftField, d desiredService, live *service.ServiceGetOut) []fieldDrift {
	var drifts []fieldDrift
	for _, f := range fields {
		if specValue, liveValue, drifted := f.compare(d, live); drifted {
			drifts = append(drifts, fieldDrift{name: f.name(), spec: specValue, live: liveValue})
		}
	}
	return drifts
}

// compareScalar reports drift for a plain comparable value; a nil spec value is an omitted field.
func compareScalar[V comparable](spec *V, live V) (specValue, liveValue string, drifted bool) {
	if spec == nil {
		return "", "", false
	}
	return fmt.Sprint(*spec), fmt.Sprint(live), *spec != live
}

// tagSet is a service's tags; a service without tags has no tags key, so nil equals {}.
type tagSet map[string]string

func (t tagSet) Equal(o tagSet) bool { return maps.Equal(t, o) }

// String formats the tags as "{k=v, ...}" in key order.
func (t tagSet) String() string {
	parts := make([]string, 0, len(t))
	for _, k := range slices.Sorted(maps.Keys(t)) {
		parts = append(parts, k+"="+t[k])
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// emailSet is a service's technical emails as a sorted set; Aiven keeps the case but not the order.
type emailSet []string

// newEmailSet accepts the spec's and the API's email structs, which differ only in name.
func newEmailSet[E v1alpha1.ServiceTechEmail | service.TechEmailOut](emails []E) emailSet {
	addresses := make([]string, 0, len(emails))
	for _, e := range emails {
		addresses = append(addresses, service.TechEmailOut(e).Email)
	}
	return slices.Compact(slices.Sorted(slices.Values(addresses)))
}

func (e emailSet) Equal(o emailSet) bool { return slices.Equal(e, o) }

// String formats the emails as "[a, b]".
func (e emailSet) String() string { return "[" + strings.Join(e, ", ") + "]" }
