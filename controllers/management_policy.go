package controllers

import (
	"context"
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

// parseManagementPolicy reads the management-policy annotation; Full when absent, an error for any value other than CreateOnly.
func parseManagementPolicy(obj client.Object) (managementPolicy, error) {
	raw, ok := obj.GetAnnotations()[managementPolicyAnnotation]
	if !ok {
		return managementPolicyFull, nil
	}
	if p := managementPolicy(raw); p == managementPolicyCreateOnly {
		return p, nil
	}
	return "", fmt.Errorf("invalid management policy %q, only %q is allowed", raw, managementPolicyCreateOnly)
}

// managementPolicyUnsupported marks a controller that cannot observe the live resource, so no policy applies to its kind.
type managementPolicyUnsupported interface {
	managementPolicyUnsupported()
}

// resolveManagementPolicy parses the annotation and rejects a non-default policy on a controller that does not support one.
func resolveManagementPolicy(obj client.Object, controller any) (managementPolicy, error) {
	policy, err := parseManagementPolicy(obj)
	if err != nil {
		return "", err
	}
	if _, ok := controller.(managementPolicyUnsupported); ok && policy != managementPolicyFull {
		return "", fmt.Errorf("%s is not supported for this kind", managementPolicyAnnotation)
	}
	return policy, nil
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
