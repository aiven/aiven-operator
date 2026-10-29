package controllers

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const managementPolicyAnnotation = "controllers.aiven.io/management-policy"

// managementPolicy is how far the reconciler drives an existing Aiven resource towards the spec.
type managementPolicy string

const (
	// managementPolicyFull applies every spec change and reverts drift; the default when the annotation is absent.
	managementPolicyFull managementPolicy = "Full"
	// managementPolicyCreateOnly creates a missing resource and keeps the Secret in sync, but never calls Update.
	managementPolicyCreateOnly managementPolicy = "CreateOnly"
)

// allowsUpdate reports whether Update may run under this policy.
func (p managementPolicy) allowsUpdate() bool {
	return p == managementPolicyFull
}

// parseManagementPolicy reads the management-policy annotation; Full when absent, an error for any value other than Full or CreateOnly.
func parseManagementPolicy(obj client.Object) (managementPolicy, error) {
	raw, ok := obj.GetAnnotations()[managementPolicyAnnotation]
	if !ok {
		return managementPolicyFull, nil
	}
	switch p := managementPolicy(raw); p {
	case managementPolicyFull, managementPolicyCreateOnly:
		return p, nil
	}
	return "", fmt.Errorf("invalid management policy %q, allowed values are %q and %q", raw, managementPolicyFull, managementPolicyCreateOnly)
}

// managementPolicySupported marks a controller whose Observe publishes the Secret from the live resource when Update is skipped.
type managementPolicySupported interface {
	managementPolicySupported()
}

// errManagementPolicyNotSupported reports the annotation on a kind that ignores it.
var errManagementPolicyNotSupported = errors.New(managementPolicyAnnotation + " is not supported for this kind yet, spec changes are applied")

// resolveManagementPolicy parses the annotation; errManagementPolicyNotSupported for an annotation on a controller that does not support one.
func resolveManagementPolicy(obj client.Object, controller any) (managementPolicy, error) {
	if _, ok := controller.(managementPolicySupported); !ok {
		if _, set := obj.GetAnnotations()[managementPolicyAnnotation]; set {
			return "", errManagementPolicyNotSupported
		}
		return managementPolicyFull, nil
	}
	return parseManagementPolicy(obj)
}

type managementPolicyContextKey struct{}

// withManagementPolicy hands the policy parsed by the reconciler to the controller methods.
func withManagementPolicy(ctx context.Context, p managementPolicy) context.Context {
	return context.WithValue(ctx, managementPolicyContextKey{}, p)
}

// managementPolicyFrom returns the policy the reconciler parsed for this reconcile, Full when none was stored.
func managementPolicyFrom(ctx context.Context) managementPolicy {
	if p, ok := ctx.Value(managementPolicyContextKey{}).(managementPolicy); ok {
		return p
	}
	return managementPolicyFull
}
