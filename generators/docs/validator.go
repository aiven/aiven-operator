package main

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonschema"
)

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

// validateYAML validates yaml document by given crd
// Converts yaml to json, because there is no yaml validator
func validateYAML(crd []byte, document []byte) error {
	// Creates validation schema from CRD
	spec := new(crdSchema)
	if err := yaml.Unmarshal(crd, spec); err != nil {
		return fmt.Errorf("can't unmarshal CRD: %w", err)
	}

	schema := patchSchema(spec.Spec.Versions[0].Schema.OpenAPIV3Schema)

	// There is no yaml validator, turns into json
	jsonSchema, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("can't convert yaml to json: %w", err)
	}

	s := gojsonschema.NewSchemaLoader()
	yamlSchema, err := s.Compile(gojsonschema.NewBytesLoader(jsonSchema))
	if err != nil {
		return fmt.Errorf("can't compile yaml schema: %w", err)
	}

	// Validates given document
	jsonDocument, err := yaml.YAMLToJSON(document)
	if err != nil {
		return fmt.Errorf("can't convert yaml to json: %w", err)
	}

	r, err := yamlSchema.Validate(gojsonschema.NewBytesLoader(jsonDocument))
	if err != nil {
		return fmt.Errorf("can't validate yaml document: %w", err)
	}

	if !r.Valid() {
		return fmt.Errorf("yaml document is invalid: %v", r.Errors())
	}
	return nil
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
