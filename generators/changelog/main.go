// Command changelog prints a release's CHANGELOG.md section to stdout.
//
//	go run ./generators/changelog --version=v0.41.0
//
// The release workflows use it to fill the release PR body and the GitHub release
// notes. It exits non-zero with a specific message when the section is missing or
// empty, so callers need no checks of their own.
package main

import (
	"flag"
	"fmt"
	"log"
	"path"
)

func main() {
	log.SetFlags(0)

	if err := execute(); err != nil {
		log.Fatal(err)
	}
}

func execute() error {
	var version, repoPath string

	flag.StringVar(&version, "version", "", "Release version to print notes for, with the v prefix, e.g. v0.41.0")
	flag.StringVar(&repoPath, "repo", "./", "Path to the repository root holding CHANGELOG.md")
	flag.Parse()

	if version == "" {
		return fmt.Errorf("version is a required flag")
	}

	notes, err := readNotes(path.Join(repoPath, changelogFile), version)
	if err != nil {
		return err
	}

	fmt.Println(notes)

	return nil
}
