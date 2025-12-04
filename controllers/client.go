package controllers

import (
	"context"
	"errors"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// AivenController manages the lifecycle of a resource.
// Controllers implement this interface to define how to process their specific resource type.
// Implementations are expected to update status fields and annotations directly on obj.
type AivenController[T v1alpha1.AivenManagedObject] interface {
	// Observe the external resource and return its current state.
	// This method should:
	// - Check if the resource exists on Aiven side
	// - Verify preconditions (e.g., parent service is running)
	// - Determine if the resource is up-to-date with the desired state
	// - Fetch connection details (credentials, etc.)
	// - Update status fields on the object if needed
	//
	// Observe should be idempotent and not modify the external resource.
	Observe(ctx context.Context, obj T) (Observation, error)

	// Create a new resource.
	// This is called when Observe indicates the resource doesn't exist.
	// It may return optional information about the created external resource (for example, connection details).
	Create(ctx context.Context, obj T) (CreateResult, error)

	// Update an existing resource.
	// This is called when Observe indicates the resource exists but is not up-to-date.
	Update(ctx context.Context, obj T) (UpdateResult, error)

	// Delete the resource.
	// This is called when the Kubernetes object is being deleted.
	// If the resource is already deleted (not found), should return nil.
	Delete(ctx context.Context, obj T) error
}

type SecretDetails = map[string]string

// Observation is the result of observing the resource.
// Can be extended with additional fields as needed.
type Observation struct {
	// ResourceExists indicates whether the external resource exists on Aiven side.
	ResourceExists bool

	// ResourceUpToDate indicates whether the external resource matches the desired state.
	// Only meaningful when ResourceExists is true.
	ResourceUpToDate bool

	// SecretDetails contains secret data for the resource (credentials, endpoints, CA certs, etc.).
	// Will be written to the connInfoSecretTarget if not nil and not empty.
	// Keys should NOT include prefixes - the reconciler will apply the appropriate prefix.
	// Example keys: "HOST", "PORT", "USERNAME", "PASSWORD", "CA_CERT"
	SecretDetails SecretDetails
}

// CreateResult is returned from Create and carries optional information about the created external resource (for example, connection details).
type CreateResult = Observation

// UpdateResult is returned from Update and carries optional information about the external resource (for example, connection details).
type UpdateResult = Observation

var errPreconditionNotMet = errors.New("preconditions are not met")
