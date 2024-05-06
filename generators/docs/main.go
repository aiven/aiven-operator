package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

const (
	// crdDirPath CRDs location
	crdDirPath = "./config/crd/bases/"
	// docsDirPath CRDs docs location to export to
	docsDirPath     = "./docs/docs/api-reference/"
	examplesDirPath = docsDirPath + "examples"
	// allCRDYaml contains all crds, should ignore
	allCRDYaml = "aiven.io_crd-all.gen.yaml"
)

func main() {
	err := generate(crdDirPath, docsDirPath, examplesDirPath)
	if err != nil {
		log.Fatal(err)
	}
}

func generate(crdDir, docsDir, examplesDir string) error {
	crd, err := os.ReadDir(crdDir)
	if err != nil {
		return err
	}

	schemas := make(map[string]*schemaType)
	validators := make(map[string]schemaValidator)
	for _, crd := range crd {
		if crd.IsDir() {
			continue
		}

		if crd.Name() == allCRDYaml {
			continue
		}

		crdData, err := os.ReadFile(path.Join(crdDir, crd.Name()))
		if err != nil {
			return err
		}

		schema, err := parseSchema(crdData)
		if err != nil {
			return fmt.Errorf("%q generation error: %w", crd.Name(), err)
		}

		validators[schema.Kind], err = newSchemaValidator(schema.Kind, crdData)
		if err != nil {
			return err
		}

		schemas[schema.Kind] = schema
	}

	for _, s := range schemas {
		err = setUsageExamples(examplesDir, validators, s)
		if err != nil {
			return err
		}

		data, err := renderTemplate(schemaTemplate, s)
		if err != nil {
			return err
		}

		dest := path.Join(docsDir, strings.ToLower(s.Kind)+".md")
		err = os.WriteFile(dest, data, 0o644)
		if err != nil {
			return err
		}
	}

	return nil
}
