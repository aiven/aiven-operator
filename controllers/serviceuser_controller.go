// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-kubernetes-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceUserReconciler reconciles a ServiceUser object
type ServiceUserReconciler struct {
	Controller
}

type ServiceUserHandler struct {
	Handlers
	client *aiven.Client
}

// +kubebuilder:rbac:groups=aiven.io,resources=serviceusers,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=aiven.io,resources=serviceusers/status,verbs=get

func (r *ServiceUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	su := &k8soperatorv1alpha1.ServiceUser{}
	err := r.Get(ctx, req.NamespacedName, su)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	c, err := r.InitAivenClient(ctx, req, su.Spec.AuthSecretRef)
	if err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileInstance(ctx, &ServiceUserHandler{client: c}, su)
}

func (r *ServiceUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ServiceUser{}).
		Complete(r)
}

func (h *ServiceUserHandler) createOrUpdate(i client.Object) (client.Object, error) {
	user, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	u, err := h.client.ServiceUsers.Create(user.Spec.Project, user.Spec.ServiceName,
		aiven.CreateServiceUserRequest{
			Username: user.Name,
			AccessControl: aiven.AccessControl{
				RedisACLCategories: []string{},
				RedisACLCommands:   []string{},
				RedisACLKeys:       []string{},
			},
		})
	if err != nil && !aiven.IsAlreadyExists(err) {
		return nil, fmt.Errorf("cannot createOrUpdate service user on aiven side: %w", err)
	}

	if u != nil {
		user.Status.Type = u.Type
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Created",
			"Instance was created or update on Aiven side"))

	meta.SetStatusCondition(&user.Status.Conditions,
		getRunningCondition(metav1.ConditionUnknown, "Created",
			"Instance was created or update on Aiven side, status remains unknown"))

	metav1.SetMetaDataAnnotation(&user.ObjectMeta,
		processedGeneration, strconv.FormatInt(user.GetGeneration(), 10))

	return user, nil
}

func (h ServiceUserHandler) delete(i client.Object) (bool, error) {
	user, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = h.client.ServiceUsers.Delete(user.Spec.Project, user.Spec.ServiceName, user.Name)
	if !aiven.IsNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h ServiceUserHandler) get(i client.Object) (client.Object, *corev1.Secret, error) {
	user, err := h.convert(i)
	if err != nil {
		return nil, nil, err
	}

	u, err := h.client.ServiceUsers.Get(user.Spec.Project, user.Spec.ServiceName, user.Name)
	if err != nil {
		return nil, nil, err
	}

	s, err := h.client.Services.Get(user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return nil, nil, err
	}

	params := s.URIParams

	caCert, err := h.client.CA.Get(user.Spec.Project)
	if err != nil {
		return nil, nil, fmt.Errorf("aiven client error %w", err)
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&user.ObjectMeta, isRunning, "true")

	return user, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(user),
			Namespace: user.Namespace,
			Labels: map[string]string{
				"app": user.Name,
			},
		},
		StringData: map[string]string{
			"HOST":        params["host"],
			"PORT":        params["port"],
			"USERNAME":    u.Username,
			"PASSWORD":    u.Password,
			"ACCESS_CERT": u.AccessCert,
			"ACCESS_KEY":  u.AccessKey,
			"CA_CERT":     caCert,
		},
	}, nil
}

func (h ServiceUserHandler) getSecretName(user *k8soperatorv1alpha1.ServiceUser) string {
	if user.Spec.ConnInfoSecretTarget.Name != "" {
		return user.Spec.ConnInfoSecretTarget.Name
	}
	return user.Name
}

func (h ServiceUserHandler) checkPreconditions(i client.Object) bool {
	user, err := h.convert(i)
	if err != nil {
		return false
	}

	return checkServiceIsRunning(h.client, user.Spec.Project, user.Spec.ServiceName)
}

func (h ServiceUserHandler) convert(i client.Object) (*k8soperatorv1alpha1.ServiceUser, error) {
	db, ok := i.(*k8soperatorv1alpha1.ServiceUser)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceUser")
	}

	return db, nil
}
