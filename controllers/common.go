// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
)

const (
	conditionTypeRunning     = "Running"
	conditionTypeInitialized = "Initialized"

	secretProtectionFinalizer = "finalizers.aiven.io/needed-to-delete-services"
	instanceDeletionFinalizer = "finalizers.aiven.io/delete-remote-resource"

	processedGenerationAnnotation = "controllers.aiven.io/generation-was-processed"
	instanceIsRunningAnnotation   = "controllers.aiven.io/instance-is-running"

	pollTimeout  = 6 * time.Minute
	pollInterval = 20 * time.Second

	formatIntBaseDecimal = 10
)

func requeueCtrlResult() ctrl.Result {
	// nolint: gomnd
	requeueTimeout := 10 * time.Second

	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: requeueTimeout,
	}
}

func checkServiceIsRunning(c *aiven.Client, project, serviceName string) (bool, error) {
	s, err := c.Services.Get(project, serviceName)
	if err != nil {
		return false, err
	}
	return s.State == "RUNNING", nil
}

func getInitializedCondition(reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionTypeInitialized,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: message,
	}
}

func getRunningCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    conditionTypeRunning,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

func markedForDeletion(o client.Object) bool {
	return !o.GetDeletionTimestamp().IsZero()
}

func addFinalizer(ctx context.Context, client client.Client, o client.Object, f string) error {
	controllerutil.AddFinalizer(o, f)
	return client.Update(ctx, o)
}

func removeFinalizer(ctx context.Context, client client.Client, o client.Object, f string) error {
	controllerutil.RemoveFinalizer(o, f)
	return client.Update(ctx, o)
}

func isAlreadyProcessed(o client.Object) bool {
	return o.GetAnnotations()[processedGenerationAnnotation] == strconv.FormatInt(o.GetGeneration(), formatIntBaseDecimal)
}

func isAlreadyRunning(o client.Object) bool {
	_, found := o.GetAnnotations()[instanceIsRunningAnnotation]
	return found
}

func findTypedResource(ctx context.Context, k8s client.Client, key client.ObjectKey, types ...client.Object) (client.Object, error) {
	for i := range types {
		rtype := types[i]
		if err := k8s.Get(ctx, key, rtype); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		return rtype, nil
	}
	return nil, nil
}

// convenience function to find named services
func findService(ctx context.Context, k8s client.Client, key client.ObjectKey) (client.Object, error) {
	return findTypedResource(ctx, k8s, key, &v1alpha1.PostgreSQL{}, &v1alpha1.Kafka{})
}

// convenience for resource that are owned by a single resource
func findSingleOwner(ctx context.Context, k8s client.Client, key client.ObjectKey, types ...client.Object) ([]client.Object, error) {
	owner, err := findTypedResource(ctx, k8s, key, types...)
	if err != nil {
		return nil, err
	}
	if owner != nil {
		return []client.Object{owner}, nil
	}
	return nil, nil
}
