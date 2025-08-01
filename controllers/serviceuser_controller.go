// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"fmt"
	"strconv"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
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

func newServiceUserReconciler(c Controller) reconcilerType {
	return &ServiceUserReconciler{Controller: c}
}

type ServiceUserHandler struct {
	k8s client.Client
}

//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers,verbs=update;get;list;watch;create;delete
//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers/finalizers,verbs=get;create;update

func (r *ServiceUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	handler := &ServiceUserHandler{
		k8s: r.Client,
	}
	return r.reconcileInstance(ctx, req, handler, &v1alpha1.ServiceUser{})
}

func (r *ServiceUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceUser{}).
		Complete(r)
}

func (h *ServiceUserHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	user, err := h.convert(obj)
	if err != nil {
		return err
	}

	newPassword, err := GetPasswordFromSecret(ctx, h.k8s, user)
	if err != nil {
		return fmt.Errorf("failed to get password from secret: %w", err)
	}

	u, err := avnGen.ServiceUserCreate(
		ctx, user.Spec.Project, user.Spec.ServiceName,
		&service.ServiceUserCreateIn{
			Username: user.Name,
		},
	)
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("cannot createOrUpdate service user on aiven side: %w", err)
	}

	// modify credentials using the password from source secret
	if newPassword != "" {
		modifyReq := &service.ServiceUserCredentialsModifyIn{
			NewPassword: &newPassword,
			Operation:   service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
		}

		_, err = avnGen.ServiceUserCredentialsModify(ctx, user.Spec.Project, user.Spec.ServiceName, user.Name, modifyReq)
		if err != nil {
			return err
		}
	}

	if u != nil {
		user.Status.Type = u.Type
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Created",
			"Successfully created or updated the instance in Aiven"))

	metav1.SetMetaDataAnnotation(&user.ObjectMeta,
		processedGenerationAnnotation, strconv.FormatInt(user.GetGeneration(), formatIntBaseDecimal))

	return nil
}

func (h *ServiceUserHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	user, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// skip deletion for built-in users that cannot be deleted
	if isBuiltInUser(user.Name) {
		// built-in users like avnadmin cannot be deleted, this is expected behavior
		// we consider this a successful deletion since we can't and shouldn't delete built-in users
		return true, nil
	}

	err = avnGen.ServiceUserDelete(ctx, user.Spec.Project, user.Spec.ServiceName, user.Name)
	if isNotFound(err) {
		// consider it a successful deletion
		return true, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (h *ServiceUserHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	user, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	u, err := avnGen.ServiceUserGet(ctx, user.Spec.Project, user.Spec.ServiceName, user.Name)
	if err != nil {
		return nil, err
	}

	s, err := avnGen.ServiceGet(ctx, user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return nil, err
	}

	var component *service.ComponentOut
	for _, c := range s.Components {
		if c.Component == s.ServiceType || (s.ServiceType == "alloydbomni" && c.Component == "pg") {
			component = &c
			break
		}
	}

	if component == nil {
		return nil, fmt.Errorf("service component %q not found", s.ServiceType)
	}

	caCert, err := avnGen.ProjectKmsGetCA(ctx, user.Spec.Project)
	if err != nil {
		return nil, fmt.Errorf("aiven client error %w", err)
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getRunningCondition(metav1.ConditionTrue, "CheckRunning",
			"Instance is running on Aiven side"))

	metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

	prefix := getSecretPrefix(user)
	stringData := map[string]string{
		prefix + "HOST":        component.Host,
		prefix + "PORT":        fmt.Sprintf("%d", component.Port),
		prefix + "USERNAME":    u.Username,
		prefix + "PASSWORD":    u.Password,
		prefix + "ACCESS_CERT": fromAnyPointer(u.AccessCert),
		prefix + "ACCESS_KEY":  fromAnyPointer(u.AccessKey),
		prefix + "CA_CERT":     caCert,
		// todo: remove in future releases
		"HOST":        component.Host,
		"PORT":        fmt.Sprintf("%d", component.Port),
		"USERNAME":    u.Username,
		"PASSWORD":    u.Password,
		"ACCESS_CERT": fromAnyPointer(u.AccessCert),
		"ACCESS_KEY":  fromAnyPointer(u.AccessKey),
		"CA_CERT":     caCert,
	}

	return newSecret(user, stringData, false), nil
}

func (h *ServiceUserHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	user, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsOperational(ctx, avnGen, user.Spec.Project, user.Spec.ServiceName)
}

func (h *ServiceUserHandler) convert(i client.Object) (*v1alpha1.ServiceUser, error) {
	db, ok := i.(*v1alpha1.ServiceUser)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceUser")
	}

	return db, nil
}
