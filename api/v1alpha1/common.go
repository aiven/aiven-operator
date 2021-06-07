package v1alpha1

// AuthSecretReference reference to Aiven token
type AuthSecretReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}
