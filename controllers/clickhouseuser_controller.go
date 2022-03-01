// Copyright (c) 2022 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/aiven/aiven-go-client"
	"github.com/aiven/aiven-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ClickhouseUserReconciler reconciles a ClickhouseUser object
type ClickhouseUserReconciler struct {
	Controller
}

//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=aiven.io,resources=clickhouseusers/status,verbs=get;update;patch

func (r *ClickhouseUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("clickhouseuser", req.NamespacedName)

	// fetch the clickhouse userinstance
	user := &v1alpha1.ClickhouseUser{}
	err := r.Get(ctx, req.NamespacedName, user)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not token, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("clickhouse user resource not token. ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "failed to get clickhouse user")
		return ctrl.Result{}, err
	}

	r.Controller.Recorder.Event(user, corev1.EventTypeNormal, eventReconciliationStarted, "starting reconciliation")

	clientAuthSecret := &corev1.Secret{}
	if err := r.Controller.Get(ctx, types.NamespacedName{Name: user.AuthSecretRef().Name, Namespace: req.Namespace}, clientAuthSecret); err != nil {
		r.Controller.Recorder.Eventf(user, corev1.EventTypeWarning, eventUnableToGetAuthSecret, err.Error())
		return ctrl.Result{}, fmt.Errorf("cannot get secret %q: %w", user.AuthSecretRef().Name, err)
	}

	avn, err := aiven.NewTokenClient(string(clientAuthSecret.Data[user.AuthSecretRef().Key]), "k8s-operator/")
	if err != nil {
		r.Controller.Recorder.Event(user, corev1.EventTypeWarning, eventUnableToCreateClient, err.Error())
		return ctrl.Result{}, fmt.Errorf("cannot initialize aiven client: %w", err)
	}

	// check if the clickhouse User instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isCHUserMarkedToBeDeleted := user.GetDeletionTimestamp() != nil
	if isCHUserMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(user, instanceDeletionFinalizer) {
			// run finalization logic for instanceDeletionFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			r.Controller.Recorder.Event(user, corev1.EventTypeNormal, eventTryingToDeleteAtAiven, "deleting clickhouse user on aiven side")
			if len(user.Status.UUID) > 0 {
				if err := avn.ClickhouseUser.Delete(user.Spec.Project, user.Spec.ServiceName, user.Status.UUID); err != nil {
					r.Controller.Recorder.Event(user, corev1.EventTypeWarning, eventUnableToDeleteAtAiven, err.Error())
					return reconcile.Result{}, err
				}
			}
			r.Controller.Recorder.Event(user, corev1.EventTypeNormal, eventSuccessfullyDeletedAtAiven, "clickhouse user was deleted on aiven side")

			// remove instanceDeletionFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(user, instanceDeletionFinalizer)
			err := r.Client.Update(ctx, user)
			if err != nil {
				r.Controller.Recorder.Event(user, corev1.EventTypeWarning, eventUnableToDeleteFinalizer, err.Error())
				return reconcile.Result{}, err
			}
		}

		return reconcile.Result{}, nil
	}

	// add finalizer for this CR
	if !controllerutil.ContainsFinalizer(user, instanceDeletionFinalizer) {
		if err := addFinalizer(ctx, r.Client, user, instanceDeletionFinalizer); err != nil {
			r.Controller.Recorder.Event(user, corev1.EventTypeWarning, eventUnableToAddFinalizer, err.Error())
			return reconcile.Result{}, err
		}
		r.Controller.Recorder.Event(user, corev1.EventTypeNormal, eventAddedFinalizer, "added to clickhouse user instance")
	}

	var password string

	ready, err := checkPreconditions(avn, user)
	if err != nil {
		log.Error(err, "failed to check clickhouse user preconditions")
		return ctrl.Result{}, err
	}

	if !ready {
		log.Info("clickuse user prorecontions are not met, requeue in a 10s")
		return ctrl.Result{Requeue: true, RequeueAfter: formatIntBaseDecimal * time.Second}, nil
	}

	// check if Clickhouse User already exists on the Aiven side, create a
	// new one if it is not found
	uuid, err := r.isAlreadyExists(avn, user)
	if err != nil {
		log.Error(err, "failed to check is clickhouse user exists")
		return ctrl.Result{}, err
	}

	if !isAlreadyProcessed(user) {
		if len(uuid) > 0 {
			pass, err := r.generatePassword()
			if err != nil {
				log.Error(err, "failed to generate a clickhouse user password")
				return ctrl.Result{}, err
			}

			p, err := avn.ClickhouseUser.ResetPassword(user.Spec.Project, user.Spec.ServiceName, uuid, pass)
			if err != nil {
				r.Controller.Recorder.Event(user, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
				log.Error(err, "failed to reset clickhouse user password")
				return ctrl.Result{}, err
			}
			password = p
		} else {
			u, err := avn.ClickhouseUser.Create(user.Spec.Project, user.Spec.ServiceName, user.Name)
			if err != nil {
				r.Controller.Recorder.Event(user, corev1.EventTypeWarning, eventUnableToCreateOrUpdateAtAiven, err.Error())
				log.Error(err, "failed to create a clickhouse user")
				return ctrl.Result{}, err
			}
			uuid = u.User.UUID
			password = u.User.Password

			meta.SetStatusCondition(&user.Status.Conditions,
				getInitializedCondition("Updated",
					"Instance was updated on Aiven side"))
		}

		// marking generation as processed and running
		metav1.SetMetaDataAnnotation(&user.ObjectMeta,
			processedGenerationAnnotation, strconv.FormatInt(user.GetGeneration(), formatIntBaseDecimal))
		metav1.SetMetaDataAnnotation(&user.ObjectMeta, instanceIsRunningAnnotation, "true")

		// setting status conditions to running
		meta.SetStatusCondition(&user.Status.Conditions,
			getRunningCondition(metav1.ConditionTrue, "CheckRunning",
				"Instance is running on Aiven side"))

		// creation of a secret
		err := r.createSecret(ctx, avn, user, password)
		if err != nil {
			log.Error(err, "failed to create a clickhouse user secret")
			return ctrl.Result{}, err
		}

		// updating clickhouse user resource status
		user.Status.UUID = uuid
		err = r.Status().Update(context.Background(), user)
		if err != nil {
			log.Error(err, "failed to update a clickhouse user cr status")
			return ctrl.Result{}, err
		}
	}

	r.Controller.Recorder.Event(user, corev1.EventTypeNormal, eventInstanceIsRunning, "clickhouse user is in a RUNNING state")
	log.Info("clickhouse user instance was successfully reconciled")

	return ctrl.Result{}, nil
}

func (*ClickhouseUserReconciler) generatePassword() (string, error) {
	c := 16
	b := make([]byte, c)
	_, err := rand.Read(b)
	return fmt.Sprintf("%x", b), err
}

func (r *ClickhouseUserReconciler) createSecret(ctx context.Context, avn *aiven.Client, user *v1alpha1.ClickhouseUser, password string) error {
	s, err := avn.Services.Get(user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return fmt.Errorf("cannot get a clickhouse service %w", err)
	}

	params := s.URIParams

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.getSecretName(user),
			Namespace: user.Namespace,
		},
		StringData: map[string]string{
			"HOST":     params["host"],
			"PORT":     params["port"],
			"PASSWORD": password,
			"USERNAME": user.Name,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		return ctrl.SetControllerReference(user, secret, r.Scheme)
	})

	return err
}

func (*ClickhouseUserReconciler) isAlreadyExists(avn *aiven.Client, user *v1alpha1.ClickhouseUser) (string, error) {
	l, err := avn.ClickhouseUser.List(user.Spec.Project, user.Spec.ServiceName)
	if err != nil {
		return "", err
	}

	for _, u := range l.Users {
		if u.Name == user.Name {
			return u.UUID, nil
		}
	}

	return "", nil
}

func checkPreconditions(avn *aiven.Client, user *v1alpha1.ClickhouseUser) (bool, error) {
	meta.SetStatusCondition(&user.Status.Conditions,
		getInitializedCondition("Preconditions", "Checking preconditions"))

	return checkServiceIsRunning(avn, user.Spec.Project, user.Spec.ServiceName)
}

func (r *ClickhouseUserReconciler) getSecretName(user *v1alpha1.ClickhouseUser) string {
	if user.Spec.ConnInfoSecretTarget.Name != "" {
		return user.Spec.ConnInfoSecretTarget.Name
	}
	return user.Name
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClickhouseUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClickhouseUser{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
