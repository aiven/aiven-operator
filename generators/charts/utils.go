package main

import (
	"bytes"
	"os"

	"gopkg.in/yaml.v3"
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
	return os.WriteFile(filePath, b, 0644)
}
