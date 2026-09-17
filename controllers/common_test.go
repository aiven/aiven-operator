package controllers

import (
	"slices"
	"testing"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kafkaconnectuserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/integration/kafka_connect"
)

// TestCreateEmptyUserConfiguration shouldn't panic
func TestCreateEmptyUserConfiguration(t *testing.T) {
	var uc *kafkaconnectuserconfig.KafkaConnectUserConfig
	m, err := CreateUserConfiguration(uc)
	assert.Empty(t, m)
	assert.NoError(t, err)
}

// testComponents is the shared components fixture.
func testComponents() []service.ComponentOut {
	return []service.ComponentOut{
		{Component: "pg", Host: "pg-dynamic-primary", Port: 1, Route: service.RouteTypeDynamic, Usage: service.UsageTypePrimary},
		{Component: "pg", Host: "pg-dynamic-replica", Port: 2, Route: service.RouteTypeDynamic, Usage: service.UsageTypeReplica},
		{Component: "pg", Host: "pg-privatelink-primary", Port: 3, Route: service.RouteTypePrivatelink, Usage: service.UsageTypePrimary},
		{Component: "pg", Host: "pg-public-primary", Port: 4, Route: service.RouteTypePublic, Usage: service.UsageTypePrimary},
		{Component: "kafka", Host: "kafka-dynamic-cert", Port: 5, Route: service.RouteTypeDynamic, Usage: service.UsageTypePrimary, KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeCertificate},
		{Component: "kafka", Host: "kafka-dynamic-sasl", Port: 6, Route: service.RouteTypeDynamic, Usage: service.UsageTypePrimary, KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeSasl},
		{Component: "kafka", Host: "kafka-privatelink-cert", Port: 7, Route: service.RouteTypePrivatelink, Usage: service.UsageTypePrimary, KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeCertificate},
		{Component: "kafka", Host: "kafka-privatelink-sasl", Port: 8, Route: service.RouteTypePrivatelink, Usage: service.UsageTypePrimary, KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeSasl, KafkaSslCa: service.KafkaSslCaTypeProjectCa},
		{Component: "schema_registry", Host: "sr-dynamic", Port: 9, Route: service.RouteTypeDynamic, Usage: service.UsageTypePrimary},
		{Component: "schema_registry", Host: "sr-privatelink", Port: 10, Route: service.RouteTypePrivatelink, Usage: service.UsageTypePrimary},
	}
}

// withLetsencryptSasl appends the second SASL entry letsencrypt_sasl exposes on each route.
func withLetsencryptSasl(components []service.ComponentOut) []service.ComponentOut {
	return append(slices.Clone(components),
		service.ComponentOut{Component: "kafka", Host: "kafka-dynamic-sasl-letsencrypt", Port: 11, Route: service.RouteTypeDynamic, Usage: service.UsageTypePrimary, KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeSasl, KafkaSslCa: service.KafkaSslCaTypeLetsencrypt},
		service.ComponentOut{Component: "kafka", Host: "kafka-privatelink-sasl-letsencrypt", Port: 12, Route: service.RouteTypePrivatelink, Usage: service.UsageTypePrimary, KafkaAuthenticationMethod: service.KafkaAuthenticationMethodTypeSasl, KafkaSslCa: service.KafkaSslCaTypeLetsencrypt},
	)
}

// recordedKafkaPrivatelinkComponents is the components list the Aiven API returned for a Kafka
// in an AWS project VPC with two PrivateLink connections.
func recordedKafkaPrivatelinkComponents() []service.ComponentOut {
	const (
		dyn   = "rt-pl-kafka-aiven-ci-kubernetes-operator.c.aivencloud.com"
		pl1   = "privatelink-1-rt-pl-kafka-aiven-ci-kubernetes-operator.c.aivencloud.com"
		pl2   = "privatelink-2-rt-pl-kafka-aiven-ci-kubernetes-operator.c.aivencloud.com"
		conn1 = "plc5e3f907f46c"
		conn2 = "plc5e3f9573e9a"
	)
	kafka := func(route service.RouteType, auth service.KafkaAuthenticationMethodType, conn *string, host string, port int) service.ComponentOut {
		return service.ComponentOut{
			Component: "kafka", Route: route, Usage: service.UsageTypePrimary, Host: host, Port: port,
			KafkaAuthenticationMethod: auth, KafkaSslCa: service.KafkaSslCaTypeProjectCa, PrivatelinkConnectionId: conn,
		}
	}
	sr := func(route service.RouteType, conn *string, host string) service.ComponentOut {
		return service.ComponentOut{Component: "schema_registry", Route: route, Usage: service.UsageTypePrimary, Host: host, Port: 14614, PrivatelinkConnectionId: conn}
	}
	c1, c2 := conn1, conn2
	return []service.ComponentOut{
		kafka(service.RouteTypeDynamic, service.KafkaAuthenticationMethodTypeCertificate, nil, dyn, 14611),
		kafka(service.RouteTypeDynamic, service.KafkaAuthenticationMethodTypeSasl, nil, dyn, 14622),
		kafka(service.RouteTypePrivatelink, service.KafkaAuthenticationMethodTypeCertificate, &c1, pl1, 23220),
		kafka(service.RouteTypePrivatelink, service.KafkaAuthenticationMethodTypeCertificate, &c2, pl2, 23228),
		kafka(service.RouteTypePrivatelink, service.KafkaAuthenticationMethodTypeSasl, &c1, pl1, 23224),
		kafka(service.RouteTypePrivatelink, service.KafkaAuthenticationMethodTypeSasl, &c2, pl2, 23232),
		sr(service.RouteTypeDynamic, nil, dyn),
		sr(service.RouteTypePrivatelink, &c1, pl1),
		sr(service.RouteTypePrivatelink, &c2, pl2),
	}
}

// isLetsencryptSasl matches the entries withLetsencryptSasl adds.
func isLetsencryptSasl(c service.ComponentOut) bool {
	return c.KafkaSslCa == service.KafkaSslCaTypeLetsencrypt
}

// reversed returns a copy of components in reverse order, to prove results do not depend on
// API ordering.
func reversed(components []service.ComponentOut) []service.ComponentOut {
	out := slices.Clone(components)
	slices.Reverse(out)
	return out
}

// without returns a copy of components without the entries drop matches.
func without(components []service.ComponentOut, drop func(service.ComponentOut) bool) []service.ComponentOut {
	return slices.DeleteFunc(slices.Clone(components), drop)
}

func TestFindComponent(t *testing.T) {
	cases := []struct {
		name     string
		comp     string
		route    service.RouteType
		match    func(service.ComponentOut) bool
		wantHost string // empty means not found
	}{
		{name: "dynamic primary picked for pg", comp: "pg", route: service.RouteTypeDynamic, wantHost: "pg-dynamic-primary"},
		{name: "privatelink primary picked", comp: "pg", route: service.RouteTypePrivatelink, wantHost: "pg-privatelink-primary"},
		{name: "public picked", comp: "pg", route: service.RouteTypePublic, wantHost: "pg-public-primary"},
		{name: "missing route returns false", comp: "pg", route: service.RouteTypePrivate},
		{name: "missing component name returns false", comp: "mysql", route: service.RouteTypeDynamic},
		{name: "sasl predicate picks sasl kafka on dynamic", comp: "kafka", route: service.RouteTypeDynamic, match: isSaslComponent, wantHost: "kafka-dynamic-sasl"},
		{name: "sasl predicate picks sasl kafka on privatelink", comp: "kafka", route: service.RouteTypePrivatelink, match: isSaslComponent, wantHost: "kafka-privatelink-sasl"},
		{name: "certificate predicate picks certificate kafka", comp: "kafka", route: service.RouteTypePrivatelink, match: isCertificateComponent, wantHost: "kafka-privatelink-cert"},
		{name: "sasl predicate on route without sasl returns false", comp: "kafka", route: service.RouteTypePublic, match: isSaslComponent},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listed, okListed := findComponent(testComponents(), tc.comp, tc.route, tc.match)
			rev, okRev := findComponent(reversed(testComponents()), tc.comp, tc.route, tc.match)
			assert.Equal(t, okListed, okRev)
			assert.Equal(t, listed, rev)

			if tc.wantHost == "" {
				assert.False(t, okListed)
				assert.Nil(t, listed)
				return
			}
			require.True(t, okListed)
			assert.Equal(t, tc.wantHost, listed.Host)
		})
	}
}
