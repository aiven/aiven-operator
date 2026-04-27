package controllers

import (
	"net/http"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestGenericServiceHandlerDelete(t *testing.T) {
	t.Parallel()

	newPG := func(tp *bool) *v1alpha1.PostgreSQL {
		pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		pg.Spec.TerminationProtection = tp
		return pg
	}

	newHandler := func() *genericServiceHandler {
		return &genericServiceHandler{fabric: newPostgreSQLAdapterFactory(nil)}
	}

	t.Run("Blocks deletion when termination protection is enabled in spec", func(t *testing.T) {
		t.Parallel()

		enabled := true
		pg := newPG(&enabled)

		avn := avngen.NewMockClient(t)
		// No API calls expected — the K8s-side guard should block before any call.

		h := newHandler()
		finalised, err := h.delete(t.Context(), avn, pg)
		require.False(t, finalised)
		require.ErrorIs(t, err, errTerminationProtectionOn)
	})

	t.Run("Disables termination protection on Aiven before deletion", func(t *testing.T) {
		t.Parallel()

		disabled := false
		pg := newPG(&disabled)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, &service.ServiceUpdateIn{
				TerminationProtection: &disabled,
			}).
			Return(nil, nil).Once()
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil).Once()

		h := newHandler()
		finalised, err := h.delete(t.Context(), avn, pg)
		require.True(t, finalised)
		require.NoError(t, err)
	})

	t.Run("Deletes service when termination protection is nil in spec", func(t *testing.T) {
		t.Parallel()

		pg := newPG(nil)

		disabled := false
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, &service.ServiceUpdateIn{
				TerminationProtection: &disabled,
			}).
			Return(nil, nil).Once()
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil).Once()

		h := newHandler()
		finalised, err := h.delete(t.Context(), avn, pg)
		require.True(t, finalised)
		require.NoError(t, err)
	})

	t.Run("Skips ServiceUpdate 404 and proceeds with deletion", func(t *testing.T) {
		t.Parallel()

		disabled := false
		pg := newPG(&disabled)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
			Return(nil, avngen.Error{Status: http.StatusNotFound, Message: "Service not found"}).Once()
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil).Once()

		h := newHandler()
		finalised, err := h.delete(t.Context(), avn, pg)
		require.True(t, finalised)
		require.NoError(t, err)
	})

	t.Run("Returns error when ServiceUpdate fails with non-404", func(t *testing.T) {
		t.Parallel()

		disabled := false
		pg := newPG(&disabled)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
			Return(nil, avngen.Error{Status: http.StatusForbidden, Message: "Forbidden"}).Once()
		// ServiceDelete should NOT be called.

		h := newHandler()
		finalised, err := h.delete(t.Context(), avn, pg)
		require.False(t, finalised)
		require.Error(t, err, "expected error when ServiceUpdate fails with non-404")
	})

	t.Run("Returns error when ServiceDelete fails", func(t *testing.T) {
		t.Parallel()

		disabled := false
		pg := newPG(&disabled)

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, mock.Anything).
			Return(nil, nil).Once()
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(avngen.Error{Status: http.StatusInternalServerError, Message: "Internal error"}).Once()

		h := newHandler()
		finalised, err := h.delete(t.Context(), avn, pg)
		require.False(t, finalised)
		require.Error(t, err, "expected error when ServiceDelete fails")
	})

	// When spec.terminationProtection=false but Aiven still has it enabled,
	// ServiceDelete would fail. Without the proactive ServiceUpdate call,
	// the controller would be stuck in an infinite retry loop.
	t.Run("Ensures ServiceUpdate disables TP before ServiceDelete is called", func(t *testing.T) {
		t.Parallel()

		disabled := false
		pg := newPG(&disabled)

		avn := avngen.NewMockClient(t)
		updateCall := avn.EXPECT().
			ServiceUpdate(mock.Anything, pg.Spec.Project, pg.Name, &service.ServiceUpdateIn{
				TerminationProtection: &disabled,
			}).
			Return(nil, nil).Once()
		avn.EXPECT().
			ServiceDelete(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil).Once().
			NotBefore(updateCall)

		h := newHandler()
		finalised, err := h.delete(t.Context(), avn, pg)
		require.True(t, finalised)
		require.NoError(t, err)
	})
}

func TestUpdateMigrationStatus(t *testing.T) {
	t.Parallel()

	newPGWithMigration := func() (*v1alpha1.PostgreSQL, serviceAdapter) {
		pg := newObjectFromYAML[v1alpha1.PostgreSQL](t, yamlPostgres)
		pg.Spec.MigrationSecretSource = &v1alpha1.MigrationSecretSource{Name: "creds"}
		adapter := &postgreSQLAdapter{PostgreSQL: pg, k8s: nil}
		return pg, adapter
	}

	h := &genericServiceHandler{
		fabric: newPostgreSQLAdapterFactory(nil),
		log:    logr.Discard(),
	}

	t.Run("Sets MigrationDone on 404 (no active migration)", func(t *testing.T) {
		t.Parallel()

		pg, adapter := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil, avngen.Error{Status: http.StatusNotFound, Message: "not found"}).Once()

		require.NoError(t, h.updateMigrationStatus(t.Context(), avn, adapter, &pg.Spec.ServiceCommonSpec))

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.MigrationReasonDone, cond.Reason)
	})

	t.Run("Sets MigrationDone when status is done", func(t *testing.T) {
		t.Parallel()

		pg, adapter := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetMigrationStatusOut{
				Migration: service.MigrationOut{Status: service.MigrationStatusTypeDone},
			}, nil).Once()

		require.NoError(t, h.updateMigrationStatus(t.Context(), avn, adapter, &pg.Spec.ServiceCommonSpec))

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, v1alpha1.MigrationReasonDone, cond.Reason)
	})

	t.Run("Sets MigrationFailed with error message", func(t *testing.T) {
		t.Parallel()

		pg, adapter := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetMigrationStatusOut{
				Migration: service.MigrationOut{
					Status: service.MigrationStatusTypeFailed,
					Error:  new("connection refused"),
				},
			}, nil).Once()

		require.NoError(t, h.updateMigrationStatus(t.Context(), avn, adapter, &pg.Spec.ServiceCommonSpec))

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, v1alpha1.MigrationReasonFailed, cond.Reason)
		assert.Equal(t, "connection refused", cond.Message)
	})

	t.Run("Sets MigrationInProgress for other statuses", func(t *testing.T) {
		t.Parallel()

		pg, adapter := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(&service.ServiceGetMigrationStatusOut{
				Migration: service.MigrationOut{Status: "syncing"},
			}, nil).Once()

		require.NoError(t, h.updateMigrationStatus(t.Context(), avn, adapter, &pg.Spec.ServiceCommonSpec))

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, v1alpha1.MigrationReasonInProgress, cond.Reason)
	})

	t.Run("Returns error on non-404 error", func(t *testing.T) {
		t.Parallel()

		pg, adapter := newPGWithMigration()
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			ServiceGetMigrationStatus(mock.Anything, pg.Spec.Project, pg.Name).
			Return(nil, avngen.Error{Status: http.StatusInternalServerError, Message: "server error"}).Once()

		err := h.updateMigrationStatus(t.Context(), avn, adapter, &pg.Spec.ServiceCommonSpec)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server error")

		cond := meta.FindStatusCondition(pg.Status.Conditions, v1alpha1.ConditionTypeMigrationComplete)
		assert.Nil(t, cond)
	})
}

func TestHasPendingMigration(t *testing.T) {
	t.Parallel()

	t.Run("No condition — returns false", func(t *testing.T) {
		t.Parallel()
		pg := &v1alpha1.PostgreSQL{}
		assert.False(t, hasPendingMigration(pg))
	})

	t.Run("Condition True — returns false", func(t *testing.T) {
		t.Parallel()
		pg := &v1alpha1.PostgreSQL{}
		meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.ConditionTypeMigrationComplete,
			Status: metav1.ConditionTrue,
			Reason: v1alpha1.MigrationReasonDone,
		})
		assert.False(t, hasPendingMigration(pg))
	})

	t.Run("Condition False (in progress) — returns true", func(t *testing.T) {
		t.Parallel()
		pg := &v1alpha1.PostgreSQL{}
		meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.ConditionTypeMigrationComplete,
			Status: metav1.ConditionFalse,
			Reason: v1alpha1.MigrationReasonInProgress,
		})
		assert.True(t, hasPendingMigration(pg))
	})

	t.Run("Condition False (failed) — returns false (terminal state)", func(t *testing.T) {
		t.Parallel()
		pg := &v1alpha1.PostgreSQL{}
		meta.SetStatusCondition(&pg.Status.Conditions, metav1.Condition{
			Type:   v1alpha1.ConditionTypeMigrationComplete,
			Status: metav1.ConditionFalse,
			Reason: v1alpha1.MigrationReasonFailed,
		})
		assert.False(t, hasPendingMigration(pg))
	})
}
