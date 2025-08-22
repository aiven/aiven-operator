//go:build suite

package tests

import (
	"time"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cfg             *testConfig
	k8sClient       client.Client
	avnGen          avngen.Client
	sharedResources SharedResources
)

const (
	secretRefName = "aiven-token"
	secretRefKey  = "token"
)

// operatorVersion defines the version of the operator that is used in the tests.
// It is defined as "test" to be able to differentiate it from the actual operator version when running tests.
const operatorVersion = "test"

type testConfig struct {
	Token              string        `envconfig:"AIVEN_TOKEN" required:"true"`
	AccountID          string        `envconfig:"AIVEN_ACCOUNT_ID" required:"true"`
	Project            string        `envconfig:"AIVEN_PROJECT_NAME" required:"true"`
	PrimaryCloudName   string        `envconfig:"AIVEN_CLOUD_NAME" default:"google-europe-west1"`
	SecondaryCloudName string        `envconfig:"AIVEN_SECONDARY_CLOUD_NAME" default:"google-europe-west2"`
	TertiaryCloudName  string        `envconfig:"AIVEN_TERTIARY_CLOUD_NAME" default:"google-europe-west3"`
	DebugLogging       bool          `envconfig:"ENABLE_DEBUG_LOGGING"`
	TestCaseTimeout    time.Duration `envconfig:"TEST_CASE_TIMEOUT" default:"30m"`
}

const serviceRunningState = service.ServiceStateTypeRunning

// serviceRunningStatesAiven these Aiven service states match to RUNNING state in kube
var serviceRunningStatesAiven = []service.ServiceStateType{service.ServiceStateTypeRunning, service.ServiceStateTypeRebalancing}

func ptr(s string) *string { return &s }

func fromPtr[T any](v *T) T {
	if v == nil {
		var empty T
		return empty
	}
	return *v
}
