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

// KafkaACLReconciler reconciles a KafkaACL object
type KafkaACLReconciler struct {
	Controller
}

func newKafkaACLReconciler(c Controller) reconcilerType {
	return &KafkaACLReconciler{Controller: c}
}

type KafkaACLHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaacls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaacls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaacls/finalizers,verbs=get;create;update

func (r *KafkaACLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaACLHandler{}, &v1alpha1.KafkaACL{})
}

func (r *KafkaACLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaACL{}).
		Complete(r)
}

func (h KafkaACLHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) (bool, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// ACL can't be really modified
	// Tries to delete it instead
	_, err = h.delete(ctx, avnGen, obj)
	if err != nil {
		return false, err
	}

	// Creates it from scratch
	_, err = avnGen.ServiceKafkaAclAdd(
		ctx,
		acl.Spec.Project,
		acl.Spec.ServiceName,
		&kafka.ServiceKafkaAclAddIn{
			Permission: acl.Spec.Permission,
			Topic:      acl.Spec.Topic,
			Username:   acl.Spec.Username,
		},
	)
	if err != nil {
		return false, err
	}

	// Resets the old ID in case it was set
	acl.Status.ID = ""

	// Gets the ID of the newly created ACL
	// The server doesn't return the ACL we created, but the list of all ACLs currently defined.
	// Need to find the correct one manually.
	acl.Status.ID, err = h.getID(ctx, avnGen, acl)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (h KafkaACLHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	id, err := h.getID(ctx, avnGen, acl)
	if err == nil {
		_, err = avnGen.ServiceKafkaAclDelete(ctx, acl.Spec.Project, acl.Spec.ServiceName, id)
	}

	if err != nil && !isNotFound(err) {
		return false, fmt.Errorf("aiven client delete Kafka ACL error: %w", err)
	}

	return true, nil
}

// todo: remove in v1
// getID returns ACL's ID in < v0.5.1 compatible mode
func (h KafkaACLHandler) getID(ctx context.Context, avnGen avngen.Client, acl *v1alpha1.KafkaACL) (string, error) {
	// ACLs made prior to v0.5.1 doesn't have an ID.
	// This block is for fresh made ACLs only
	// The rest of this function tries to guess it filtering the list.
	if acl.Status.ID != "" {
		return acl.Status.ID, nil
	}

	// For old ACLs only
	list, err := avnGen.ServiceKafkaAclList(ctx, acl.Spec.Project, acl.Spec.ServiceName)
	if err != nil {
		return "", err
	}

	// There could be multiple ACLs with same attributes.
	// Assume the one that was created is the last one matching.
	var latestID string
	for _, a := range list {
		if acl.Spec.Topic == a.Topic && acl.Spec.Username == a.Username && acl.Spec.Permission == a.Permission {
			latestID = fromAnyPointer(a.Id)
		}
	}

	if latestID != "" {
		return latestID, nil
	}

	// Error should mimic client error to play well with isNotFound(err)
	return "", NewNotFound(fmt.Sprintf("Kafka ACL %q not found", acl.Name))
}

func (h KafkaACLHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	_, err = h.getID(ctx, avnGen, acl)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h KafkaACLHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsOperational(ctx, avnGen, acl.Spec.Project, acl.Spec.ServiceName)
}

func (h KafkaACLHandler) convert(i client.Object) (*v1alpha1.KafkaACL, error) {
	acl, ok := i.(*v1alpha1.KafkaACL)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaACL")
	}

	return acl, nil
}
