package tests

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestYamlFile(t *testing.T, yamlContent string) (string, func()) {
	tempDir, err := os.MkdirTemp("", "yaml-tests")
	assert.NoError(t, err)

	examplesDirPath = tempDir

	// Use last part of the test name as the filename:
	// Example:
	//   "TestLoadExampleYaml/basic_operations/should_update_top_level_value"
	//   => "should_update_top_level_value.yaml"
	parts := strings.Split(t.Name(), "/")
	filename := parts[len(parts)-1] + ".yaml"
	filePath := path.Join(tempDir, filename)
	err = os.WriteFile(filePath, []byte(yamlContent), 0o644)
	assert.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return filename, cleanup
}

type testCase struct {
	name         string
	yamlContent  string
	replacements map[string]string
	expected     []string
	errorMsg     *string
}

func TestLoadExampleYaml(t *testing.T) {
	errNodeNotFound := "node not found for path"
	errFilterNode := "failed to filter node"
	errDocIndex := "document index out of range"

	tests := map[string][]testCase{
		"basic_operations": {
			{
				name: "should_update_top_level_value",
				yamlContent: `
data:
  app_name: "my-app"`,
				replacements: map[string]string{
					"data.app_name": "my-new-app",
				},
				expected: []string{"my-new-app"},
			},
			{
				name: "should_update_multiline_value",
				yamlContent: `
data:
  credentials: |
    key: value`,
				replacements: map[string]string{
					"data.credentials": "\n  key: new_value",
				},
				expected: []string{"new_value"},
			},
			{
				name:        "should_handle_empty_yaml",
				yamlContent: "",
				replacements: map[string]string{
					"key": "value",
				},
				errorMsg: &errNodeNotFound,
			},
			{
				name: "should_handle_missing_key",
				yamlContent: `
data:
  app_name: "my-app"`,
				replacements: map[string]string{
					"data.non_existent_key": "value",
				},
				errorMsg: &errNodeNotFound,
			},
		},
		"nested_operations": {
			{
				name: "should_update_deeply_nested_value",
				yamlContent: `
data:
  level1:
    level2:
      level3:
        level4:
          key: value`,
				replacements: map[string]string{
					"data.level1.level2.level3.level4.key": "new-nested-value",
				},
				expected: []string{"new-nested-value"},
			},
		},
		"removal_operations": {
			{
				name: "should_remove_key",
				yamlContent: `
data:
  key_to_remove: value1
  key_to_keep: value2`,
				replacements: map[string]string{
					"data.key_to_remove": "REMOVE",
				},
				expected: []string{"key_to_keep"},
			},
			{
				name: "should_handle_mixed_operations",
				yamlContent: `
data:
  remove_me: value1
  update_me: value2
  keep_me: value3`,
				replacements: map[string]string{
					"data.remove_me": "REMOVE",
					"data.update_me": "new-value",
				},
				expected: []string{"new-value", "keep_me", "value3"},
			},
		},
		"array_operations": {
			{
				name: "should_update_value_in_array_index",
				yamlContent: `
items:
  - name: item1
    value: value1
  - name: item2
    value: value2`,
				replacements: map[string]string{
					"items[0].value": "new-array-value",
				},
				expected: []string{"new-array-value"},
			},
			{
				name: "should_handle_array_out_of_bounds",
				yamlContent: `
items:
  - value: value1`,
				replacements: map[string]string{
					"items[1].value": "new-value",
				},
				errorMsg: &errFilterNode,
			},
		},
		"multiple_documents": {
			{
				name: "should_update_values_in_both_documents",
				yamlContent: `
key1: value1
---
key1: value2`,
				replacements: map[string]string{
					"doc[0].key1": "new-value1",
					"doc[1].key1": "new-value2",
				},
				expected: []string{"new-value1", "new-value2"},
			},
			{
				name: "should_handle_invalid_document_index",
				yamlContent: `
key1: value1`,
				replacements: map[string]string{
					"doc[1].key1": "new-value",
				},
				errorMsg: &errDocIndex,
			},
			{
				name: "should_remove_entire_document",
				yamlContent: `
key1: value1
---
key2: value2`,
				replacements: map[string]string{
					"doc[1]": "REMOVE",
				},
				expected: []string{"key1: value1"},
			},
		},
		"special_cases": {
			{
				name: "should_handle_unicode",
				yamlContent: `
data:
  key: "🦄"`,
				replacements: map[string]string{
					"data.key": "🦀",
				},
				expected: []string{"🦀"},
			},
		},
	}

	for groupName, testCases := range tests {
		t.Run(groupName, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					filename, cleanup := setupTestYamlFile(t, tc.yamlContent)
					defer cleanup()

					result, err := loadExampleYaml(filename, tc.replacements)

					if tc.errorMsg != nil {
						assert.Error(t, err)
						if !assert.Contains(t, err.Error(), *tc.errorMsg) {
							t.Logf("\nExpected error: %q\nActual error:   %q", *tc.errorMsg, err.Error())
						}
						return
					}

					assert.NoError(t, err)

					for _, expected := range tc.expected {
						if !assert.Contains(t, result, expected) {
							t.Logf("\nExpected to find: %q\nIn result:\n%s", expected, result)
						}
					}
				})
			}
		})
	}
}
