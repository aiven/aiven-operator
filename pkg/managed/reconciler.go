package managed

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// Reconciler handles the boilerplate reconciliation logic for Aiven resources.
// It orchestrates the ExternalClient lifecycle methods and manages:
// - Finalizers
// - Status conditions
// - Secrets (connection details)
// - Events
// - Requeue logic
type Reconciler[T v1alpha1.AivenManagedObject] struct {
	client.Client
	AivenClient AivenClient[T]
	Log         logr.Logger
	Recorder    record.EventRecorder
	Scheme      *runtime.Scheme
}

// Reconcile performs the full reconciliation loop for a managed resource.
func (r *Reconciler[T]) Reconcile(_ context.Context, _ T) (ctrl.Result, error) {
	// TODO: Implement reconciliation flow:
	//
	// 1. Handle deletion: If the object is being deleted, call the AivenClient's Delete
	//    method and remove the finalizer.
	//
	// 2. Add finalizer: If the object is not being deleted and does not have a finalizer,
	//    add one.
	//
	// 3. Observe the external resource: Call the AivenClient's Observe method to get the
	//    current state of the external resource.
	//
	// 4. Handle preconditions: If the observation indicates that preconditions are not met,
	//    update the status of the object with the reason and requeue.
	//
	// 5. Create or Update: Based on the observation, either create a new external
	//    resource by calling the AivenClient's Create method, or update the existing
	//    one by calling the Update method.
	//
	// 6. Update status: Update the status of the object based on the observation and the
	//    outcome of the create/update operations. This includes setting the Running
	//    condition and other resource-specific status fields.
	//
	// 7. Manage connection secret: If the observation indicates that the resource is ready
	//    and provides connection details, create or update the connection secret.
	//
	// 8. Event recording: Record Kubernetes events.
	//
	// 9. Return result: Return a result that indicates whether the reconciliation should
	//    be requeued and after what time (configurable based on resource spec).

	return ctrl.Result{}, nil
}
