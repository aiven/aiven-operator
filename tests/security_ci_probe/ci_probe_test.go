//go:build security_ci_probe

package security_ci_probe

import (
	"os"
	"testing"
)

func TestAivenCISecretReachability(t *testing.T) {
	token := os.Getenv("AIVEN_TOKEN")
	account := os.Getenv("AIVEN_ACCOUNT_ID")
	project := os.Getenv("AIVEN_PROJECT_NAME_PREFIX")

	t.Logf("SECURITY_CI_PROBE_EXECUTED=true")
	t.Logf("AIVEN_TOKEN_present=%t", token != "")
	t.Logf("AIVEN_ACCOUNT_ID_present=%t", account != "")
	t.Logf("AIVEN_PROJECT_NAME_PREFIX_present=%t", project != "")

	if token == "" {
		t.Skip("AIVEN_TOKEN not present")
	}
}
