// Copyright (c) 2020 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	k8soperatorv1alpha1 "github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ServiceUserReconciler reconciles a ServiceUser object
type ServiceUserReconciler struct {
	Controller
}

// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s-operator.aiven.io,resources=serviceusers/status,verbs=get;update;patch

func (r *ServiceUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("serviceuser", req.NamespacedName)

	if err := r.InitAivenClient(req, ctx, log); err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the ServiceUser instance
	user := &k8soperatorv1alpha1.ServiceUser{}
	err := r.Get(ctx, req.NamespacedName, user)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("ServiceUser resource not token. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ServiceUser")
		return ctrl.Result{}, err
	}

	// Check if service user already exists on the Aiven side, create a
	// new one if it is not found
	_, err = r.AivenClient.ServiceUsers.Get(user.Spec.Project, user.Spec.ServiceName, user.Spec.Username)
	if err != nil {
		// Create a new one if service user does not exists and update CR status
		if aiven.IsNotFound(err) {
			_, err = r.createUser(user)
			if err != nil {
				log.Error(err, "Failed to create ServiceUser")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ServiceUserReconciler) createUser(user *k8soperatorv1alpha1.ServiceUser) (*aiven.ServiceUser, error) {
	r.Log.Info("project: " + user.Spec.Project + " service: " + user.Spec.ServiceName)
	u, err := r.AivenClient.ServiceUsers.Create(user.Spec.Project, user.Spec.ServiceName,
		aiven.CreateServiceUserRequest{
			Username: user.Spec.Username,
			AccessControl: aiven.AccessControl{
				RedisACLCategories: []string{},
				RedisACLCommands:   []string{},
				RedisACLKeys:       []string{},
			},
		})
	if err != nil {
		return nil, err
	}

	// Update service user custom resource status
	err = r.updateCRStatus(user, u)
	if err != nil {
		return nil, fmt.Errorf("failed to update ServiceUser status: %w", err)
	}

	// Create password and access cert/key k8s secrets
	err = r.createSecret(user, u)
	if err != nil {
		return nil, fmt.Errorf("failed to create ServiceUser Secrets: %w", err)
	}

	return u, nil
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *ServiceUserReconciler) updateCRStatus(user *k8soperatorv1alpha1.ServiceUser, u *aiven.ServiceUser) error {
	user.Status.Username = u.Username
	user.Status.ServiceName = user.Spec.ServiceName
	user.Status.Project = user.Spec.Project
	user.Status.Type = u.Type
	user.Status.Authentication = user.Spec.Authentication

	return r.Status().Update(context.Background(), user)
}

// ServiceUserReconciler creates password and access cert/key k8s secrets
func (r *ServiceUserReconciler) createSecret(user *k8soperatorv1alpha1.ServiceUser, u *aiven.ServiceUser) error {
	secret := &corev1.Secret{
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
	}
	err := r.Client.Create(context.Background(), secret)
	if err != nil {
		return fmt.Errorf("create ServiceUser sercret %w", err)
	}

	// Set ServiceUser instance as the owner and controller
	err = controllerutil.SetControllerReference(user, secret, r.Scheme)
	if err != nil {
		return fmt.Errorf("k8s set ServiceUser sercret controller error %w", err)
	}

	return nil
}

func (r *ServiceUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ServiceUser{}).
		Complete(r)
}
