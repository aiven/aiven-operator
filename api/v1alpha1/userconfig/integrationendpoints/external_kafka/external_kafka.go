// Code generated by user config generator. DO NOT EDIT.
// +kubebuilder:object:generate=true

package externalkafkauserconfig

type ExternalKafkaUserConfig struct {
	// +kubebuilder:validation:MinLength=3
	// +kubebuilder:validation:MaxLength=256
	// Bootstrap servers
	BootstrapServers string `groups:"create,update" json:"bootstrap_servers"`

	// +kubebuilder:validation:Enum="PLAIN";"SCRAM-SHA-256";"SCRAM-SHA-512"
	// SASL mechanism used for connections to the Kafka server.
	SaslMechanism *string `groups:"create,update" json:"sasl_mechanism,omitempty"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	// Password for SASL PLAIN mechanism in the Kafka server.
	SaslPlainPassword *string `groups:"create,update" json:"sasl_plain_password,omitempty"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=256
	// Username for SASL PLAIN mechanism in the Kafka server.
	SaslPlainUsername *string `groups:"create,update" json:"sasl_plain_username,omitempty"`

	// +kubebuilder:validation:Enum="PLAINTEXT";"SASL_PLAINTEXT";"SASL_SSL";"SSL"
	// Security protocol
	SecurityProtocol string `groups:"create,update" json:"security_protocol"`

	// +kubebuilder:validation:MaxLength=16384
	// PEM-encoded CA certificate
	SslCaCert *string `groups:"create,update" json:"ssl_ca_cert,omitempty"`

	// +kubebuilder:validation:MaxLength=16384
	// PEM-encoded client certificate
	SslClientCert *string `groups:"create,update" json:"ssl_client_cert,omitempty"`

	// +kubebuilder:validation:MaxLength=16384
	// PEM-encoded client key
	SslClientKey *string `groups:"create,update" json:"ssl_client_key,omitempty"`

	// +kubebuilder:validation:Enum="https"
	// The endpoint identification algorithm to validate server hostname using server certificate.
	SslEndpointIdentificationAlgorithm *string `groups:"create,update" json:"ssl_endpoint_identification_algorithm,omitempty"`
}
