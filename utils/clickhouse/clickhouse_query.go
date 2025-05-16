package chutils

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/clickhouse"
)

const (
	defaultDatabase = "system"
)

func ExecuteClickHouseQuery(ctx context.Context, avnGen avngen.Client, project, serviceName, statement string) (*clickhouse.ServiceClickHouseQueryOut, error) {
	res, err := avnGen.ServiceClickHouseQuery(ctx, project, serviceName, &clickhouse.ServiceClickHouseQueryIn{
		// We are running GRANT and REVOKE which don't need to be ran against a
		// specific database. Here "system" is used as its guaranteed to exist.
		Database: defaultDatabase,
		Query:    statement,
	})
	if err != nil {
		return nil, fmt.Errorf("ClickHouse query error: %w", err)
	}
	return res, nil
}
