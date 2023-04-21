package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

const (
	// srcDirPath CRDs location
	srcDirPath = "./config/crd/bases/"
	// dstDirPath CRDs docs location to export to
	dstDirPath  = "./docs/docs/api-reference/"
	examplesDir = dstDirPath + "examples"
	// allCRDYaml contains all crds, should ignore
	allCRDYaml = "aiven.io_crd-all.gen.yaml"
)

func main() {
	err := generate(srcDirPath, dstDirPath)
	if err != nil {
		log.Fatal(err)
	}
}

func generate(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if entry.Name() == allCRDYaml {
			continue
		}

		kind, err := parseSchema(path.Join(srcDir, entry.Name()), examplesDir)
		if err != nil {
			return fmt.Errorf("%q generation error: %w", entry.Name(), err)
		}

		data, err := renderTemplate(schemaTemplate, kind)
		if err != nil {
			return err
		}

		dest := path.Join(dstDir, strings.ToLower(kind.Kind)+".md")
		err = os.WriteFile(dest, data, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
