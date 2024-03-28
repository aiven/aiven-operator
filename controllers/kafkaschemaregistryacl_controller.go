// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client/v2"
	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/kafkaschemaregistry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// KafkaSchemaRegistryACLReconciler reconciles a KafkaSchemaRegistryACL object
type KafkaSchemaRegistryACLReconciler struct {
	Controller
}

func newKafkaSchemaRegistryACLReconciler(c Controller) reconcilerType {
	return &KafkaSchemaRegistryACLReconciler{Controller: c}
}

//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemaregistryacls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemaregistryacls/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=kafkaschemaregistryacls/finalizers,verbs=update

func (r *KafkaSchemaRegistryACLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaSchemaRegistryACLHandler{}, &v1alpha1.KafkaSchemaRegistryACL{})
}

// SetupWithManager sets up the controller with the Manager.
func (r *KafkaSchemaRegistryACLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaSchemaRegistryACL{}).
		Complete(r)
}

type KafkaSchemaRegistryACLHandler struct{}

func (h KafkaSchemaRegistryACLHandler) createOrUpdate(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object, refs []client.Object) error {
	acl, err := h.convert(obj)
	if err != nil {
		return err
	}

	exists, err := h.exists(ctx, avnGen, acl)
	if err != nil {
		return err
	}

	if !exists {
		in := kafkaschemaregistry.ServiceSchemaRegistryAclAddIn{
			Permission: kafkaschemaregistry.PermissionType(acl.Spec.Permission),
			Resource:   acl.Spec.Resource,
			Username:   acl.Spec.Username,
		}

		list, err := avnGen.ServiceSchemaRegistryAclAdd(ctx, acl.Spec.Project, acl.Spec.ServiceName, &in)
		if err != nil {
			return fmt.Errorf("cannot create KafkaSchemaRegistryAC on Aiven side: %w", err)
		}

		for _, v := range list {
			if in.Permission == v.Permission && in.Resource == v.Resource && in.Username == v.Username {
				acl.Status.ACLId = v.Id
				break
			}
		}
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("Created",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "Created",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(acl.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h KafkaSchemaRegistryACLHandler) delete(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	_, err = avnGen.ServiceSchemaRegistryAclDelete(ctx, acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ACLId)
	if err != nil && !isNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h KafkaSchemaRegistryACLHandler) exists(ctx context.Context, avnGen avngen.Client, acl *v1alpha1.KafkaSchemaRegistryACL) (bool, error) {
	list, err := avnGen.ServiceSchemaRegistryAclList(ctx, acl.Spec.Project, acl.Spec.ServiceName)
	if err != nil {
		return false, err
	}

	for _, v := range list {
		if v.Id == acl.Status.ACLId {
			return true, nil
		}
	}
	return false, nil
}

func (h KafkaSchemaRegistryACLHandler) get(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	exists, err := h.exists(ctx, avnGen, acl)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, NewNotFound(fmt.Sprintf("KafkaSchemaRegistryACL %q not found", acl.Name))
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h KafkaSchemaRegistryACLHandler) checkPreconditions(ctx context.Context, avn *aiven.Client, avnGen avngen.Client, obj client.Object) (bool, error) {
	acl, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsRunning(ctx, avn, avnGen, acl.Spec.Project, acl.Spec.ServiceName)
}

func (h KafkaSchemaRegistryACLHandler) convert(i client.Object) (*v1alpha1.KafkaSchemaRegistryACL, error) {
	db, ok := i.(*v1alpha1.KafkaSchemaRegistryACL)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaSchemaRegistryACL")
	}

	return db, nil
}
