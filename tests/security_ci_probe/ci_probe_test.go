package security_ci_probe

import (
	"os"
	"testing"
)

func TestAivenCITrustBoundaryProbe(t *testing.T) {
	t.Logf("SECURITY_CI_TRUST_BOUNDARY_PROBE_EXECUTED=true")
	t.Logf("GITHUB_ACTOR=%s", os.Getenv("GITHUB_ACTOR"))
	t.Logf("GITHUB_EVENT_NAME=%s", os.Getenv("GITHUB_EVENT_NAME"))
	t.Logf("GITHUB_REF=%s", os.Getenv("GITHUB_REF"))
	t.Logf("GITHUB_HEAD_REF=%s", os.Getenv("GITHUB_HEAD_REF"))
	t.Logf("GITHUB_BASE_REF=%s", os.Getenv("GITHUB_BASE_REF"))

	t.Logf("AIVEN_TOKEN_present=%t", os.Getenv("AIVEN_TOKEN") != "")
	t.Logf("AIVEN_ACCOUNT_ID_present=%t", os.Getenv("AIVEN_ACCOUNT_ID") != "")
	t.Logf("AIVEN_PROJECT_NAME_PREFIX_present=%t", os.Getenv("AIVEN_PROJECT_NAME_PREFIX") != "")
}
