package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	apiVersionShort = "v1alpha1"
	apiVersionFull  = "aiven.io/" + apiVersionShort
)

func main() {
	const (
		apiReferenceTargetFile = "docs/docs/api-reference.md"
	)
	f, err := os.OpenFile(apiReferenceTargetFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatal("unable to open target file: ", err)
	}
	defer f.Close()

	renderedApiDocs, err := initialRenderApiDocs()
	if err != nil {
		log.Fatal("unable to initially render api docs: ", err)
	}
	renderedApiDocs, err = fixInternalTypeAnchors(renderedApiDocs)
	if err != nil {
		log.Fatal("unable to fix anchors: ", err)
	}

	renderedApiDocs = fixPadding(renderedApiDocs)

	if _, err = f.Write(renderedApiDocs); err != nil {
		log.Fatal("unable to write apid docs to file: ", err)
	}
}

func initialRenderApiDocs() ([]byte, error) {
	tmpFile, err := os.CreateTemp(os.TempDir(), "gen-api-reference-*")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	genCmd := exec.Command(
		"bin/gen-crd-api-reference-docs",
		"-config", "hack/genrefs/config.json",
		"-template-dir", "hack/genrefs/template",
		"-api-dir", fmt.Sprintf("./api/%s", apiVersionShort),
		"-out-file", tmpFile.Name(),
	)

	if err := genCmd.Run(); err != nil {
		return nil, err
	}

	return io.ReadAll(tmpFile)
}

func fixInternalTypeAnchors(renderedApiDocs []byte) ([]byte, error) {
	aivenTypeAnchorRegexp, err := regexp.Compile(fmt.Sprintf(`\(#%s\.([a-zA-Z]*)\)`, regexp.QuoteMeta(apiVersionFull)))
	if err != nil {
		return nil, err
	}
	return aivenTypeAnchorRegexp.ReplaceAllFunc(renderedApiDocs, func(in []byte) []byte {
		fields := bytes.Split(bytes.Trim(in, "()"), []byte("."))
		if len(fields) == 0 {
			return nil
		}
		builder := bytes.NewBuffer(nil)
		builder.WriteString("(#")
		builder.Write(bytes.ToLower(fields[len(fields)-1]))
		builder.WriteString(")")

		return builder.Bytes()
	}), nil
}

// fixPadding headers, tables has to be padded with double new lines
func fixPadding(src []byte) []byte {
	isTable := false
	lines := strings.Split(string(src), "\n")
	for i, s := range lines {
		// Headers management
		if strings.HasPrefix(s, "#") {
			lines[i] = "\n" + s + "\n"
			continue
		}

		// Tables management
		isRow := strings.HasPrefix(s, "|")
		if isRow == isTable {
			// no need extra line between table lines
			continue
		}

		isTable = isRow && !isTable
		lines[i] = "\n" + s
	}

	// Removes extra newlines
	result := strings.Join(lines, "\n")
	result = regexp.MustCompile("\n{2,}").ReplaceAllString(result, "\n\n")
	return []byte(result)
}
