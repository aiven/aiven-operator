// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
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
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceusers,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceusers/status,verbs=get

func (r *ServiceUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("serviceuser", req.NamespacedName)
	log.Info("Reconciling Aiven ServiceUser")

	const finalizer = "serviceuser-finalizer.k8s-operator.aiven.io"
	su := &k8soperatorv1alpha1.ServiceUser{}
	return r.reconcileInstance(&ServiceUserHandler{}, ctx, log, req, su, finalizer)
}

func (r *ServiceUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ServiceUser{}).
		Complete(r)
}

func (h *ServiceUserHandler) create(log logr.Logger, i client.Object) (client.Object, error) {
	user, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	log.Info("Creating Service User")

	u, err := aivenClient.ServiceUsers.Create(user.Spec.Project, user.Spec.ServiceName,
		aiven.CreateServiceUserRequest{
			Username: user.Spec.Username,
			AccessControl: aiven.AccessControl{
				RedisACLCategories: []string{},
				RedisACLCommands:   []string{},
				RedisACLKeys:       []string{},
			},
		})
	if err != nil {
		if aiven.IsAlreadyExists(err) {
			return user, nil
		}
		return nil, fmt.Errorf("cannot create service user on Aiven side: %w", err)
	}

	h.setStatus(user, u)

	return user, nil

}

func (h ServiceUserHandler) delete(_ logr.Logger, i client.Object) (client.Object, bool, error) {
	user, err := h.convert(i)
	if err != nil {
		return nil, false, err
	}

	err = aivenClient.ServiceUsers.Delete(user.Spec.Project, user.Spec.ServiceName, user.Spec.Username)
	if !aiven.IsNotFound(err) {
		return nil, false, err
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", user.Spec.Username, "-secret"),
			Namespace: user.Namespace,
			Labels: map[string]string{
				"app": user.Name,
			},
		},
	}, true, nil
}

func (h ServiceUserHandler) exists(_ logr.Logger, i client.Object) (exists bool, error error) {
	user, err := h.convert(i)
	if err != nil {
		return false, err
	}

	u, err := aivenClient.ServiceUsers.Get(user.Spec.Project, user.Spec.ServiceName, user.Spec.Username)
	if !aiven.IsNotFound(err) {
		return false, err
	}

	return u != nil, nil
}

func (h ServiceUserHandler) update(_ logr.Logger, _ client.Object) (updatedObj client.Object, error error) {
	return nil, nil
}

func (h ServiceUserHandler) getSecret(_ logr.Logger, i client.Object) (secret *corev1.Secret, error error) {
	user, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	u, err := aivenClient.ServiceUsers.Get(user.Spec.Project, user.Spec.ServiceName, user.Spec.Username)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", u.Username, "-secret"),
			Namespace: user.Namespace,
			Labels: map[string]string{
				"app": user.Name,
			},
		},
		StringData: map[string]string{
			"password":    u.Password,
			"access_cert": u.AccessCert,
			"access_key":  u.AccessKey,
		},
	}, nil
}

func (h ServiceUserHandler) checkPreconditions(log logr.Logger, i client.Object) bool {
	user, err := h.convert(i)
	if err != nil {
		return false
	}

	log.Info("Checking ServiceUser preconditions")

	return checkServiceIsRunning(user.Spec.Project, user.Spec.ServiceName)
}

func (h ServiceUserHandler) isActive(_ logr.Logger, _ client.Object) (bool, error) {
	return true, nil
}

func (h ServiceUserHandler) convert(i client.Object) (*k8soperatorv1alpha1.ServiceUser, error) {
	db, ok := i.(*k8soperatorv1alpha1.ServiceUser)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceUser")
	}

	return db, nil
}

func (h ServiceUserHandler) setStatus(user *k8soperatorv1alpha1.ServiceUser, u *aiven.ServiceUser) {
	user.Status.Username = u.Username
	user.Status.ServiceName = user.Spec.ServiceName
	user.Status.Project = user.Spec.Project
	user.Status.Type = u.Type
	user.Status.Authentication = user.Spec.Authentication
}
