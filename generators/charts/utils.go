package main

import (
	"bytes"
	"os"

	"github.com/goccy/go-yaml"
)

const compactIndent = 2

func marshalCompactYaml(in any) (*bytes.Buffer, error) {
	var b bytes.Buffer
	y := yaml.NewEncoder(&b)
	y.SetIndent(compactIndent)
	err := y.Encode(in)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func writeFile(filePath string, b []byte) error {
	return os.WriteFile(filePath, b, 0o644)
}
