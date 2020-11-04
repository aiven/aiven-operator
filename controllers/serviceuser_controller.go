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

func (r *ServiceUserReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
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
	// new one if service user is not found
	_, err = r.AivenClient.ServiceUsers.Get(user.Spec.Project, user.Spec.ServiceName, user.Spec.Username)
	if err != nil {
		// Create a new service user if it does not exists and update CR status
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

// createUser creates a service user on Aiven side
func (r *ServiceUserReconciler) createUser(user *k8soperatorv1alpha1.ServiceUser) (*aiven.ServiceUser, error) {
	u, err := r.AivenClient.ServiceUsers.Create(user.Spec.Project, user.Spec.ServiceName,
		aiven.CreateServiceUserRequest{
			Username: user.Spec.Username,
		})
	if err != nil {
		return nil, err
	}

	// Update service user custom resource status
	err = r.updateCRStatus(user, u)
	if err != nil {
		return nil, fmt.Errorf("failed to update service user status: %w", err)
	}

	// Get CA Certificate of a newly created service user and save it as K8s secret
	err = r.createSecret(user, u)
	if err != nil {
		return nil, fmt.Errorf("failed to create service user secret: %w", err)
	}

	return u, err
}

// createSecret creates a CA service user certificate secret
func (r *ServiceUserReconciler) createSecret(user *k8soperatorv1alpha1.ServiceUser, u *aiven.ServiceUser) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s", u.Username),
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
		return fmt.Errorf("k8s client create error %w", err)
	}

	// Set service user instance as the owner and controller
	err = controllerutil.SetControllerReference(user, secret, r.Scheme)
	if err != nil {
		return fmt.Errorf("k8s set controller error %w", err)
	}

	return nil
}

// updateCRStatus updates Kubernetes Custom Resource status
func (r *ServiceUserReconciler) updateCRStatus(user *k8soperatorv1alpha1.ServiceUser, u *aiven.ServiceUser) error {
	user.Status.Username = u.Username
	user.Status.Type = u.Type

	err := r.Status().Update(context.Background(), user)
	if err != nil {
		return err
	}

	return nil
}

func (r *ServiceUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8soperatorv1alpha1.ServiceUser{}).
		Complete(r)
}
