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

const (
	// avnadminBuiltInUser is the built-in admin user that cannot be deleted
	// This is an exception case - built-in users are created automatically by Aiven
	avnadminBuiltInUser = "avnadmin"
)

type ServiceUserHandler struct {
	k8s client.Client
}

//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers,verbs=update;get;list;watch;create;delete
//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=serviceusers/finalizers,verbs=get;create;update

func (r *ServiceUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.reconcileInstance(ctx, req, ServiceUserHandler{k8s: r.Client}, &v1alpha1.ServiceUser{})
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

	var newPassword string
	if user.Spec.ConnInfoSecretSource != nil {
		newPassword, err = h.getPasswordFromSecret(ctx, user)
		if err != nil {
			return fmt.Errorf("failed to get password from secret: %w", err)
		}
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
		if err = h.modifyCredentials(ctx, avnGen, user, newPassword); err != nil {
			return fmt.Errorf("failed to modify service user credentials: %w", err)
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

// getPasswordFromSecret retrieves and validates the password from connInfoSecretSource
func (h ServiceUserHandler) getPasswordFromSecret(ctx context.Context, user *v1alpha1.ServiceUser) (string, error) {
	secretSource := user.Spec.ConnInfoSecretSource
	if secretSource == nil {
		return "", nil
	}

	sourceNamespace := secretSource.Namespace
	if sourceNamespace == "" {
		sourceNamespace = user.GetNamespace()
	}

	sourceSecret := &corev1.Secret{}
	err := h.k8s.Get(ctx, types.NamespacedName{
		Name:      secretSource.Name,
		Namespace: sourceNamespace,
	}, sourceSecret)
	if err != nil {
		return "", fmt.Errorf("failed to read connInfoSecretSource %s/%s: %w", sourceNamespace, secretSource.Name, err)
	}

	passwordKey := secretSource.PasswordKey
	passwordBytes, exists := sourceSecret.Data[passwordKey]
	if !exists {
		return "", fmt.Errorf("password not found in source secret %s/%s (expected %s key)", sourceNamespace, secretSource.Name, passwordKey)
	}

	newPassword := string(passwordBytes)

	// validate password length according to Aiven API requirements
	if len(newPassword) < 8 || len(newPassword) > 256 {
		return "", fmt.Errorf("password length must be between 8 and 256 characters, got %d characters from source secret %s/%s (key: %s)",
			len(newPassword), sourceNamespace, secretSource.Name, passwordKey)
	}

	return newPassword, nil
}

// modifyCredentials performs the actual credential modification
func (h ServiceUserHandler) modifyCredentials(ctx context.Context, avnGen avngen.Client, user *v1alpha1.ServiceUser, password string) error {
	modifyReq := &service.ServiceUserCredentialsModifyIn{
		NewPassword: &password,
		Operation:   service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
	}

	_, err := avnGen.ServiceUserCredentialsModify(ctx, user.Spec.Project, user.Spec.ServiceName, user.Name, modifyReq)
	if err != nil {
		return fmt.Errorf("failed to modify service user credentials in Aiven: %w", err)
	}

	return nil
}

// isBuiltInUser checks if the username is a known built-in user that cannot be deleted.
// Built-in users like 'avnadmin' are created automatically by Aiven and persist even when ServiceUser resources are deleted.
func (h ServiceUserHandler) isBuiltInUser(username string) bool {
	return username == avnadminBuiltInUser
}

func (h ServiceUserHandler) delete(ctx context.Context, avnGen avngen.Client, obj client.Object) (bool, error) {
	user, err := h.convert(obj)
	if err != nil {
		return false, err
	}

	// skip deletion for built-in users that cannot be deleted
	if h.isBuiltInUser(user.Name) {
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

func (h ServiceUserHandler) get(ctx context.Context, avnGen avngen.Client, obj client.Object) (*corev1.Secret, error) {
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
