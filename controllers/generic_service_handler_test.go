package controllers

import (
	"net/http"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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
		return &genericServiceHandler{fabric: newPostgreSQLAdapter}
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
