// Copyright (c) 2025 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"reflect"

	"github.com/aiven/go-client-codegen/handler/service"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// StatusUpdate accumulates all status changes during reconciliation
type StatusUpdate struct {
	conditions   map[string]metav1.Condition
	annotations  map[string]string
	processedGen string
	runningState string // "true", "false", or ""
	hasChanges   bool
	statusFields map[string]any
}

func NewStatusUpdate() *StatusUpdate {
	return &StatusUpdate{
		conditions:   make(map[string]metav1.Condition),
		annotations:  make(map[string]string),
		statusFields: make(map[string]any),
		hasChanges:   false,
	}
}

// SetCondition adds or updates a condition. If a condition with the same Type
// already exists, it will be replaced with the new condition.
func (s *StatusUpdate) SetCondition(condition metav1.Condition) {
	s.conditions[condition.Type] = condition
	s.hasChanges = true
}

// SetAnnotation adds or updates an annotation that will be applied to the resource.
func (s *StatusUpdate) SetAnnotation(key, value string) {
	s.annotations[key] = value
	s.hasChanges = true
}

// RemoveAnnotation marks an annotation for removal. This will be applied during commit.
func (s *StatusUpdate) RemoveAnnotation(key string) {
	s.annotations[key] = ""
	s.hasChanges = true
}

// RemoveForceTriggerAnnotations automatically detects and marks trigger annotations for removal.
// This should be called with the current object to clean up any trigger annotations.
func (s *StatusUpdate) RemoveForceTriggerAnnotations(obj v1alpha1.AivenManagedObject) {
	for k := range obj.GetAnnotations() {
		if k == ForceReconcileAnnotation {
			s.RemoveAnnotation(k)
		}
	}
}

// SetProcessedGeneration sets the processed generation annotation.
func (s *StatusUpdate) SetProcessedGeneration(generation int64) {
	s.processedGen = fmt.Sprintf("%d", generation)
	s.SetAnnotation(processedGenerationAnnotation, s.processedGen)
}

// SetRunningState sets the instance running annotation.
func (s *StatusUpdate) SetRunningState(running bool) {
	if running {
		s.runningState = "true"
	} else {
		s.runningState = "false"
	}
	s.SetAnnotation(instanceIsRunningAnnotation, s.runningState)
}

// SetStatusField sets a status field.
func (s *StatusUpdate) SetStatusField(key string, value interface{}) {
	s.statusFields[key] = value
	s.hasChanges = true
}

// HasChanges returns true if any changes have been made to this StatusUpdate.
func (s *StatusUpdate) HasChanges() bool {
	return s.hasChanges
}

// GetConditions returns a copy of all conditions that have been set.
func (s *StatusUpdate) GetConditions() map[string]metav1.Condition {
	result := make(map[string]metav1.Condition)
	for k, v := range s.conditions {
		result[k] = v
	}

	return result
}

// GetAnnotations returns a copy of all annotations that have been set.
func (s *StatusUpdate) GetAnnotations() map[string]string {
	result := make(map[string]string)
	for k, v := range s.annotations {
		result[k] = v
	}

	return result
}

// GetStatusFields returns a copy of all status fields that have been set.
func (s *StatusUpdate) GetStatusFields() map[string]any {
	result := make(map[string]any)
	for k, v := range s.statusFields {
		result[k] = v
	}

	return result
}

// CommitStatusUpdates applies all accumulated status changes for any AivenManagedObject.
// It handles both status conditions and annotations in a single transaction.
func CommitStatusUpdates[T v1alpha1.AivenManagedObject](
	ctx context.Context,
	c client.Client,
	obj T,
	statusUpdate *StatusUpdate,
) error {
	if statusUpdate == nil || !statusUpdate.HasChanges() {
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// get the latest version to avoid conflicts
		key := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}

		latest := reflect.New(reflect.TypeOf(obj).Elem()).Interface().(T)

		if err := c.Get(ctx, key, latest); err != nil {
			return fmt.Errorf("failed to get latest object for status update: %w", err)
		}

		// clean up trigger annotations
		statusUpdate.RemoveForceTriggerAnnotations(latest)

		// apply condition changes
		for _, condition := range statusUpdate.conditions {
			meta.SetStatusCondition(latest.Conditions(), condition)
		}

		// apply status field changes based on resource type
		// TODO: make this more generic if we have more types with status fields to update
		if len(statusUpdate.statusFields) > 0 {
			switch v := any(latest).(type) { //nolint:gocritic
			case *v1alpha1.Kafka:
				for k, val := range statusUpdate.statusFields {
					switch k { //nolint:gocritic
					case "state":
						if state, ok := val.(service.ServiceStateType); ok {
							v.Status.State = state
						}
					}
				}
			}
		}

		// update status subresource
		if err := c.Status().Update(ctx, latest); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}

		if len(statusUpdate.annotations) > 0 {
			annotationTarget := reflect.New(reflect.TypeOf(obj).Elem()).Interface().(T)
			if err := c.Get(ctx, key, annotationTarget); err != nil {
				return fmt.Errorf("failed to get latest object for annotation update: %w", err)
			}

			original := annotationTarget.DeepCopyObject().(client.Object)
			annotations := annotationTarget.GetAnnotations()
			if annotations == nil {
				annotations = make(map[string]string)
			}

			for k, v := range statusUpdate.annotations {
				if v == "" {
					delete(annotations, k)
				} else {
					annotations[k] = v
				}
			}

			annotationTarget.SetAnnotations(annotations)

			patch := client.MergeFrom(original)
			if err := c.Patch(ctx, annotationTarget, patch); err != nil {
				return fmt.Errorf("failed to update annotations: %w", err)
			}
		}

		return nil
	})
}
