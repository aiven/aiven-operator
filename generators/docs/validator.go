package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonschema"
)

// exampleKind gets example's kind to validate with kind's schema
type exampleKind struct {
	Kind string `yaml:"kind"`
}

// crdSchema schema
type crdSchema struct {
	Spec struct {
		Versions []struct {
			Schema struct {
				OpenAPIV3Schema map[string]any `yaml:"openAPIV3Schema"`
			}
		}
	}
}

// validateYAML validates yaml document
// Converts yaml to json, because there is no yaml validator
func validateYAML(validators map[string]schemaValidator, document []byte) error {
	for _, d := range bytes.Split(document, []byte("---")) {
		if len(bytes.TrimSpace(d)) == 0 {
			continue
		}

		example := new(exampleKind)
		err := yaml.Unmarshal(d, example)
		if err != nil {
			return err
		}

		// Validates given document
		jsonDocument, err := yaml.YAMLToJSON(d)
		if err != nil {
			return fmt.Errorf("can't convert yaml to json: %w", err)
		}

		validate, ok := validators[example.Kind]
		if !ok {
			return fmt.Errorf("validator for kind %q not found", example.Kind)
		}

		err = validate(jsonDocument)
		if err != nil {
			return err
		}
	}

	return nil
}

type schemaValidator func([]byte) error

// newSchemaValidator creates a validator for a given kind from its CRD
func newSchemaValidator(kind string, crd []byte) (schemaValidator, error) {
	// Creates validation schema from CRD
	spec := new(crdSchema)
	if err := yaml.Unmarshal(crd, spec); err != nil {
		return nil, fmt.Errorf("can't unmarshal CRD: %w", err)
	}

	schema := patchSchema(spec.Spec.Versions[0].Schema.OpenAPIV3Schema)

	// There is no yaml validator, turns into json
	jsonSchema, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("can't convert yaml to json: %w", err)
	}

	s := gojsonschema.NewSchemaLoader()
	yamlSchema, err := s.Compile(gojsonschema.NewBytesLoader(jsonSchema))
	if err != nil {
		return nil, fmt.Errorf("can't compile yaml schema: %w", err)
	}

	return func(document []byte) error {
		// Validates given document
		jsonDocument, err := yaml.YAMLToJSON(document)
		if err != nil {
			return fmt.Errorf("can't convert yaml to json: %w", err)
		}

		r, err := yamlSchema.Validate(gojsonschema.NewBytesLoader(jsonDocument))
		if err != nil {
			return fmt.Errorf("%q can't create validator: %w", kind, err)
		}

		if !r.Valid() {
			return fmt.Errorf("%q yaml is invalid: %v", kind, r.Errors())
		}
		return nil
	}, nil
}

// patchSchema adds additionalProperties=false, and schema for metadata.
// If not to do so, new properties allowed on validation,
// but won't work when applied with kubectl
func patchSchema(m map[string]any) map[string]any {
	if m["type"].(string) != "object" {
		return m
	}

	if p, ok := m["properties"]; ok {
		prop := p.(map[string]any)
		for k, v := range prop {
			vv := v.(map[string]any)

			// metadata schema is empty, replaces with a good one
			if k == "metadata" && len(vv) == 1 {
				vv["properties"] = map[string]map[string]any{
					"name":      {"type": "string", "minLength": 1},
					"namespace": {"type": "string", "minLength": 1},
					"uid":       {"type": "string", "minLength": 1},
				}
				vv["required"] = []string{"name"}
				vv["additionalProperties"] = false

				// This should not go recursive for metadata.
				// On the next call "properties" will panic on map[string]any type asserting
				continue
			}

			prop[k] = patchSchema(vv)
		}
		m["properties"] = prop
	}

	if i, ok := m["items"]; ok {
		items := i.(map[string]any)
		m["items"] = patchSchema(items)
	}

	if _, ok := m["additionalProperties"]; !ok {
		m["additionalProperties"] = false
	}
	return m
}

// setUsageExamples assigns and validates examples for a given schema
func setUsageExamples(examplesDir string, validators map[string]schemaValidator, schema *schemaType) error {
	matches, err := filepath.Glob(fmt.Sprintf("%s/%s.*", examplesDir, strings.ToLower(schema.Kind)))
	if err != nil {
		return err
	}

	// Adds usage example
	// Mkdocs can embed files, but we use docker, we can include files only within the dir
	// So if they are moved out elsewhere, this will be broken
	for _, match := range matches {
		exampleData, _ := os.ReadFile(match)
		if exampleData != nil {
			err = validateYAML(validators, exampleData)
			if err != nil {
				return fmt.Errorf("%q: %w", match, err)
			}
			example := usageExample{
				Value: strings.TrimSpace(string(exampleData)),
			}
			title := strings.Split(match, ".")
			// nolint:mnd // splits foo.title.yaml and takes the middle part
			if len(title) > 2 {
				// Just takes the part after the kind name
				example.Title = title[1]
			}

			schema.UsageExamples = append(schema.UsageExamples, example)
		}
	}
	return nil
}
