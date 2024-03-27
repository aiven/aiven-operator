// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// KafkaACLReconciler reconciles a KafkaACL object
type KafkaACLReconciler struct {
	Controller
}

func newKafkaACLReconciler(c Controller) reconcilerType {
	return &KafkaACLReconciler{Controller: c}
}

type KafkaACLHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls/status,verbs=get;list;watch;create;delete

func (r *KafkaACLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaACLHandler{}, &v1alpha1.KafkaACL{})
}

func (r *KafkaACLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaACL{}).
		Complete(r)
}

func (h KafkaACLHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	acl, err := h.convert(obj)
	if err != nil {
		return err
	}

	// ACL can't be really modified
	// Tries to delete it instead
	_, err = h.delete(ctx, avn, avnGen, obj)
	if err != nil {
		return err
	}

	// Creates it from scratch
	r, err := avn.KafkaACLs.Create(
		ctx,
		acl.Spec.Project,
		acl.Spec.ServiceName,
		aiven.CreateKafkaACLRequest{
			Permission: acl.Spec.Permission,
			Topic:      acl.Spec.Topic,
			Username:   acl.Spec.Username,
		},
	)
	if err != nil {
		return err
	}

	// New created ACL id set
	acl.Status.ID = r.ID
	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("CreatedOrUpdate",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "CreatedOrUpdate",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(acl.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h KafkaACLHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	id, err := h.getID(ctx, avn, acl)
	if err == nil {
		err = avn.KafkaACLs.Delete(ctx, acl.Spec.Project, acl.Spec.ServiceName, id)
	}

	if err != nil && !isNotFound(err) {
		return false, fmt.Errorf("aiven client delete Kafka ACL error: %w", err)
	}

	return true, nil
}

// todo: remove in v1
// getID returns ACL's ID in < v0.5.1 compatible mode
func (h KafkaACLHandler) getID(ctx context.Context, avn *aiven.Client, acl *v1alpha1.KafkaACL) (string, error) {
	// ACLs made prior to v0.5.1 doesn't have an ID.
	// This block is for fresh made ACLs only
	// The rest of this function tries to guess it filtering the list.
	if acl.Status.ID != "" {
		return acl.Status.ID, nil
	}

	// For old ACLs only
	list, err := avn.KafkaACLs.List(ctx, acl.Spec.Project, acl.Spec.ServiceName)
	if err != nil {
		return "", err
	}

	for _, a := range list {
		if acl.Spec.Topic == a.Topic && acl.Spec.Username == a.Username && acl.Spec.Permission == a.Permission {
			return a.ID, nil
		}
	}

	// Error should mimic client error to play well with isNotFound(err)
	return "", NewNotFound(fmt.Sprintf("Kafka ACL %q not found", acl.Name))
}

func (h KafkaACLHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	id, err := h.getID(ctx, avn, acl)
	if err != nil {
		return nil, err
	}

	_, err = avn.KafkaACLs.Get(ctx, acl.Spec.Project, acl.Spec.ServiceName, id)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h KafkaACLHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsRunning(ctx, avn, avnGen, acl.Spec.Project, acl.Spec.ServiceName)
}

func (h KafkaACLHandler) convert(i client.Object) (*v1alpha1.KafkaACL, error) {
	acl, ok := i.(*v1alpha1.KafkaACL)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaACL")
	}

	return acl, nil
}
