// Copyright (c) 2021 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// KafkaACLReconciler reconciles a KafkaACL object
type KafkaACLReconciler struct {
	Controller
}

type KafkaACLHandler struct {
	Handlers
	client *aiven.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=aiven.io,resources=kafkaacls/status,verbs=get

func (r *KafkaACLReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	acl := &k8soperatorv1alpha1.KafkaACL{}
	err := r.Get(ctx, req.NamespacedName, acl)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c, err := r.InitAivenClient(ctx, req, acl.Spec.AuthSecretRef)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileInstance(ctx, &KafkaACLHandler{
		client: c,
	}, acl)
}

func (r *KafkaACLReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.KafkaACL{}).
		Complete(r)
}

func (h KafkaACLHandler) createOrUpdate(i client.Object) (client.Object, error) {
	acl, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	exists, err := h.exists(acl)
	if err != nil {
		return nil, err
	}

	if !exists {
		_, err = h.client.KafkaACLs.Create(
			acl.Spec.Project,
			acl.Spec.ServiceName,
			aiven.CreateKafkaACLRequest{
				Permission: acl.Spec.Permission,
				Topic:      acl.Spec.Topic,
				Username:   acl.Spec.Username,
			},
		)
		if err != nil && !aiven.IsAlreadyExists(err) {
			return nil, err
		}
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getInitializedCondition("CreatedOrUpdate",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "CreatedOrUpdate",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta,
		processedGeneration, strconv.FormatInt(acl.GetGeneration(), 10))

	return acl, nil
}

func (h KafkaACLHandler) delete(i client.Object) (bool, error) {
	acl, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = h.client.KafkaACLs.Delete(acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
	if err != nil && !aiven.IsNotFound(err) {
		return false, fmt.Errorf("aiven client delete Kafka ACL error: %w", err)
	}

	return true, nil
}

func (h KafkaACLHandler) exists(acl *k8soperatorv1alpha1.KafkaACL) (bool, error) {
	var aivenACL *aiven.KafkaACL
	var err error
	if acl.Status.ID != "" {
		aivenACL, err = h.client.KafkaACLs.Get(acl.Spec.Project, acl.Spec.ServiceName, acl.Status.ID)
		if err != nil {
			return false, err
		}
	} else {
		list, err := h.client.KafkaACLs.List(acl.Spec.Project, acl.Spec.ServiceName)
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

func (h KafkaACLHandler) get(i client.Object) (client.Object, *corev1.Secret, error) {
	acl, err := h.convert(i)
	if err != nil {
		return nil, nil, err
	}

	meta.SetStatusCondition(&acl.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "Get",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&acl.ObjectMeta, isRunning, "true")

	return acl, nil, nil
}

func (h KafkaACLHandler) checkPreconditions(i client.Object) bool {
	acl, err := h.convert(i)
	if err != nil {
		return false
	}

	return checkServiceIsRunning(h.client, acl.Spec.Project, acl.Spec.ServiceName)
}

func (h KafkaACLHandler) convert(i client.Object) (*k8soperatorv1alpha1.KafkaACL, error) {
	acl, ok := i.(*k8soperatorv1alpha1.KafkaACL)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to KafkaACL")
	}

	return acl, nil
}
