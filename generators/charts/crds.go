package main

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	cp "github.com/otiai10/copy"
)

const (
	crdSourceDir      = "config/crd/bases/"
	crdDestinationDir = "templates"
)

// copyCRDs copies CRDs, like MySQL, Postgres, etc
func copyCRDs(operatorPath, crdCharts string) error {
	srcCRDs := path.Join(operatorPath, crdSourceDir)
	dstCRDs := path.Join(crdCharts, crdDestinationDir)
	err := cp.Copy(srcCRDs, dstCRDs)
	if err != nil {
		return err
	}

	return filepath.Walk(dstCRDs, fixCRD)
}

// fixCRD escapes "{{" so it's not rendered by the template engine
func fixCRD(filePath string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	content = bytes.ReplaceAll(content, []byte("`{{"), []byte("{{`{{"))
	content = bytes.ReplaceAll(content, []byte("}}`"), []byte("}}`}}"))
	return writeFile(filePath, content)
}
