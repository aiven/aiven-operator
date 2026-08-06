package controllers

import (
	"errors"
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/organization"
	"github.com/aiven/go-client-codegen/handler/organizationprojects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestOrganizationProjectReconciler(t *testing.T) {
	t.Parallel()

	newOrganizationProject := func(t *testing.T) *v1alpha1.OrganizationProject {
		t.Helper()
		op := newObjectFromExampleYAML[v1alpha1.OrganizationProject](t, "organizationproject")
		op.Namespace = "default"
		return op
	}

	const parentAccountID = "a123456789a"

	expectParentIDResolution := func(op *v1alpha1.OrganizationProject, avn *avngen.MockClient) {
		avn.EXPECT().
			OrganizationGet(mock.Anything, op.Spec.ParentID).
			Return(&organization.OrganizationGetOut{AccountId: parentAccountID}, nil).Once()
	}

	remoteFromSpec := func(op *v1alpha1.OrganizationProject) *organizationprojects.OrganizationProjectsGetOut {
		emails := make([]organizationprojects.TechEmailOut, 0, len(op.Spec.TechnicalEmails))
		for _, e := range op.Spec.TechnicalEmails {
			emails = append(emails, organizationprojects.TechEmailOut{Email: e})
		}
		return &organizationprojects.OrganizationProjectsGetOut{
			OrganizationId: op.Spec.OrganizationID,
			ProjectId:      op.Spec.ProjectID,
			ParentId:       parentAccountID,
			BillingGroupId: &op.Spec.BillingGroupID,
			BasePort:       op.Spec.BasePort,
			Tags:           op.Spec.Tags,
			TechEmails:     emails,
		}
	}

	runScenarioErr := func(t *testing.T, op *v1alpha1.OrganizationProject, avn avngen.Client) (*Reconciler[*v1alpha1.OrganizationProject], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		r := newOrganizationProjectReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.OrganizationProject{}).
				WithObjects([]client.Object{op}...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(20),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.OrganizationProject])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{Name: op.Name, Namespace: op.Namespace},
		})
		return r, res, err
	}

	runScenario := func(t *testing.T, op *v1alpha1.OrganizationProject, avn avngen.Client) (*Reconciler[*v1alpha1.OrganizationProject], ctrlruntime.Result) {
		t.Helper()

		r, res, err := runScenarioErr(t, op, avn)
		require.NoError(t, err)
		return r, res
	}

	t.Run("Creates organization project on Aiven when it doesn't exist", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		// Seed the running annotation so the assertion below genuinely verifies that
		// Create clears it (delete(cr.GetAnnotations(), instanceIsRunningAnnotation)).
		op.Annotations = map[string]string{
			instanceIsRunningAnnotation: "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(nil, newAivenError(404, "project not found")).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsCreate(mock.Anything, op.Spec.OrganizationID, mock.MatchedBy(func(in *organizationprojects.OrganizationProjectsCreateIn) bool {
				return in.ProjectId == op.Spec.ProjectID &&
					in.BillingGroupId == op.Spec.BillingGroupID &&
					in.ParentId != nil && *in.ParentId == parentAccountID &&
					in.BasePort != nil && *in.BasePort == *op.Spec.BasePort &&
					assert.Equal(t, op.Spec.Tags, in.Tags) &&
					in.TechEmails != nil &&
					assert.Equal(t, organizationProjectTechEmails(op.Spec.TechnicalEmails), *in.TechEmails)
			})).
			Return(&organizationprojects.OrganizationProjectsCreateOut{ProjectId: op.Spec.ProjectID}, nil).Once()

		r, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionUnknown, condition.Status)

		secret := &corev1.Secret{}
		err := r.Get(t.Context(), types.NamespacedName{Name: op.Spec.ConnInfoSecretTarget.Name, Namespace: op.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Requeues without hard error on transient server error during create", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(nil, newAivenError(404, "project not found")).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsCreate(mock.Anything, op.Spec.OrganizationID, mock.Anything).
			Return(nil, newAivenError(500, "temporary create failure")).Once()

		r, res, err := runScenarioErr(t, op, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Nil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))
	})

	t.Run("Marks running and writes CA cert secret when project exists and is up to date", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(remoteFromSpec(op), nil).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, op.Spec.ProjectID).
			Return("ca-cert", nil).Once()

		r, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionTrue, condition.Status)

		secret := &corev1.Secret{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Spec.ConnInfoSecretTarget.Name, Namespace: op.Namespace}, secret))
		require.Equal(t, "ca-cert", string(secret.Data["ORGANIZATIONPROJECT_CA_CERT"]))
	})

	t.Run("Updates existing organization project when generation changed", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 2
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(&organizationprojects.OrganizationProjectsGetOut{
				OrganizationId: op.Spec.OrganizationID,
				ProjectId:      op.Spec.ProjectID,
				ParentId:       op.Spec.ParentID,
			}, nil).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsUpdate(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID, mock.MatchedBy(func(in *organizationprojects.OrganizationProjectsUpdateIn) bool {
				return in.BillingGroupId != nil && *in.BillingGroupId == op.Spec.BillingGroupID &&
					in.ParentId != nil && *in.ParentId == parentAccountID &&
					// projectId is immutable, so no rename must ever be sent.
					in.ProjectName == nil &&
					in.BasePort != nil && *in.BasePort == *op.Spec.BasePort &&
					in.Tags != nil && assert.Equal(t, op.Spec.Tags, *in.Tags) &&
					in.TechEmails != nil &&
					assert.Equal(t, organizationProjectTechEmails(op.Spec.TechnicalEmails), *in.TechEmails)
			})).
			Return(&organizationprojects.OrganizationProjectsUpdateOut{ProjectId: op.Spec.ProjectID}, nil).Once()

		r, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.Equal(t, "2", got.Annotations[processedGenerationAnnotation])
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionUnknown, condition.Status)
	})

	t.Run("Deletes organization project and removes finalizer on deletion", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		op.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsDelete(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(nil).Once()

		r, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.OrganizationProject{}
		err := r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Ignores not found on deletion", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		op.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsDelete(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(newAivenError(404, "not found")).Once()

		r, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.OrganizationProject{}
		err := r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Sets error condition on non-retryable create failure", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(nil, newAivenError(404, "project not found")).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsCreate(mock.Anything, op.Spec.OrganizationID, mock.Anything).
			Return(nil, newAivenError(400, "bad request")).Once()

		r, res, err := runScenarioErr(t, op, avn)
		require.Error(t, err)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
		require.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))
	})

	t.Run("Sends empty technical emails list when none are configured", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Spec.TechnicalEmails = nil

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(nil, newAivenError(404, "project not found")).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsCreate(mock.Anything, op.Spec.OrganizationID, mock.MatchedBy(func(in *organizationprojects.OrganizationProjectsCreateIn) bool {
				return in.TechEmails != nil && len(*in.TechEmails) == 0
			})).
			Return(&organizationprojects.OrganizationProjectsCreateOut{ProjectId: op.Spec.ProjectID}, nil).Once()

		_, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
	})

	t.Run("Sends empty tags map when none are configured", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Spec.Tags = nil

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(nil, newAivenError(404, "project not found")).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsCreate(mock.Anything, op.Spec.OrganizationID, mock.MatchedBy(func(in *organizationprojects.OrganizationProjectsCreateIn) bool {
				return in.Tags != nil && len(in.Tags) == 0
			})).
			Return(&organizationprojects.OrganizationProjectsCreateOut{ProjectId: op.Spec.ProjectID}, nil).Once()

		_, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
	})

	t.Run("Reverts out-of-band tags with an empty map when spec has no tags", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}
		op.Spec.Tags = nil

		// Someone added tags directly in the Aiven console; the spec has none, so the drift must be reverted.
		drifted := remoteFromSpec(op)
		drifted.Tags = map[string]string{"env": "hijacked"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(drifted, nil).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsUpdate(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID, mock.MatchedBy(func(in *organizationprojects.OrganizationProjectsUpdateIn) bool {
				// Clearing tags must send a non-nil.
				return in.Tags != nil && *in.Tags != nil && len(*in.Tags) == 0
			})).
			Return(&organizationprojects.OrganizationProjectsUpdateOut{ProjectId: op.Spec.ProjectID}, nil).Once()

		_, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
	})

	t.Run("Passes account-form parentId through without resolving it", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Spec.ParentID = parentAccountID

		// No OrganizationGet expectation: account/unit IDs must be sent as-is.
		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(nil, newAivenError(404, "project not found")).Once()
		avn.EXPECT().
			OrganizationProjectsCreate(mock.Anything, op.Spec.OrganizationID, mock.MatchedBy(func(in *organizationprojects.OrganizationProjectsCreateIn) bool {
				return in.ParentId != nil && *in.ParentId == parentAccountID
			})).
			Return(&organizationprojects.OrganizationProjectsCreateOut{ProjectId: op.Spec.ProjectID}, nil).Once()

		_, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
	})

	t.Run("Requeues without hard error on transient server error during update", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 2
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(&organizationprojects.OrganizationProjectsGetOut{
				OrganizationId: op.Spec.OrganizationID,
				ProjectId:      op.Spec.ProjectID,
				ParentId:       op.Spec.ParentID,
			}, nil).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsUpdate(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID, mock.Anything).
			Return(nil, newAivenError(500, "temporary update failure")).Once()

		r, res, err := runScenarioErr(t, op, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.NotEqual(t, "2", got.Annotations[processedGenerationAnnotation])
		require.Nil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))
	})

	t.Run("Stays up to date when Aiven echoes the account form of parentId", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
		}

		avn := avngen.NewMockClient(t)
		// remoteFromSpec returns parent_id in the account form (a...) while the spec
		// uses the org... form: the normalization echo must NOT be treated as drift.
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(remoteFromSpec(op), nil).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, op.Spec.ProjectID).
			Return("ca-cert", nil).Once()

		r, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		condition := meta.FindStatusCondition(got.Status.Conditions, conditionTypeRunning)
		require.NotNil(t, condition)
		require.Equal(t, metav1.ConditionTrue, condition.Status)
	})

	t.Run("Stays up to date when Aiven lowercases an email domain", func(t *testing.T) {
		// Aiven lowercases the domain, so the echo must not read as drift.
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
		}
		op.Spec.TechnicalEmails = []string{"Ops.Team@Example.COM"}

		remote := remoteFromSpec(op)
		remote.TechEmails = []organizationprojects.TechEmailOut{{Email: "Ops.Team@example.com"}}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(remote, nil).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, op.Spec.ProjectID).
			Return("ca-cert", nil).Once()
		// No OrganizationProjectsUpdate expectation: the domain-case echo must not
		// be read as drift. The mock fails the test if Update is called.

		r, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Stays up to date when Aiven collapses domain-case duplicates", func(t *testing.T) {
		// Byte-wise unique, so the CEL rule allows it, but Aiven stores one entry.
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
		}
		op.Spec.TechnicalEmails = []string{"dup@Example.com", "dup@example.com"}

		remote := remoteFromSpec(op)
		remote.TechEmails = []organizationprojects.TechEmailOut{{Email: "dup@example.com"}}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(remote, nil).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, op.Spec.ProjectID).
			Return("ca-cert", nil).Once()
		// No Update expectation: the collapse must not read as drift.

		r, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("Treats a local-part case difference as real drift", func(t *testing.T) {
		// Aiven keeps these as two addresses, so folding the local part would
		// wrongly report in-sync. Guards against EqualFold creeping in.
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
		}
		op.Spec.TechnicalEmails = []string{"Ops@example.com"}

		remote := remoteFromSpec(op)
		remote.TechEmails = []organizationprojects.TechEmailOut{{Email: "ops@example.com"}}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(remote, nil).Once()
		// Only one resolution (in Update): the drift check short-circuits on the
		// email mismatch before reaching the parent_id comparison.
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsUpdate(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID, mock.MatchedBy(func(in *organizationprojects.OrganizationProjectsUpdateIn) bool {
				return in.TechEmails != nil && assert.Equal(t, organizationProjectTechEmails(op.Spec.TechnicalEmails), *in.TechEmails)
			})).
			Return(&organizationprojects.OrganizationProjectsUpdateOut{ProjectId: op.Spec.ProjectID}, nil).Once()

		_, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
	})

	t.Run("Reverts out-of-band remote changes even when generation is processed", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		// Someone changed tags directly in the Aiven console.
		drifted := remoteFromSpec(op)
		drifted.Tags = map[string]string{"env": "hijacked"}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(drifted, nil).Once()
		// Only one resolution (in Update): the drift check short-circuits on the
		// tags mismatch before reaching the parent_id comparison.
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsUpdate(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID, mock.MatchedBy(func(in *organizationprojects.OrganizationProjectsUpdateIn) bool {
				return in.Tags != nil && assert.Equal(t, op.Spec.Tags, *in.Tags)
			})).
			Return(&organizationprojects.OrganizationProjectsUpdateOut{ProjectId: op.Spec.ProjectID}, nil).Once()

		_, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
	})

	t.Run("Reverts an out-of-band parent move", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
			instanceIsRunningAnnotation:   "true",
		}

		// The project was moved to another organizational unit outside Kubernetes.
		drifted := remoteFromSpec(op)
		drifted.ParentId = "a-other-unit-id"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(drifted, nil).Once()
		// One resolution in the drift check (Observe) and one in Update.
		expectParentIDResolution(op, avn)
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			OrganizationProjectsUpdate(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID, mock.MatchedBy(func(in *organizationprojects.OrganizationProjectsUpdateIn) bool {
				return in.ParentId != nil && *in.ParentId == parentAccountID
			})).
			Return(&organizationprojects.OrganizationProjectsUpdateOut{ProjectId: op.Spec.ProjectID}, nil).Once()

		_, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)
	})

	t.Run("Requeues on transient server error during Get", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(nil, newAivenError(500, "temporary get failure")).Once()

		r, res, err := runScenarioErr(t, op, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.NotEqual(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Does not mark running when CA fetch fails", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
		}

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(remoteFromSpec(op), nil).Once()
		expectParentIDResolution(op, avn)
		avn.EXPECT().
			ProjectKmsGetCA(mock.Anything, op.Spec.ProjectID).
			Return("", errors.New("ca unavailable")).Once()

		r, res, err := runScenarioErr(t, op, avn)
		require.ErrorContains(t, err, "getting project KMS CA: ca unavailable")
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.NotContains(t, got.Annotations, instanceIsRunningAnnotation)
		require.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))

		secret := &corev1.Secret{}
		err = r.Get(t.Context(), types.NamespacedName{Name: op.Spec.ConnInfoSecretTarget.Name, Namespace: op.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Skips the CA fetch and writes no secret when connection secret is disabled", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Annotations = map[string]string{
			processedGenerationAnnotation: "1",
		}
		disabled := true
		op.Spec.ConnInfoSecretTargetDisabled = &disabled

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsGet(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(remoteFromSpec(op), nil).Once()
		expectParentIDResolution(op, avn)
		// No ProjectKmsGetCA expectation: the cert only feeds the connection secret,
		// so the call must not be made at all. The mock fails on an unexpected call.

		r, res := runScenario(t, op, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		secret := &corev1.Secret{}
		err := r.Get(t.Context(), types.NamespacedName{Name: op.Spec.ConnInfoSecretTarget.Name, Namespace: op.Namespace}, secret)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Retains finalizer and requeues on transient server error during delete", func(t *testing.T) {
		op := newOrganizationProject(t)
		op.Generation = 1
		op.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		op.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			OrganizationProjectsDelete(mock.Anything, op.Spec.OrganizationID, op.Spec.ProjectID).
			Return(newAivenError(500, "temporary delete failure")).Once()

		r, res, err := runScenarioErr(t, op, avn)
		require.NoError(t, err)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := &v1alpha1.OrganizationProject{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: op.Name, Namespace: op.Namespace}, got))
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		require.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, ConditionTypeError))
	})
}

func TestNormalizeTechEmails(t *testing.T) {
	t.Parallel()

	// Behaviors verified against the live API.
	cases := []struct {
		name string
		in   []string
		want []string
	}{{
		name: "lowercases the domain but not the local part",
		in:   []string{"MixedCase.Probe@Example.COM"},
		want: []string{"MixedCase.Probe@example.com"},
	}, {
		name: "keeps local-part case differences as distinct addresses",
		in:   []string{"Case.Dup@example.com", "case.dup@example.com"},
		want: []string{"Case.Dup@example.com", "case.dup@example.com"},
	}, {
		name: "collapses domain-case duplicates",
		in:   []string{"domain.dup@Example.com", "domain.dup@example.com"},
		want: []string{"domain.dup@example.com"},
	}, {
		name: "drops exact duplicates",
		in:   []string{"dup@example.com", "dup@example.com"},
		want: []string{"dup@example.com"},
	}, {
		name: "sorts so order is not drift",
		in:   []string{"b@example.com", "a@example.com"},
		want: []string{"a@example.com", "b@example.com"},
	}, {
		name: "leaves a value without @ alone",
		in:   []string{"not-an-email"},
		want: []string{"not-an-email"},
	}, {
		name: "handles nil",
		in:   nil,
		want: []string{},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, normalizeTechEmails(tc.in))
		})
	}
}
