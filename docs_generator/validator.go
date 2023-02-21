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
				OpenAPIV3Schema interface{} `yaml:"openAPIV3Schema"`
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

	// There is no yaml validator, turns into json
	jsonSchema, err := json.Marshal(spec.Spec.Versions[0].Schema.OpenAPIV3Schema)
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
