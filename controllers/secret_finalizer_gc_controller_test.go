package controllers

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestSecretFinalizerGCEventPredicate(t *testing.T) {
	t.Parallel()

	pred := secretFinalizerGCEventPredicate()
	secret := newObjectFromYAML[corev1.Secret](t, yamlAuthSecret)
	protectedSecret := secret.DeepCopy()
	protectedSecret.Finalizers = []string{secretProtectionFinalizer}
	deletingProtectedSecret := protectedSecret.DeepCopy()
	now := metav1.Now()
	deletingProtectedSecret.DeletionTimestamp = &now
	deletingUnprotectedSecret := secret.DeepCopy()
	deletingUnprotectedSecret.DeletionTimestamp = &now
	deletingProtectedCR := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
	deletingProtectedCR.Finalizers = []string{secretProtectionFinalizer}
	deletingProtectedCR.DeletionTimestamp = &now
	crWithoutAuth := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUser)
	crWithOldAuth := newObjectFromYAML[v1alpha1.ClickhouseUser](t, yamlClickhouseUserWithAuth)
	crWithSameAuth := crWithOldAuth.DeepCopy()
	crWithNewAuth := crWithOldAuth.DeepCopy()
	crWithNewAuth.Spec.AuthSecretRef.Name = "new-token"
	crWithNewAuthKey := crWithOldAuth.DeepCopy()
	crWithNewAuthKey.Spec.AuthSecretRef.Key = "new-token-key"

	require.False(t, pred.Create(event.CreateEvent{Object: secret}))
	require.False(t, pred.Create(event.CreateEvent{Object: protectedSecret}))
	require.False(t, pred.Create(event.CreateEvent{Object: deletingUnprotectedSecret}))
	require.True(t, pred.Create(event.CreateEvent{Object: deletingProtectedSecret}))
	require.False(t, pred.Create(event.CreateEvent{Object: deletingProtectedCR}))
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: protectedSecret, ObjectNew: protectedSecret.DeepCopy()}))
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: protectedSecret, ObjectNew: deletingProtectedSecret}))
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: secret, ObjectNew: deletingUnprotectedSecret}))
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: deletingProtectedCR.DeepCopy(), ObjectNew: deletingProtectedCR}))
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: crWithOldAuth, ObjectNew: crWithSameAuth}))
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: crWithOldAuth, ObjectNew: crWithNewAuthKey}))
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: crWithOldAuth, ObjectNew: crWithNewAuth}))
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: crWithoutAuth, ObjectNew: crWithOldAuth}))
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: crWithOldAuth, ObjectNew: crWithoutAuth}))
	require.True(t, pred.Delete(event.DeleteEvent{Object: deletingProtectedSecret}))
	require.False(t, pred.Generic(event.GenericEvent{Object: deletingProtectedSecret}))
}
