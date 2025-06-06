package alloydbomniutils

import (
	"errors"

	"github.com/xeipuuv/gojsonschema"
)

func ValidateServiceAccountCredentials(s string) error {
	if s == "" {
		return nil
	}

	r, err := gojsonschema.Validate(
		gojsonschema.NewStringLoader(serviceAccountCredentialsSchema),
		gojsonschema.NewStringLoader(s),
	)
	if err != nil {
		return err
	}

	if !r.Valid() {
		var errMsg string
		for _, e := range r.Errors() {
			errMsg += e.String() + "\n"
		}
		return errors.New(errMsg)
	}

	return nil
}

// trunk-ignore-all(gitleaks/private-key)
const serviceAccountCredentialsSchema = `{
  "title": "Google service account credentials map",
  "type": "object",
  "properties": {
    "type": {
      "type": "string",
      "title": "Credentials type",
      "description": "Always service_account for credentials created in Gcloud console or CLI",
      "example": "service_account"
    },
    "project_id": {
      "type": "string",
      "title": "Gcloud project id",
      "example": "some-my-project"
    },
    "private_key_id": {
      "type": "string",
      "title": "Hexadecimal ID number of your private key",
      "example": "5fdeb02a11ddf081930ac3ac60bf376a0aef8fad"
    },
    "private_key": {
      "type": "string",
      "title": "PEM-encoded private key",
      "example": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
    },
    "client_email": {
      "type": "string",
      "title": "Email of the service account",
      "example": "my-service-account@some-my-project.iam.gserviceaccount.com"
    },
    "client_id": {
      "type": "string",
      "title": "Numeric client id for this service account",
      "example": "103654484443722885992"
    },
    "auth_uri": {
      "type": "string",
      "title": "The authentication endpoint of Google",
      "example": "https://accounts.google.com/o/oauth2/auth"
    },
    "token_uri": {
      "type": "string",
      "title": "The token lease endpoint of Google",
      "example": "https://accounts.google.com/o/oauth2/token"
    },
    "auth_provider_x509_cert_url": {
      "type": "string",
      "title": "The certificate service of Google",
      "example": "https://www.googleapis.com/oauth2/v1/certs"
    },
    "client_x509_cert_url": {
      "type": "string",
      "title": "Certificate URL for your service account",
      "example": "https://www.googleapis.com/robot/v1/metadata/x509/my-service-account%40some-my-project.iam.gserviceaccount.com"
    },
    "universe_domain": {
      "type": "string",
      "title": "The universe domain",
      "description": "The universe domain. The default universe domain is googleapis.com."
    }
  },
  "required": [
    "private_key_id",
    "private_key",
    "client_email",
    "client_id"
  ],
  "additionalProperties": false
}`
