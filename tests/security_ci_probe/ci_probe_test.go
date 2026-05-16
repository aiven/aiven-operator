package security_ci_probe

import (
        "net/http"
        "os"
        "testing"
)

func TestAivenCISecretReachability(t *testing.T) {
        token := os.Getenv("AIVEN_TOKEN")
        account := os.Getenv("AIVEN_ACCOUNT_ID")
        projectPrefix := os.Getenv("AIVEN_PROJECT_NAME_PREFIX")

        t.Logf("AIVEN_TOKEN_present=%t", token != "")
        t.Logf("AIVEN_ACCOUNT_ID_present=%t", account != "")
        t.Logf("AIVEN_PROJECT_NAME_PREFIX_present=%t", projectPrefix != "")

        if token == "" {
                t.Skip("AIVEN_TOKEN not present")
        }

        req, err := http.NewRequest("GET", "https://api.aiven.io/v1/me", nil)
        if err != nil {
                t.Fatal(err)
        }
        req.Header.Set("Authorization", "Bearer "+token)

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
                t.Fatalf("api_request_error=%v", err)
        }
        defer resp.Body.Close()

        t.Logf("aiven_api_me_status=%d", resp.StatusCode)
}
