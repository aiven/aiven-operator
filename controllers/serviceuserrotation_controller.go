// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

const (
	serviceUserRotationPublishedAtKey     = "aiven-rotation-published-at"
	serviceUserRotationDesiredUsernameKey = "aiven-rotation-desired-username"
	serviceUserRotationDesiredPasswordKey = "aiven-rotation-desired-password"
)

//+kubebuilder:rbac:groups=aiven.io,resources=serviceuserrotations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=serviceuserrotations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=aiven.io,resources=serviceuserrotations/finalizers,verbs=get;create;update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

func newServiceUserRotationReconciler(c Controller) reconcilerType {
	if c.newAivenClient == nil {
		c.newAivenClient = NewAivenGeneratedClient
	}
	return &ServiceUserRotationReconciler{
		Controller:       c,
		now:              time.Now,
		generatePassword: rand.Text,
	}
}

// ServiceUserRotationReconciler stores pending and published credentials in one Secret.
// Publishing credentials and clearing the pending pair share one Secret update.
type ServiceUserRotationReconciler struct {
	Controller
	now              func() time.Time
	generatePassword func() string
}

func (r *ServiceUserRotationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ServiceUserRotation{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (r *ServiceUserRotationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	cr := &v1alpha1.ServiceUserRotation{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ctx = logr.NewContext(ctx, setupLogger(r.Log, cr))

	// Finalization must not depend on service readiness or valid rotation state.
	if isMarkedForDeletion(cr) {
		return ctrl.Result{}, r.reconcileDeletion(ctx, cr)
	}

	orig := cr.DeepCopy()
	defer func() {
		err = errors.Join(err, r.persistStatus(ctx, orig, cr))
		if err != nil {
			result = ctrl.Result{}
		}
	}()

	if err := r.reconcile(ctx, cr); err != nil {
		condition := getErrorCondition("ReconcileFailed", err)
		condition.ObservedGeneration = cr.Generation
		meta.SetStatusCondition(cr.Conditions(), condition)
		r.Recorder.Event(cr, corev1.EventTypeWarning, condition.Reason, err.Error())
		return ctrl.Result{}, err
	}
	meta.RemoveStatusCondition(cr.Conditions(), ConditionTypeError)
	// Encode empty conditions as [] to satisfy the CRD schema.
	if cr.Status.Conditions == nil {
		cr.Status.Conditions = []metav1.Condition{}
	}
	return ctrl.Result{RequeueAfter: min(r.pollInterval(), max(time.Nanosecond, cr.Status.NextRotationAt.Sub(r.now())))}, nil
}

func (r *ServiceUserRotationReconciler) persistStatus(ctx context.Context, orig, cr *v1alpha1.ServiceUserRotation) error {
	// metav1.Time serializes to seconds. Normalize after scheduling so an
	// unchanged publication does not cause repeated status updates.
	cr.Status.LastRotationAt.Time = cr.Status.LastRotationAt.Truncate(time.Second)
	cr.Status.NextRotationAt.Time = cr.Status.NextRotationAt.Truncate(time.Second)
	if equality.Semantic.DeepEqual(orig.Status, cr.Status) {
		return nil
	}
	return r.Status().Update(ctx, cr)
}

func (r *ServiceUserRotationReconciler) reconcile(ctx context.Context, cr *v1alpha1.ServiceUserRotation) error {
	if err := r.ensureSecretFinalizer(ctx, cr); err != nil {
		return err
	}
	if err := r.ensureFinalizer(ctx, cr); err != nil {
		return err
	}

	cl, err := r.aivenClient(ctx, cr)
	if err != nil {
		return err
	}
	svc, err := getServiceIfOperational(ctx, cl, cr.Spec.Project, cr.Spec.ServiceName)
	if err != nil {
		return err
	}

	secret, err := r.ensureSecret(ctx, cr)
	if err != nil {
		return err
	}

	if err := r.ensureUsers(ctx, cr, cl, svc); err != nil {
		return err
	}

	return r.reconcileCredentials(ctx, cl, cr, secret, svc)
}

func (r *ServiceUserRotationReconciler) pollInterval() time.Duration {
	if r.PollInterval <= 0 {
		return DefaultPollInterval
	}
	return r.PollInterval
}

// ensureSecret reads the owned rotation Secret or returns an unsaved template.
// The first write includes the pending pair so an interrupted rotation can resume.
func (r *ServiceUserRotationReconciler) ensureSecret(ctx context.Context, cr *v1alpha1.ServiceUserRotation) (*corev1.Secret, error) {
	name := connectionSecretName(cr)
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: name}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			secret = newSecret(cr, nil, false)
			secret.Type = corev1.SecretTypeOpaque
			secret.Data = map[string][]byte{}
			if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
				return nil, err
			}
			return secret, nil
		}
		return nil, fmt.Errorf("getting rotation Secret %s/%s: %w", cr.Namespace, name, err)
	}
	if !metav1.IsControlledBy(secret, cr) {
		return nil, fmt.Errorf("secret %s/%s is not controlled by ServiceUserRotation %s", cr.Namespace, name, cr.Name)
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	return secret, nil
}

func serviceUserRotationPublishedAt(secret *corev1.Secret) (time.Time, error) {
	data, exists := secret.Data[serviceUserRotationPublishedAtKey]
	if !exists {
		return time.Time{}, nil
	}
	publishedAt, err := time.Parse(time.RFC3339Nano, string(data))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid rotation publication time: %w", err)
	}
	return publishedAt, nil
}

// reconcileCredentials publishes a new rotation or refreshes the current connection details.
func (r *ServiceUserRotationReconciler) reconcileCredentials(ctx context.Context, cl avngen.Client, cr *v1alpha1.ServiceUserRotation, secret *corev1.Secret, svc *service.ServiceGetOut) error {
	orig := secret.DeepCopy()

	publishedAt, err := serviceUserRotationPublishedAt(secret)
	if err != nil {
		return err
	}

	// Reuse only a complete candidate from the configured pool. Invalid candidates must not bring the next rotation forward.
	hasPendingCredentials := slices.Contains(cr.Spec.Usernames, string(secret.Data[serviceUserRotationDesiredUsernameKey])) &&
		len(secret.Data[serviceUserRotationDesiredPasswordKey]) > 0
	rotate := hasPendingCredentials || publishedAt.IsZero() || !publishedAt.Add(cr.Spec.RotationInterval.Duration).After(r.now())
	caCert, err := cl.ProjectKmsGetCA(ctx, cr.Spec.Project)
	if err != nil {
		return fmt.Errorf("getting project CA certificate: %w", err)
	}
	if !publishedAt.IsZero() {
		if err := refreshRotationSecret(cr, secret, svc, caCert); err != nil {
			return err
		}
	}

	if rotate {
		if !hasPendingCredentials {
			idx := slices.Index(cr.Spec.Usernames, string(secret.Data[getSecretPrefix(cr)+"USERNAME"]))
			secret.Data[serviceUserRotationDesiredUsernameKey] = []byte(cr.Spec.Usernames[(idx+1)%len(cr.Spec.Usernames)])
			secret.Data[serviceUserRotationDesiredPasswordKey] = []byte(r.generatePassword())
		}

		// Commit refreshed active details with the pending pair so password failures
		// cannot block their publication. Reject stale Secrets before every reset.
		if secret.ResourceVersion == "" {
			err = r.Create(ctx, secret)
		} else {
			err = r.Update(ctx, secret)
		}
		if err != nil {
			return fmt.Errorf("saving pending rotation: %w", err)
		}

		result, err := cl.ServiceUserCredentialsModify(ctx, cr.Spec.Project, cr.Spec.ServiceName, string(secret.Data[serviceUserRotationDesiredUsernameKey]), &service.ServiceUserCredentialsModifyIn{
			NewPassword: new(string(secret.Data[serviceUserRotationDesiredPasswordKey])),
			Operation:   service.ServiceUserCredentialsModifyOperationTypeResetCredentials,
		})
		if err != nil {
			return fmt.Errorf("modifying credentials for service user %q: %w", string(secret.Data[serviceUserRotationDesiredUsernameKey]), err)
		}
		svc.Users = result.Users
		secret.Data[getSecretPrefix(cr)+"USERNAME"] = secret.Data[serviceUserRotationDesiredUsernameKey]
		secret.Data[getSecretPrefix(cr)+"PASSWORD"] = secret.Data[serviceUserRotationDesiredPasswordKey]
		if err := refreshRotationSecret(cr, secret, svc, caCert); err != nil {
			return err
		}

		// Start the handover interval after connection details are ready.
		secret.Data[serviceUserRotationPublishedAtKey] = []byte(r.now().Format(time.RFC3339Nano))
	}
	delete(secret.Data, serviceUserRotationDesiredUsernameKey)
	delete(secret.Data, serviceUserRotationDesiredPasswordKey)

	if !equality.Semantic.DeepEqual(orig.Data, secret.Data) || !maps.Equal(orig.Labels, secret.Labels) || !maps.Equal(orig.Annotations, secret.Annotations) {
		if err := r.Update(ctx, secret); err != nil {
			return fmt.Errorf("refreshing published connection details: %w", err)
		}
	}
	publishedAt, err = serviceUserRotationPublishedAt(secret)
	if err != nil {
		return err
	}
	cr.Status.ActiveUsername = string(secret.Data[getSecretPrefix(cr)+"USERNAME"])
	cr.Status.LastRotationAt = metav1.NewTime(publishedAt)
	cr.Status.NextRotationAt = metav1.NewTime(publishedAt.Add(cr.Spec.RotationInterval.Duration))
	return nil
}

// refreshRotationSecret refreshes connection details for the Secret's published username.
func refreshRotationSecret(cr *v1alpha1.ServiceUserRotation, secret *corev1.Secret, svc *service.ServiceGetOut, caCert string) error {
	componentIdx := slices.IndexFunc(svc.Components, func(c service.ComponentOut) bool { return c.Component == svc.ServiceType })
	if componentIdx < 0 {
		return fmt.Errorf("service component %q not found", svc.ServiceType)
	}
	component := &svc.Components[componentIdx]

	prefix := getSecretPrefix(cr)
	username := string(secret.Data[prefix+"USERNAME"])
	idx := slices.IndexFunc(svc.Users, func(u service.UserOut) bool { return u.Username == username })
	if idx < 0 {
		return fmt.Errorf("active user %q not found in service", username)
	}
	user := svc.Users[idx]

	if svc.ServiceType == string(serviceTypeKafka) &&
		(user.AccessCert == nil || *user.AccessCert == "" || user.AccessKey == nil || *user.AccessKey == "") {
		return fmt.Errorf("%w: Kafka user certificate and key are not yet available from the API", errPreconditionNotMet)
	}
	details := SecretDetails{
		prefix + "HOST":        component.Host,
		prefix + "PORT":        strconv.Itoa(component.Port),
		prefix + "USERNAME":    user.Username,
		prefix + "ACCESS_CERT": fromAnyPointer(user.AccessCert),
		prefix + "ACCESS_KEY":  fromAnyPointer(user.AccessKey),
		prefix + "CA_CERT":     caCert,
	}
	if svc.ServiceType == string(serviceTypeKafka) {
		refreshKafkaEndpointDetails(secret.Data, svc.Components, prefix)
	}
	for key, value := range details {
		secret.Data[key] = []byte(value)
	}
	target := cr.GetConnInfoSecretTarget()
	secret.Labels = maps.Clone(target.Labels)
	secret.Annotations = maps.Clone(target.Annotations)
	return nil
}

func (r *ServiceUserRotationReconciler) ensureUsers(ctx context.Context, cr *v1alpha1.ServiceUserRotation, cl avngen.Client, svc *service.ServiceGetOut) error {
	for _, username := range cr.Spec.Usernames {
		if !slices.ContainsFunc(svc.Users, func(user service.UserOut) bool { return user.Username == username }) {
			created, err := cl.ServiceUserCreate(ctx, cr.Spec.Project, cr.Spec.ServiceName, &service.ServiceUserCreateIn{Username: username})
			if err != nil {
				return fmt.Errorf("creating service user %q: %w", username, err)
			}
			svc.Users = append(svc.Users, service.UserOut(*created))
		}
	}

	return nil
}

func (r *ServiceUserRotationReconciler) reconcileDeletion(ctx context.Context, cr *v1alpha1.ServiceUserRotation) (err error) {
	if !controllerutil.ContainsFinalizer(cr, instanceDeletionFinalizer) {
		return nil
	}

	orig := cr.DeepCopy()
	defer func() {
		// Successful finalization may already have deleted the resource.
		if err != nil {
			condition := getErrorCondition("ReconcileFailed", err)
			condition.ObservedGeneration = cr.Generation
			meta.SetStatusCondition(cr.Conditions(), condition)
			r.Recorder.Event(cr, corev1.EventTypeWarning, condition.Reason, err.Error())
			err = errors.Join(err, r.persistStatus(ctx, orig, cr))
		}
	}()

	if policy, hasPolicy := cr.Annotations[deletionPolicyAnnotation]; hasPolicy {
		if policy != deletionPolicyOrphan {
			return fmt.Errorf("invalid deletion policy %q, only %q is allowed", policy, deletionPolicyOrphan)
		}
	} else {
		avnGen, err := r.aivenClient(ctx, cr)
		if err != nil {
			return err
		}

		var errs []error
		for _, username := range cr.Spec.Usernames {
			if err := avnGen.ServiceUserDelete(ctx, cr.Spec.Project, cr.Spec.ServiceName, username); err != nil && !isNotFound(err) {
				errs = append(errs, fmt.Errorf("deleting service user %q: %w", username, err))
			}
		}
		if err := errors.Join(errs...); err != nil {
			return err
		}
	}

	return removeFinalizer(ctx, r.Client, cr, instanceDeletionFinalizer)
}

func (r *ServiceUserRotationReconciler) ensureFinalizer(ctx context.Context, cr *v1alpha1.ServiceUserRotation) error {
	if controllerutil.ContainsFinalizer(cr, instanceDeletionFinalizer) {
		return nil
	}
	return addFinalizer(ctx, r.Client, cr, instanceDeletionFinalizer)
}

func (r *ServiceUserRotationReconciler) ensureSecretFinalizer(ctx context.Context, cr *v1alpha1.ServiceUserRotation) error {
	if r.DefaultToken != "" || cr.AuthSecretRef() == nil {
		return nil
	}

	secret, err := r.getAuthSecret(ctx, cr)
	if err != nil {
		return err
	}
	if controllerutil.ContainsFinalizer(secret, secretProtectionFinalizer) {
		return nil
	}
	return addFinalizer(ctx, r.Client, secret, secretProtectionFinalizer)
}

func (r *ServiceUserRotationReconciler) aivenClient(ctx context.Context, cr *v1alpha1.ServiceUserRotation) (avngen.Client, error) {
	var token string
	switch {
	case r.DefaultToken != "":
		token = r.DefaultToken
	case cr.AuthSecretRef() == nil:
		return nil, errNoTokenProvided
	default:
		secret, err := r.getAuthSecret(ctx, cr)
		if err != nil {
			return nil, err
		}
		token = string(secret.Data[cr.AuthSecretRef().Key])
	}

	return r.Controller.newAivenClient(token, r.KubeVersion, r.OperatorVersion)
}

func (r *ServiceUserRotationReconciler) getAuthSecret(ctx context.Context, cr *v1alpha1.ServiceUserRotation) (*corev1.Secret, error) {
	auth := cr.AuthSecretRef()
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: auth.Name, Namespace: cr.Namespace}, secret); err != nil {
		return nil, fmt.Errorf("cannot get secret %q: %w", auth.Name, err)
	}
	return secret, nil
}
