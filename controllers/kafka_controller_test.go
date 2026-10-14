package controllers

import (
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestAddKafkaEndpointDetails(t *testing.T) {
	withoutSchemaRegistry := func() []service.ComponentOut {
		return without(testComponents(), func(c service.ComponentOut) bool { return c.Component == "schema_registry" })
	}
	letsencryptSasl := func() []service.ComponentOut { return withLetsencryptSasl(testComponents()) }
	letsencryptSaslOnly := func() []service.ComponentOut {
		return without(letsencryptSasl(), func(c service.ComponentOut) bool { return isSaslComponent(c) && !isLetsencryptSasl(c) })
	}

	cases := []struct {
		name       string
		components func() []service.ComponentOut
		route      service.RouteType
		want       SecretDetails
	}{
		{
			name:       "dynamic route yields dynamic sasl and schema registry",
			components: testComponents,
			route:      service.RouteTypeDynamic,
			want: SecretDetails{
				"P_SASL_HOST":            "kafka-dynamic-sasl",
				"P_SASL_PORT":            "6",
				"P_SCHEMA_REGISTRY_HOST": "sr-dynamic",
				"P_SCHEMA_REGISTRY_PORT": "9",
			},
		},
		{
			name:       "letsencrypt sasl on dynamic keeps the project CA sasl entry",
			components: letsencryptSasl,
			route:      service.RouteTypeDynamic,
			want: SecretDetails{
				"P_SASL_HOST":            "kafka-dynamic-sasl",
				"P_SASL_PORT":            "6",
				"P_SCHEMA_REGISTRY_HOST": "sr-dynamic",
				"P_SCHEMA_REGISTRY_PORT": "9",
			},
		},
		{
			name:       "letsencrypt sasl on privatelink keeps the project CA sasl entry",
			components: letsencryptSasl,
			route:      service.RouteTypePrivatelink,
			want: SecretDetails{
				"P_SASL_HOST":            "kafka-privatelink-sasl",
				"P_SASL_PORT":            "8",
				"P_SCHEMA_REGISTRY_HOST": "sr-privatelink",
				"P_SCHEMA_REGISTRY_PORT": "10",
			},
		},
		{
			name:       "letsencrypt sasl alone is the fallback",
			components: letsencryptSaslOnly,
			route:      service.RouteTypeDynamic,
			want: SecretDetails{
				"P_SASL_HOST":            "kafka-dynamic-sasl-letsencrypt",
				"P_SASL_PORT":            "11",
				"P_SCHEMA_REGISTRY_HOST": "sr-dynamic",
				"P_SCHEMA_REGISTRY_PORT": "9",
			},
		},
		{
			name:       "privatelink route yields privatelink sasl and schema registry",
			components: testComponents,
			route:      service.RouteTypePrivatelink,
			want: SecretDetails{
				"P_SASL_HOST":            "kafka-privatelink-sasl",
				"P_SASL_PORT":            "8",
				"P_SCHEMA_REGISTRY_HOST": "sr-privatelink",
				"P_SCHEMA_REGISTRY_PORT": "10",
			},
		},
		{
			name:       "no schema registry on dynamic blanks the schema registry keys",
			components: withoutSchemaRegistry,
			route:      service.RouteTypeDynamic,
			want: SecretDetails{
				"P_SASL_HOST":            "kafka-dynamic-sasl",
				"P_SASL_PORT":            "6",
				"P_SCHEMA_REGISTRY_HOST": "",
				"P_SCHEMA_REGISTRY_PORT": "",
			},
		},
		{
			name:       "no schema registry on privatelink blanks the schema registry keys",
			components: withoutSchemaRegistry,
			route:      service.RouteTypePrivatelink,
			want: SecretDetails{
				"P_SASL_HOST":            "kafka-privatelink-sasl",
				"P_SASL_PORT":            "8",
				"P_SCHEMA_REGISTRY_HOST": "",
				"P_SCHEMA_REGISTRY_PORT": "",
			},
		},
		{
			name:       "route without components blanks the keys",
			components: testComponents,
			route:      service.RouteTypePrivate,
			want: SecretDetails{
				"P_SASL_HOST":            "",
				"P_SASL_PORT":            "",
				"P_SCHEMA_REGISTRY_HOST": "",
				"P_SCHEMA_REGISTRY_PORT": "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listed := SecretDetails{}
			addKafkaEndpointDetails(listed, tc.components(), tc.route, "P_")
			assert.Equal(t, tc.want, listed)

			rev := SecretDetails{}
			addKafkaEndpointDetails(rev, reversed(tc.components()), tc.route, "P_")
			assert.Equal(t, listed, rev)
		})
	}

	t.Run("keeps existing keys", func(t *testing.T) {
		details := SecretDetails{"P_HOST": "keep"}
		addKafkaEndpointDetails(details, testComponents(), service.RouteTypeDynamic, "P_")
		assert.Equal(t, "keep", details["P_HOST"])
		assert.Len(t, details, 5)
	})

	// Unset route: the selection of releases before the field, kept so upgrades do not rewrite secrets.
	t.Run("legacy route takes the last entry in API order whatever its route", func(t *testing.T) {
		listed := SecretDetails{}
		addKafkaEndpointDetails(listed, testComponents(), routeLegacy, "P_")
		assert.Equal(t, SecretDetails{
			"P_SASL_HOST":            "kafka-privatelink-sasl",
			"P_SASL_PORT":            "8",
			"P_SCHEMA_REGISTRY_HOST": "sr-privatelink",
			"P_SCHEMA_REGISTRY_PORT": "10",
		}, listed)

		rev := SecretDetails{}
		addKafkaEndpointDetails(rev, reversed(testComponents()), routeLegacy, "P_")
		assert.Equal(t, SecretDetails{
			"P_SASL_HOST":            "kafka-dynamic-sasl",
			"P_SASL_PORT":            "6",
			"P_SCHEMA_REGISTRY_HOST": "sr-dynamic",
			"P_SCHEMA_REGISTRY_PORT": "9",
		}, rev)
	})

	t.Run("legacy route sets no keys without sasl or schema registry", func(t *testing.T) {
		details := SecretDetails{}
		addKafkaEndpointDetails(details, without(testComponents(), func(c service.ComponentOut) bool {
			return isSaslComponent(c) || c.Component == "schema_registry"
		}), routeLegacy, "P_")
		assert.Empty(t, details)
	})
}

// The service secret keeps the legacy selection: the last SASL and schema registry entry in API order.
func TestKafkaAdapterNewSecret(t *testing.T) {
	cert, key, rest, sr := "cert", "key", "https://rest", "https://sr"
	svc := func(components []service.ComponentOut) *service.ServiceGetOut {
		return &service.ServiceGetOut{
			Components:       components,
			ServiceUriParams: map[string]string{"host": "kafka-host", "port": "9092"},
			Users:            []service.UserOut{{Username: "avnadmin", Password: "pw"}},
			ConnectionInfo:   &service.ConnectionInfoOut{KafkaAccessCert: &cert, KafkaAccessKey: &key, KafkaRestUri: &rest, SchemaRegistryUri: &sr},
		}
	}
	a := &kafkaAdapter{Kafka: &v1alpha1.Kafka{
		TypeMeta:   metav1.TypeMeta{Kind: "Kafka"},
		ObjectMeta: metav1.ObjectMeta{Name: "kafka", Namespace: "default"},
	}}

	want := func(saslHost, saslPort, srHost, srPort string) map[string]string {
		return map[string]string{
			"KAFKA_HOST":                 "kafka-host",
			"KAFKA_PORT":                 "9092",
			"KAFKA_PASSWORD":             "pw",
			"KAFKA_USERNAME":             "avnadmin",
			"KAFKA_ACCESS_CERT":          "cert",
			"KAFKA_ACCESS_KEY":           "key",
			"KAFKA_REST_URI":             "https://rest",
			"KAFKA_SCHEMA_REGISTRY_URI":  "https://sr",
			"KAFKA_SASL_HOST":            saslHost,
			"KAFKA_SASL_PORT":            saslPort,
			"KAFKA_SCHEMA_REGISTRY_HOST": srHost,
			"KAFKA_SCHEMA_REGISTRY_PORT": srPort,
			"HOST":                       "kafka-host",
			"PORT":                       "9092",
			"PASSWORD":                   "pw",
			"USERNAME":                   "avnadmin",
			"ACCESS_CERT":                "cert",
			"ACCESS_KEY":                 "key",
		}
	}

	assert.Equal(t, want("kafka-privatelink-sasl", "8", "sr-privatelink", "10"), a.newSecret(svc(testComponents())).StringData)
	assert.Equal(t, want("kafka-dynamic-sasl", "6", "sr-dynamic", "9"), a.newSecret(svc(reversed(testComponents()))).StringData)
}

func TestRefreshKafkaEndpointDetails(t *testing.T) {
	const prefix = "P_"
	optional := []string{"P_SASL_HOST", "P_SASL_PORT", "P_SCHEMA_REGISTRY_HOST", "P_SCHEMA_REGISTRY_PORT"}
	stale := func() map[string][]byte {
		data := map[string][]byte{"P_HOST": []byte("kept"), "unrelated": []byte("kept")}
		for _, key := range optional {
			data[key] = []byte("stale")
		}
		return data
	}
	endpoints := func(saslHost, saslPort, srHost, srPort string) map[string][]byte {
		return map[string][]byte{
			"P_HOST": []byte("kept"), "unrelated": []byte("kept"),
			"P_SASL_HOST": []byte(saslHost), "P_SASL_PORT": []byte(saslPort),
			"P_SCHEMA_REGISTRY_HOST": []byte(srHost), "P_SCHEMA_REGISTRY_PORT": []byte(srPort),
		}
	}
	withoutSchemaRegistry := without(testComponents(), func(c service.ComponentOut) bool { return c.Component == "schema_registry" })

	cases := []struct {
		name       string
		components []service.ComponentOut
		route      service.RouteType
		want       map[string][]byte
	}{
		{"legacy route keeps the last entry in API order", testComponents(), routeLegacy, endpoints("kafka-privatelink-sasl", "8", "sr-privatelink", "10")},
		{"legacy route follows reversed API order", reversed(testComponents()), routeLegacy, endpoints("kafka-dynamic-sasl", "6", "sr-dynamic", "9")},
		{"legacy route removes keys of a missing component", withoutSchemaRegistry, routeLegacy, map[string][]byte{
			"P_HOST": []byte("kept"), "unrelated": []byte("kept"), "P_SASL_HOST": []byte("kafka-privatelink-sasl"), "P_SASL_PORT": []byte("8"),
		}},
		{"dynamic route picks the dynamic primaries", testComponents(), service.RouteTypeDynamic, endpoints("kafka-dynamic-sasl", "6", "sr-dynamic", "9")},
		{"dynamic route ignores API order", reversed(testComponents()), service.RouteTypeDynamic, endpoints("kafka-dynamic-sasl", "6", "sr-dynamic", "9")},
		{"privatelink route picks the privatelink primaries", reversed(testComponents()), service.RouteTypePrivatelink, endpoints("kafka-privatelink-sasl", "8", "sr-privatelink", "10")},
		{"route without the components removes the keys", testComponents(), service.RouteTypePublic, map[string][]byte{
			"P_HOST": []byte("kept"), "unrelated": []byte("kept"),
		}},
		{"route without schema registry removes only its keys", withoutSchemaRegistry, service.RouteTypePrivatelink, map[string][]byte{
			"P_HOST": []byte("kept"), "unrelated": []byte("kept"), "P_SASL_HOST": []byte("kafka-privatelink-sasl"), "P_SASL_PORT": []byte("8"),
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := stale()
			refreshKafkaEndpointDetails(data, tc.components, tc.route, prefix)
			assert.Equal(t, tc.want, data)
		})
	}
}
