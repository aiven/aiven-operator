package main

import (
	"fmt"
	"os"
	"slices"
	"sort"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/goccy/go-yaml"
)

// permissionGroups is a list of roles/permission groups where:
// - first level elements have AND relationship
// - second level elements have OR relationship (interchangeable)
// For example: [[x], [y, z]] means: x AND (y OR z)
type permissionGroups [][]string

func readPermissionsFile(permissionsFile string) (map[string]permissionGroups, error) {
	permissions, err := os.ReadFile(permissionsFile)
	if err != nil {
		return nil, err
	}

	operationIDs := make(map[string][]string)
	err = yaml.Unmarshal(permissions, &operationIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal %q file: %w", permissionsFile, err)
	}

	permissionsMap, err := avngen.Permissions()
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions from the generated client: %w", err)
	}

	res := make(map[string]permissionGroups)
	for kind, ids := range operationIDs {
		perm := make(permissionGroups, 0)
		for _, id := range ids {
			v := permissionsMap[id]
			if len(v) != 0 {
				slices.Sort(v)
				perm = append(perm, v)
			}

			switch id {
			case "ServiceCreate", "ServiceClickHouseUserCreate", "ServicePGBouncerCreate":
				// These endpoints add `include_secrets=true` query
				// It is an implicit permission, so we add it manually
				perm = append(perm, []string{"service:secrets:read"})
			}
		}

		// Roles are interchangeable.
		// 1. Tracks all seen permissions
		// 2. If any permission in permissionGroups has been "seen", removes the whole group.
		// For example, the second is removed:
		// 1. project:services:write
		// 2. project:services:write or role:services:recover
		// The second group is removed because `project:services:write` is already present.
		// Smaller groups go first because they are more specific and less confusing than groups with an "or".
		sort.Slice(perm, func(i, j int) bool {
			return len(perm[i]) < len(perm[j])
		})

		i := 0
		seen := make(map[string]bool)
	outer:
		for _, group := range perm {
			for _, v := range group {
				if seen[v] {
					// An element of the group has been seen, removes the whole group.
					continue outer
				}
			}

			for _, v := range group {
				seen[v] = true
			}
			perm[i] = group
			i++
		}

		// Most groups contain a single element.
		// They are sorted by the first element to make the whole list alphabetically ordered.
		uniq := perm[:i]
		sort.Slice(uniq, func(i, j int) bool {
			return uniq[i][0] < uniq[j][0]
		})

		res[kind] = uniq
	}
	return res, nil
}

func setPermissions(permissionsMap map[string]permissionGroups, schema *schemaType) error {
	perm, ok := permissionsMap[schema.Kind]
	if !ok {
		return fmt.Errorf("no role permissions found for %q", schema.Kind)
	}

	schema.Permissions = perm
	return nil
}
