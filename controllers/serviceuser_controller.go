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
	"k8s.io/apimachinery/pkg/types"
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

type ServiceUserHandler struct{}

//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers,verbs=update;get;list;watch;create;delete
//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers/finalizers,verbs=get;create;update

func (r *ServiceUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ServiceUserHandler{}, &v1alpha1.ServiceUser{})
}

func (r *ServiceUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceUser{}).
		Complete(r)
}

func (h ServiceUserHandler) createOrUpdate(ctx context.Context, avnGen avngen.Client, obj client.Object, _ []client.Object) error {
	user, err := h.convert(obj)
	if err != nil {
		return err
	}

	targetUsername := user.GetTargetUsername()
	u, err := avnGen.ServiceUserCreate(
		ctx, user.Spec.Project, user.Spec.ServiceName,
		&service.ServiceUserCreateIn{
			Username: targetUsername,
		},
	)
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("cannot createOrUpdate service user on aiven side: %w", err)
	}

	// Note: Password modification is handled later in updateInstanceStateAndSecretUntilRunning
	// where we have access to both Aiven and Kubernetes clients

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

// isBuiltInUser checks if the username is a known built-in user that cannot be deleted
func (h ServiceUserHandler) isBuiltInUser(username string) bool {
	builtInUsers := []string{
		"avnadmin", // default admin user
	}

	for _, builtIn := range builtInUsers {
		if username == builtIn {
			return true
		}
	}
	return false
}

func (h ServiceUserHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	user, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	targetUsername := user.GetTargetUsername()

	// skip deletion for built-in users that cannot be deleted
	if h.isBuiltInUser(targetUsername) {
		// built-in users like avnadmin cannot be deleted, this is expected behavior
		// we consider this a successful deletion since we can't and shouldn't delete built-in users
		return true, nil
	}

	err = avnGen.ServiceUserDelete(ctx, user.Spec.Project, user.Spec.ServiceName, targetUsername)
	if isNotFound(err) {
		// User not found, deletion is successful
		return true, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (h ServiceUserHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
	user, err := h.convert(obj)
	if err != nil {
		return nil, err
	}

	targetUsername := user.GetTargetUsername()
	u, err := avnGen.ServiceUserGet(ctx, user.Spec.Project, user.Spec.ServiceName, targetUsername)
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

func (h ServiceUserHandler) checkPreconditions(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	user, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsOperational(ctx, avnGen, user.Spec.Project, user.Spec.ServiceName)
}

func (h ServiceUserHandler) convert(i client.Object) (*v1alpha1.ServiceUser, error) {
	db, ok := i.(*v1alpha1.ServiceUser)
	if !ok {
		return nil, fmt.Errorf("cannot convert object to ServiceUser")
	}

	return db, nil
}

// modifyCredentials allows ServiceUser to modify credentials from a source secret before secret generation
func (h ServiceUserHandler) modifyCredentials(ctx context.Context, avnGen avngen.Client, k8s client.Client, obj client.Object, secretSource *v1alpha1.ConnInfoSecretSource) error {
	user, err := h.convert(obj)
	if err != nil {
		return err
	}

	sourceNamespace := secretSource.Namespace
	if sourceNamespace == "" {
		sourceNamespace = user.GetNamespace()
	}

	sourceSecret := &corev1.Secret{}
	err = k8s.Get(ctx, types.NamespacedName{
		Name:      secretSource.Name,
		Namespace: sourceNamespace,
	}, sourceSecret)
	if err != nil {
		return fmt.Errorf("failed to read connInfoSecretSource %s/%s: %w", sourceNamespace, secretSource.Name, err)
	}

	var newPassword string

	if passwordBytes, exists := sourceSecret.Data["PASSWORD"]; exists {
		newPassword = string(passwordBytes)
	} else {
		return fmt.Errorf("password not found in source secret %s/%s (expected PASSWORD key)", sourceNamespace, secretSource.Name)
	}

	if newPassword == "" {
		return fmt.Errorf("password is empty in source secret %s/%s", sourceNamespace, secretSource.Name)
	}

	modifyReq := &service.ServiceUserCredentialsModifyIn{
		NewPassword: &newPassword,
		Operation:   service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
	}

	targetUsername := user.GetTargetUsername()
	_, err = avnGen.ServiceUserCredentialsModify(ctx, user.Spec.Project, user.Spec.ServiceName, targetUsername, modifyReq)
	if err != nil {
		return fmt.Errorf("failed to modify service user credentials in Aiven: %w", err)
	}

	return nil
}
