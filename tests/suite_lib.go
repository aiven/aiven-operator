package tests

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

var examplesDirPath = "../docs/docs/resources/examples"

type exampleYamlProcessor struct {
	docs []*ast.DocumentNode
}

func loadExampleYaml(name string, replacements map[string]string) (string, error) {
	p := &exampleYamlProcessor{}
	if err := p.loadFile(name); err != nil {
		return "", err
	}

	if err := p.applyReplacements(replacements); err != nil {
		return "", err
	}

	return p.serialize()
}

func (p *exampleYamlProcessor) loadFile(name string) error {
	filePath := path.Join(examplesDirPath, name)
	file, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	astFile, err := parser.ParseBytes(file, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}
	p.docs = astFile.Docs

	return nil
}

func (p *exampleYamlProcessor) applyReplacements(replacements map[string]string) error {
	for path, value := range replacements {
		docIndex := 0
		actualPath := path

		// Handle document index if specified (e.g., "doc[0].key")
		if strings.HasPrefix(path, "doc[") {
			endIdx := strings.Index(path, "]")
			if endIdx != -1 {
				// Extract document index
				if idx, err := strconv.Atoi(path[4:endIdx]); err == nil {
					docIndex = idx
					if len(path) > endIdx+2 && path[endIdx+1] == '.' {
						// Remove "doc[0]." from the path
						actualPath = path[endIdx+2:]
					} else if value == "REMOVE" {
						// Remove entire document if e.g. "doc[0]" is specified
						if docIndex < 0 || docIndex >= len(p.docs) {
							return fmt.Errorf("document index out of range: %d", docIndex)
						}
						p.docs = append(p.docs[:docIndex], p.docs[docIndex+1:]...)
						continue
					}
				}
			}
		}

		if docIndex < 0 || docIndex >= len(p.docs) {
			return fmt.Errorf("document index out of range: %d", docIndex)
		}

		if value == "REMOVE" {
			if err := p.removeNode(p.docs[docIndex], actualPath); err != nil {
				return err
			}
			continue
		}

		if err := p.updateNode(p.docs[docIndex], actualPath, value); err != nil {
			return err
		}
	}

	return nil
}

func (p *exampleYamlProcessor) serialize() (string, error) {
	var yamlContent strings.Builder
	for i, doc := range p.docs {
		if i > 0 {
			yamlContent.WriteString("---\n")
		}
		docYaml, err := yaml.Marshal(doc)
		if err != nil {
			return "", fmt.Errorf("failed to marshal YAML: %w", err)
		}
		yamlContent.Write(docYaml)
	}

	return yamlContent.String(), nil
}

func (p *exampleYamlProcessor) updateNode(doc *ast.DocumentNode, path, value string) error {
	fullPath := fmt.Sprintf("$.%s", path)
	yamlPath, err := yaml.PathString(fullPath)
	if err != nil {
		return fmt.Errorf("invalid path %s: %w", path, err)
	}

	oldNode, err := yamlPath.FilterNode(doc.Body)
	if err != nil {
		return fmt.Errorf("failed to filter node for path %s: %w", fullPath, err)
	}

	if oldNode == nil {
		return fmt.Errorf("node not found for path %s", fullPath)
	}

	newNode := &ast.StringNode{
		BaseNode: &ast.BaseNode{},
		Token:    oldNode.GetToken(),
		Value:    value,
	}

	docFile := &ast.File{Docs: []*ast.DocumentNode{doc}}
	if err := yamlPath.ReplaceWithNode(docFile, newNode); err != nil {
		return fmt.Errorf("failed to replace node: %w", err)
	}

	return nil
}

func (p *exampleYamlProcessor) removeNode(doc *ast.DocumentNode, path string) error {
	parentPath := path[:strings.LastIndex(path, ".")]
	lastKey := path[strings.LastIndex(path, ".")+1:]

	pp, err := yaml.PathString(fmt.Sprintf("$.%s", parentPath))
	if err != nil {
		return fmt.Errorf("failed to create parent path: %w", err)
	}

	parentNode, err := pp.FilterNode(doc.Body)
	if err != nil {
		return fmt.Errorf("failed to find parent node: %w", err)
	}

	if mapNode, ok := parentNode.(*ast.MappingNode); ok {
		for i, node := range mapNode.Values {
			if node.Key.String() == lastKey {
				mapNode.Values = append(mapNode.Values[:i], mapNode.Values[i+1:]...)
				break
			}
		}
	}

	return nil
}
