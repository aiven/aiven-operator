package v1alpha1

// AuthSecretReference references a Secret containing an Aiven authentication token
type AuthSecretReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}
