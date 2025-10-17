// Copyright (c) 2025 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ConditionFunc a function that checks if reconciliation may be skipped
type ConditionFunc func(v1alpha1.AivenManagedObject) (bool, string, error)

// ConditionEvaluator provides simple composition of condition checks
type ConditionEvaluator struct {
	conditions []ConditionFunc
}

// NewConditionEvaluator creates an evaluator with custom conditions
func NewConditionEvaluator(conditions ...ConditionFunc) *ConditionEvaluator {
	return &ConditionEvaluator{
		conditions: conditions,
	}
}

// ShouldSkip evaluates all conditions and determines if reconciliation can be skipped
// Only skip if ALL conditions are satisfied.
func (e *ConditionEvaluator) ShouldSkip(obj v1alpha1.AivenManagedObject) (bool, string, error) {
	for _, check := range e.conditions {
		shouldReconcile, reason, err := check(obj)
		if err != nil {
			return false, "condition evaluation failed: " + reason, err
		}

		if shouldReconcile {
			return false, reason, nil
		}
	}

	return true, "all conditions satisfied", nil
}

// CheckGenerationChanged checks if the resource generation has changed
func CheckGenerationChanged(obj v1alpha1.AivenManagedObject) (bool, string, error) {
	if !hasLatestGeneration(obj) {
		return true, "resource generation changed", nil
	}

	return false, "generation up to date", nil
}

// CheckRunningStatus checks if the resource is marked as running via status conditions
// This should only suggest skipping reconciliation if the service is running AND the current generation has been processed
func CheckRunningStatus(obj v1alpha1.AivenManagedObject) (bool, string, error) {
	runningCondition := meta.FindStatusCondition(*obj.Conditions(), conditionTypeRunning)

	if runningCondition == nil {
		return true, "Running condition not found", nil
	}

	if runningCondition.Status != metav1.ConditionTrue {
		return true, fmt.Sprintf("Running condition not true (status=%s, reason=%s)",
			runningCondition.Status, runningCondition.Reason), nil
	}

	if !hasLatestGeneration(obj) {
		return true, "Running but generation changed, need to reconcile", nil
	}

	return false, fmt.Sprintf("resource is running and generation current (reason=%s)", runningCondition.Reason), nil
}

// CheckTriggerAnnotations checks for force reconciliation trigger annotations.
func CheckTriggerAnnotations(obj v1alpha1.AivenManagedObject) (bool, string, error) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false, "no trigger annotations", nil
	}

	if _, exists := annotations[ForceReconcileAnnotation]; exists {
		return true, "trigger annotation: " + ForceReconcileAnnotation, nil
	}

	return false, "no trigger annotations found", nil
}

// WithCondition is a fluent builder for adding conditions
func (e *ConditionEvaluator) WithCondition(condition ConditionFunc) *ConditionEvaluator {
	e.conditions = append(e.conditions, condition)
	return e
}
