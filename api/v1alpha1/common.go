package v1alpha1

// AuthSecretReference references a Secret containing an Aiven authentication token
type AuthSecretReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// SecretConnectionInfo contains information secret name
type SecretConnectionInfo struct {
	// Name of the Secret resource to be created
	Name string `json:"name"`
}
