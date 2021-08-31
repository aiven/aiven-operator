// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
)

// KafkaACLReconciler reconciles a KafkaACL object
type KafkaACLReconciler struct {
	Controller
}

type KafkaACLHandler struct {
	k8s client.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls/status,verbs=get

func (r *KafkaACLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, KafkaACLHandler{r.Client}, &v1alpha1.KafkaACL{})
}

func (r *KafkaACLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.KafkaACL{}).
		Complete(r)
}

func (h KafkaACLHandler) createOrUpdate(avn *aiven.Client, i client.Object) error {
	acl, err := h.convert(i)
	if err != nil {
		return err
	}

	exists, err := h.exists(avn, acl)
	if err != nil {
		return err
	}

	if !exists {
		_, err = avn.KafkaACLs.Create(
			acl.Spec.Project,
			acl.Spec.ServiceName,
			aiven.CreateKafkaACLRequest{
				Permission: acl.Spec.Permission,
				Topic:      acl.Spec.Topic,
				Username:   acl.Spec.Username,
			},
		)
		if err != nil && !aiven.IsAlreadyExists(err) {
			return err
		}
	}

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

func (h KafkaACLHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	acl, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = avn.KafkaACLs.Delete(acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
	if err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete Kafka ACL error: %w", err)
	}

	return true, nil
}

func (h KafkaACLHandler) exists(avn *aiven.Client, acl *v1alpha1.KafkaACL) (bool, error) {
	var aivenACL *aiven.KafkaACL
	var err error
	if acl.Status.ID != "" {
		aivenACL, err = avn.KafkaACLs.Get(acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
		if err != nil {
			return false, err
		}
	} else {
		list, err := avn.KafkaACLs.List(acl.Spec.Project, acl.Spec.ServiceName)
		if err != nil {
			return false, err
		}

		for _, a := range list {
			if acl.Spec.Topic == a.Topic && acl.Spec.Username == a.Username && acl.Spec.Permission == a.Permission {
				aivenACL = a
			}
		}
	}

	return aivenACL != nil, nil
}

func (h KafkaACLHandler) get(_ *aiven.Client, i client.Object) (*corev1.Secret, error) {
	acl, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta, instanceIsRunningAnnotation, "true")

	return nil, nil
}

func (h KafkaACLHandler) checkPreconditions(avn *aiven.Client, i client.Object) (bool, error) {
	acl, err := h.convert(i)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsRunning(avn, acl.Spec.Project, acl.Spec.ServiceName)
}

func (h KafkaACLHandler) fetchOwners(ctx context.Context, i client.Object) ([]client.Object, error) {
	kacl, err := h.convert(i)
	if err != nil {
		return nil, err
	}
	ownerKey := types.NamespacedName{Name: kacl.Spec.Topic, Namespace: kacl.GetNamespace()}

	return findSingleOwner(ctx, h.k8s, ownerKey, &v1alpha1.KafkaTopic{})
}

func (h KafkaACLHandler) convert(i client.Object) (*v1alpha1.KafkaACL, error) {
	acl, ok := i.(*v1alpha1.KafkaACL)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaACL")
	}

	return acl, nil
}
