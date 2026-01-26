package main

import (
	"fmt"
	"os"
	"slices"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/goccy/go-yaml"
)

// kindOperations is a list of API operations for a kind (e.g. PostgreSQL, Kafka, etc.)
type kindOperations []operationPermissions

type operationPermissions struct {
	OperationID string
	Permissions []string // Any one of these permissions is sufficient
}

func readPermissionsFile(permissionsFile string) (map[string]kindOperations, error) {
	permissions, err := os.ReadFile(permissionsFile)
	if err != nil {
		return nil, err
	}

	operationIDs := make(map[string][]string)
	err = yaml.Unmarshal(permissions, &operationIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal %q file: %w", permissionsFile, err)
	}

	permissionsMap := avngen.Permissions()
	res := make(map[string]kindOperations)
	for kind, ids := range operationIDs {
		slices.Sort(ids)

		for _, id := range ids {
			v := permissionsMap[id]
			if len(v) == 0 {
				continue
			}

			// Copy the permissions to avoid modifying the original map
			perm := append([]string{}, v...)
			slices.Sort(perm)

			if id == "ServiceGet" {
			outer:
				for _, i := range ids {
					switch i {
					case "ServiceCreate", "ServiceClickHouseUserCreate", "ServicePGBouncerCreate":
						// These endpoints add `include_secrets=true` query.
						// It requires an implicit permission, and already inherits `project:services:read`.
						perm = []string{"service:secrets:read"}
						break outer
					}
				}
			}

			res[kind] = append(res[kind], operationPermissions{
				OperationID: id,
				Permissions: perm,
			})
		}
	}
	return res, nil
}

func setPermissions(permissionsMap map[string]kindOperations, schema *schemaType) error {
	perm, ok := permissionsMap[schema.Kind]
	if !ok {
		return fmt.Errorf("no role permissions found for %q", schema.Kind)
	}

	schema.Permissions = perm
	return nil
}
