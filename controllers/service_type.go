// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

type serviceType string

const (
	// Aiven API service_type values of the service kinds.
	serviceTypeKafka        serviceType = "kafka"
	serviceTypeKafkaConnect serviceType = "kafka_connect"
	serviceTypeMySQL        serviceType = "mysql"
	serviceTypePostgreSQL   serviceType = "pg"
	serviceTypeClickhouse   serviceType = "clickhouse"
	serviceTypeOpenSearch   serviceType = "opensearch"
	serviceTypeGrafana      serviceType = "grafana"
	serviceTypeFlink        serviceType = "flink"
	serviceTypeValkey       serviceType = "valkey"
)

// serviceVersionKeys maps a service type to the key holding its running version in the
// service metadata. kafka_connect is absent: its metadata has no comparable version string.
var serviceVersionKeys = map[serviceType]string{
	serviceTypeKafka:      "kafka_version",
	serviceTypeMySQL:      "mysql_version",
	serviceTypePostgreSQL: "pg_version",
	serviceTypeClickhouse: "clickhouse_version",
	serviceTypeOpenSearch: "opensearch_version",
	serviceTypeGrafana:    "grafana_version",
	serviceTypeFlink:      "flink_version",
	serviceTypeValkey:     "valkey_version",
}

// serviceVersion extracts the running service version from the service metadata.
// Returns an empty string when the API doesn't report one.
func serviceVersion(t serviceType, metadata map[string]any) string {
	v, _ := metadata[serviceVersionKeys[t]].(string)
	return v
}
