// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aiven/aiven-go-client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

// ServiceUserReconciler reconciles a ServiceUser object
type ServiceUserReconciler struct {
	Controller
}

type ServiceUserHandler struct{}

// +kubebuilder:rbac:groups=aiven.io,resources=serviceusers,verbs=update;get;list;watch;create;delete
// +kubebuilder:rbac:groups=aiven.io,resources=serviceusers/status,verbs=get;update
// +kubebuilder:rbac:groups=aiven.io,resources=serviceusers/finalizers,verbs=update

func (r *ServiceUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ServiceUserHandler{}, &v1alpha1.ServiceUser{})
}

func (r *ServiceUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceUser{}).
		Complete(r)
}

func (h ServiceUserHandler) createOrUpdate(avn *aiven.Client, i client.Object, refs []client.Object) error {
	user, err := h.convert(i)
	if err != nil {
		return err
	}

	u, err := avn.ServiceUsers.Create(user.Spec.Project, user.Spec.ServiceName,
		aiven.CreateServiceUserRequest{
			Username: user.Name,
			AccessControl: &aiven.AccessControl{
				RedisACLCategories: []string{},
				RedisACLCommands:   []string{},
				RedisACLChannels:   []string{},
				RedisACLKeys:       []string{},
			},
		})
	if err != nil && !aiven.IsAlreadyExists(err) {
		return fmt.Errorf("cannot createOrUpdate service user on aiven side: %w", err)
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
		processedGenerationAnnotation, strconv.FormatInt(user.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h ServiceUserHandler) delete(avn *aiven.Client, i client.Object) (bool, error) {
	user, err := h.convert(i)
	if err != nil {
		return false, err
	}

	err = avn.ServiceUsers.Delete(user.Spec.Project, user.Spec.ServiceName, user.Name)
	if !aiven.IsNotFound(err) {
		return false, err
	}

	return true, nil
}

func (h ServiceUserHandler) get(avn *aiven.Client, i client.Object) (*corev1.Secret, error) {
	user, err := h.convert(i)
	if err != nil {
		return nil, err
	}

	u, err := avn.ServiceUsers.Get(user.Spec.Project, user.Spec.ServiceName, user.Name)
	if err != nil {
		return nil, err
	}

	s, err := avn.Services.Get(user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return nil, err
	}

	params := s.URIParams

	caCert, err := avn.CA.Get(user.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("aiven client error %w", err)
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.getSecretName(user),
			Namespace: user.Namespace,
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

func (h ServiceUserHandler) getSecretName(user *v1alpha1.ServiceUser) string {
	if user.Spec.ConnInfoSecretTarget.Name != "" {
		return user.Spec.ConnInfoSecretTarget.Name
	}
	return user.Name
}

func (h ServiceUserHandler) checkPreconditions(avn *aiven.Client, i client.Object) (bool, error) {
	user, err := h.convert(i)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsRunning(avn, user.Spec.Project, user.Spec.ServiceName)
}

func (h ServiceUserHandler) convert(i client.Object) (*v1alpha1.ServiceUser, error) {
	db, ok := i.(*v1alpha1.ServiceUser)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceUser")
	}

	return db, nil
}
