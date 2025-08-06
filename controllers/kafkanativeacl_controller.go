// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafka"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// KafkaNativeACLReconciler reconciles a KafkaNativeACL object
type KafkaNativeACLReconciler struct {
	Controller
}

func newKafkaNativeACLReconciler(c Controller) reconcilerType {
	return &KafkaNativeACLReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkanativeacls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkanativeacls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkanativeacls/finalizers,verbs=update

func (r *KafkaNativeACLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaNativeACLHandler{}, &v1alpha1.KafkaNativeACL{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaNativeACLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaNativeACL{}).
		Complete(r)
}

type KafkaNativeACLHandler struct{}

// checkPreconditions check whether all preconditions for creating (or updating) the resource are in place.
func (h KafkaNativeACLHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsOperational(ctx, avnGen, acl.Spec.Project, acl.Spec.ServiceName)
}

// createOrUpdate creates or updates an instance on the Aiven side.
func (h KafkaNativeACLHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	acl, err := h.convert(obj)
	if err != nil {
		return err
	}

	if acl.Status.ID != "" {
		// The resource already exists, nothing to do
		return nil
	}

	in := &kafka.ServiceKafkaNativeAclAddIn{
		Host:           &acl.Spec.Host,
		Operation:      acl.Spec.Operation,
		PatternType:    acl.Spec.PatternType,
		PermissionType: acl.Spec.PermissionType,
		Principal:      acl.Spec.Principal,
		ResourceName:   acl.Spec.ResourceName,
		ResourceType:   acl.Spec.ResourceType,
	}

	rsp, err := avnGen.ServiceKafkaNativeAclAdd(ctx, acl.Spec.Project, acl.Spec.ServiceName, in)
	if err != nil {
		return fmt.Errorf("create Kafka-native ACL error: %w", err)
	}

	acl.Status.ID = rsp.Id
	return nil
}

// delete removes an instance on Aiven side.
func (h KafkaNativeACLHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	err = avnGen.ServiceKafkaNativeAclDelete(ctx, acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
	switch {
	case isNotFound(err):
		return true, nil
	case err != nil:
		return false, fmt.Errorf("delete Kafka-native ACL error: %w", err)
	}

	return true, nil
}

// get retrieves an object and a secret.
func (h KafkaNativeACLHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	_, err = avnGen.ServiceKafkaNativeAclGet(ctx, acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h KafkaNativeACLHandler) convert(i client.Object) (*v1alpha1.KafkaNativeACL, error) {
	acl, ok := i.(*v1alpha1.KafkaNativeACL)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaNativeACL")
	}

	return acl, nil
}
