package controllers

import (
	"testing"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/aiven/go-client-codegen/handler/vpc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

func TestProjectVPCReconciler(t *testing.T) {
	t.Parallel()

	newProjectVPC := func(t *testing.T) *v1alpha1.ProjectVPC {
		t.Helper()
		vpcObj := newObjectFromExampleYAML[v1alpha1.ProjectVPC](t, "projectvpc")
		vpcObj.Namespace = "default"
		return vpcObj
	}

	runScenarioErr := func(t *testing.T, vpcObj *v1alpha1.ProjectVPC, avn avngen.Client) (*Reconciler[*v1alpha1.ProjectVPC], ctrlruntime.Result, error) {
		t.Helper()

		scheme := runtime.NewScheme()
		require.NoError(t, clientgoscheme.AddToScheme(scheme))
		require.NoError(t, v1alpha1.AddToScheme(scheme))

		r := newProjectVPCReconciler(Controller{
			Client: fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&v1alpha1.ProjectVPC{}).
				WithObjects([]client.Object{vpcObj}...).
				Build(),
			Scheme:       scheme,
			Recorder:     record.NewFakeRecorder(10),
			DefaultToken: "test-token",
			PollInterval: testPollInterval,
		}).(*Reconciler[*v1alpha1.ProjectVPC])
		r.newAivenGeneratedClient = func(_, _, _ string) (avngen.Client, error) {
			return avn, nil
		}

		res, err := r.Reconcile(t.Context(), ctrlruntime.Request{
			NamespacedName: types.NamespacedName{Name: vpcObj.Name, Namespace: vpcObj.Namespace},
		})
		return r, res, err
	}

	runScenario := func(t *testing.T, vpcObj *v1alpha1.ProjectVPC, avn avngen.Client) (*Reconciler[*v1alpha1.ProjectVPC], ctrlruntime.Result) {
		t.Helper()

		r, res, err := runScenarioErr(t, vpcObj, avn)
		require.NoError(t, err)
		return r, res
	}

	getVPC := func(t *testing.T, r *Reconciler[*v1alpha1.ProjectVPC], vpcObj *v1alpha1.ProjectVPC) *v1alpha1.ProjectVPC {
		t.Helper()
		got := &v1alpha1.ProjectVPC{}
		require.NoError(t, r.Get(t.Context(), types.NamespacedName{Name: vpcObj.Name, Namespace: vpcObj.Namespace}, got))
		return got
	}

	t.Run("Creates VPC on Aiven when status.ID is empty", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		// Pre-seed the running annotation so the assertion that Create clears it is meaningful.
		metav1.SetMetaDataAnnotation(&vpcObj.ObjectMeta, instanceIsRunningAnnotation, "true")

		avn := avngen.NewMockClient(t)
		// Observe lists VPCs first to adopt any pre-existing match; none here, so it creates.
		avn.EXPECT().
			VpcList(mock.Anything, vpcObj.Spec.Project).
			Return([]vpc.VpcOut{}, nil).Once()
		avn.EXPECT().
			VpcCreate(mock.Anything, vpcObj.Spec.Project, mock.MatchedBy(func(in *vpc.VpcCreateIn) bool {
				return in.CloudName == vpcObj.Spec.CloudName &&
					in.NetworkCidr == vpcObj.Spec.NetworkCidr &&
					len(in.PeeringConnections) == 0
			})).
			Return(&vpc.VpcCreateOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeApproved}, nil).Once()

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getVPC(t, r, vpcObj)
		require.Equal(t, "vpc-id-1", got.Status.ID)
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		// Create clears the pre-seeded running annotation.
		require.NotEqual(t, "true", got.Annotations[instanceIsRunningAnnotation])

		// Create sets Initialized(True) and Running(Unknown) conditions.
		initialized := meta.FindStatusCondition(got.Status.Conditions, "Initialized")
		require.NotNil(t, initialized)
		require.Equal(t, metav1.ConditionTrue, initialized.Status)
		running := meta.FindStatusCondition(got.Status.Conditions, "Running")
		require.NotNil(t, running)
		require.Equal(t, metav1.ConditionUnknown, running.Status)
	})

	t.Run("Adopts existing VPC matching spec when status.ID is empty", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcList(mock.Anything, vpcObj.Spec.Project).
			Return([]vpc.VpcOut{
				// Different cloud: must not match even though the CIDR is the same.
				{ProjectVpcId: "other-id", CloudName: "different-cloud", NetworkCidr: vpcObj.Spec.NetworkCidr, State: vpc.VpcStateTypeActive},
				// Exact spec match: adopted.
				{ProjectVpcId: "adopted-id", CloudName: vpcObj.Spec.CloudName, NetworkCidr: vpcObj.Spec.NetworkCidr, State: vpc.VpcStateTypeActive},
			}, nil).Once()
		// VpcCreate must NOT be called: the pre-existing VPC is adopted instead of recreated.

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := getVPC(t, r, vpcObj)
		require.Equal(t, "adopted-id", got.Status.ID)
		require.Equal(t, vpc.VpcStateTypeActive, got.Status.State)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("APPROVED state does not mark running and soft-requeues", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		// Seed the processed generation so Observe drives the result rather than the create path.
		metav1.SetMetaDataAnnotation(&vpcObj.ObjectMeta, processedGenerationAnnotation, "1")

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcGetOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeApproved}, nil).Once()

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getVPC(t, r, vpcObj)
		require.Equal(t, vpc.VpcStateTypeApproved, got.Status.State)
		require.NotEqual(t, "true", got.Annotations[instanceIsRunningAnnotation])
	})

	t.Run("ACTIVE state marks running and requeues at poll interval", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		// IsReadyToUse needs the processed generation to match; otherwise the poll-interval
		// assertion would pass accidentally.
		metav1.SetMetaDataAnnotation(&vpcObj.ObjectMeta, processedGenerationAnnotation, "1")

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcGetOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeActive}, nil).Once()

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: testPollInterval}, res)

		got := getVPC(t, r, vpcObj)
		require.Equal(t, vpc.VpcStateTypeActive, got.Status.State)
		require.Equal(t, "true", got.Annotations[instanceIsRunningAnnotation])
		require.Equal(t, "1", got.Annotations[processedGenerationAnnotation])
	})

	t.Run("Recreates VPC when VpcGet returns 404 for a non-empty status.ID", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "old-vpc-id"

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "old-vpc-id").
			Return(nil, newAivenError(404, "not found")).Once()
		avn.EXPECT().
			VpcCreate(mock.Anything, vpcObj.Spec.Project, mock.Anything).
			Return(&vpc.VpcCreateOut{ProjectVpcId: "new-vpc-id", State: vpc.VpcStateTypeApproved}, nil).Once()

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getVPC(t, r, vpcObj)
		require.Equal(t, "new-vpc-id", got.Status.ID)
	})

	for _, tc := range []struct {
		name  string
		state vpc.VpcStateType
	}{
		{"DELETING", vpc.VpcStateTypeDeleting},
		{"DELETED", vpc.VpcStateTypeDeleted},
	} {
		t.Run("Deletion: already "+tc.name+" removes finalizer without VpcDelete", func(t *testing.T) {
			vpcObj := newProjectVPC(t)
			vpcObj.Generation = 1
			vpcObj.Status.ID = "vpc-id-1"
			vpcObj.Finalizers = []string{instanceDeletionFinalizer}
			now := metav1.Now()
			vpcObj.DeletionTimestamp = &now

			avn := avngen.NewMockClient(t)
			avn.EXPECT().
				VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
				Return(&vpc.VpcGetOut{ProjectVpcId: "vpc-id-1", State: tc.state}, nil).Once()

			r, res := runScenario(t, vpcObj, avn)
			require.Equal(t, ctrlruntime.Result{}, res)

			got := &v1alpha1.ProjectVPC{}
			err := r.Get(t.Context(), types.NamespacedName{Name: vpcObj.Name, Namespace: vpcObj.Namespace}, got)
			require.True(t, apierrors.IsNotFound(err))
		})
	}

	t.Run("Deletion: 404 on VpcGet removes finalizer", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(nil, newAivenError(404, "not found")).Once()

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ProjectVPC{}
		err := r.Get(t.Context(), types.NamespacedName{Name: vpcObj.Name, Namespace: vpcObj.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Deletion: empty status.ID removes finalizer immediately", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		// No Aiven calls expected: nothing was ever created.
		avn := avngen.NewMockClient(t)

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{}, res)

		got := &v1alpha1.ProjectVPC{}
		err := r.Get(t.Context(), types.NamespacedName{Name: vpcObj.Name, Namespace: vpcObj.Namespace}, got)
		require.True(t, apierrors.IsNotFound(err))
	})

	t.Run("Deletion: dependent service present soft-requeues and keeps finalizer", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcGetOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeActive}, nil).Once()
		avn.EXPECT().
			ServiceList(mock.Anything, vpcObj.Spec.Project).
			Return([]service.ServiceOut{
				{ServiceName: "my-service", ProjectVpcId: "vpc-id-1", State: service.ServiceStateTypeRunning},
			}, nil).Once()
		// VpcDelete must NOT be called while a dependent service exists.

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getVPC(t, r, vpcObj)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Deletion: VpcDelete accepted soft-requeues, then DELETING removes finalizer", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcGetOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeActive}, nil).Once()
		avn.EXPECT().
			ServiceList(mock.Anything, vpcObj.Spec.Project).
			Return([]service.ServiceOut{}, nil).Once()
		avn.EXPECT().
			VpcDelete(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcDeleteOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeDeleting}, nil).Once()

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getVPC(t, r, vpcObj)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
		// The follow-up transition (Aiven reports DELETING/DELETED -> finalizer removed) is
		// covered by the "already DELETING/DELETED" subtests above.
	})

	t.Run("Deletion: VpcDelete dependency error soft-requeues and keeps finalizer", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcGetOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeActive}, nil).Once()
		avn.EXPECT().
			ServiceList(mock.Anything, vpcObj.Spec.Project).
			Return([]service.ServiceOut{}, nil).Once()
		avn.EXPECT().
			VpcDelete(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(nil, newAivenError(409, "VPC cannot be deleted while there are services in it")).Once()

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getVPC(t, r, vpcObj)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Deletion: hard error on VpcGet surfaces error and keeps finalizer", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(nil, newAivenError(403, "forbidden")).Once()

		r, _, err := runScenarioErr(t, vpcObj, avn)
		require.Error(t, err)

		got := getVPC(t, r, vpcObj)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Deletion: ServiceList error surfaces error, keeps finalizer, no VpcDelete", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcGetOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeActive}, nil).Once()
		avn.EXPECT().
			ServiceList(mock.Anything, vpcObj.Spec.Project).
			Return(nil, newAivenError(403, "forbidden")).Once()
		// VpcDelete must NOT be called when ServiceList fails.

		r, _, err := runScenarioErr(t, vpcObj, avn)
		require.Error(t, err)

		got := getVPC(t, r, vpcObj)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Observe: non-404 VpcGet error surfaces and does not create", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		metav1.SetMetaDataAnnotation(&vpcObj.ObjectMeta, processedGenerationAnnotation, "1")

		avn := avngen.NewMockClient(t)
		// A non-retryable error (400 is not 404/5xx/403) surfaces as a hard error from Observe.
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(nil, newAivenError(400, "bad request")).Once()
		// VpcCreate must NOT be called when Observe hits a hard error.

		_, _, err := runScenarioErr(t, vpcObj, avn)
		require.Error(t, err)
	})

	t.Run("Deletion: server error on VpcGet soft-requeues and keeps finalizer", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(nil, newAivenError(500, "internal server error")).Once()

		// handleDeleteError treats 5xx as a transient error: soft requeue, no hard error.
		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getVPC(t, r, vpcObj)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Deletion: non-matching dependent service does not block VpcDelete", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcGetOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeActive}, nil).Once()
		avn.EXPECT().
			ServiceList(mock.Anything, vpcObj.Spec.Project).
			Return([]service.ServiceOut{
				{ServiceName: "other-service", ProjectVpcId: "different-vpc-id", State: service.ServiceStateTypeRunning},
			}, nil).Once()
		// A service in a different VPC must not block deletion: VpcDelete IS called.
		avn.EXPECT().
			VpcDelete(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcDeleteOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeDeleting}, nil).Once()

		r, res := runScenario(t, vpcObj, avn)
		require.Equal(t, ctrlruntime.Result{RequeueAfter: requeueTimeout}, res)

		got := getVPC(t, r, vpcObj)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Deletion: VpcDelete non-dependency error surfaces and keeps finalizer", func(t *testing.T) {
		vpcObj := newProjectVPC(t)
		vpcObj.Generation = 1
		vpcObj.Status.ID = "vpc-id-1"
		vpcObj.Finalizers = []string{instanceDeletionFinalizer}
		now := metav1.Now()
		vpcObj.DeletionTimestamp = &now

		avn := avngen.NewMockClient(t)
		avn.EXPECT().
			VpcGet(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(&vpc.VpcGetOut{ProjectVpcId: "vpc-id-1", State: vpc.VpcStateTypeActive}, nil).Once()
		avn.EXPECT().
			ServiceList(mock.Anything, vpcObj.Spec.Project).
			Return([]service.ServiceOut{}, nil).Once()
		avn.EXPECT().
			VpcDelete(mock.Anything, vpcObj.Spec.Project, "vpc-id-1").
			Return(nil, newAivenError(403, "forbidden")).Once()

		r, _, err := runScenarioErr(t, vpcObj, avn)
		require.Error(t, err)

		got := getVPC(t, r, vpcObj)
		require.Contains(t, got.Finalizers, instanceDeletionFinalizer)
	})

	t.Run("Update is a no-op with no client calls", func(t *testing.T) {
		vpcObj := newProjectVPC(t)

		avn := avngen.NewMockClient(t)
		c := &ProjectVPCController{avnGen: avn}
		res, err := c.Update(t.Context(), vpcObj)
		require.NoError(t, err)
		require.Equal(t, UpdateResult{}, res)
	})
}
