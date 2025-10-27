package managed

import (
	"context"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// AivenClient manages the lifecycle of a resource.
// Controllers implement this interface to define how to process their specific resource type.
type AivenClient[T v1alpha1.AivenManagedObject] interface {
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
	Create(ctx context.Context, obj T) error

	// Update an existing resource.
	// This is called when Observe indicates the resource exists but is not up-to-date.
	Update(ctx context.Context, obj T) error

	// Delete the resource.
	// This is called when the Kubernetes object is being deleted.
	// If the resource is already deleted (not found), should return nil.
	Delete(ctx context.Context, obj T) error
}

// Observation is the result of observing the resource.
// Can be extended with additional fields as needed.
type Observation struct {
	// ResourceExists indicates whether the external resource exists on Aiven side.
	ResourceExists bool

	// ResourceUpToDate indicates whether the external resource matches the desired state.
	// Only meaningful when ResourceExists is true.
	ResourceUpToDate bool

	// ResourceReady indicates whether the resource is in a ready/running state.
	// This is used to determine when to create the connection secret and mark the resource as ready.
	ResourceReady bool

	// IsPoweredOff indicates whether the resource is currently powered off.
	IsPoweredOff bool

	// PreconditionsMet indicates whether all preconditions for the resource are satisfied.
	PreconditionsMet bool

	// PreconditionError contains the error if preconditions are not met.
	// This error will be set in status conditions.
	// Should be nil if PreconditionsMet is true.
	PreconditionError error

	// SecretDetails contains secret data for the resource (credentials, endpoints, CA certs, etc.).
	// Will be written to the connInfoSecretTarget if not nil and not empty.
	// Keys should NOT include prefixes - the reconciler will apply the appropriate prefix.
	// Example keys: "HOST", "PORT", "USERNAME", "PASSWORD", "CA_CERT"
	SecretDetails map[string][]byte

	// Metadata contains additional observed information that should be stored in the status.
	// The reconciler may update status fields based on this data.
	Metadata map[string]string
}
