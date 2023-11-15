package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	err := execute()
	if err != nil {
		log.Fatal(err)
	}
}

// execute updates chart's files, prepares release
func execute() error {
	var version, operatorCharts, crdCharts string

	flag.StringVar(&version, "version", "", "Operator chart version, e.g. v0.7.0")
	flag.StringVar(&operatorCharts, "operator-charts", "", "The path with operator repo")
	flag.StringVar(&crdCharts, "crd-charts", "../", "The path with operator charts repo")
	flag.Parse()

	if operatorCharts == "" {
		return fmt.Errorf("operator-charts is required flag")
	}

	if crdCharts == "" {
		return fmt.Errorf("crd-charts is required flag")
	}

	return generate(version, "./", operatorCharts, crdCharts)
}

func generate(version, operatorPath, operatorCharts, crdCharts string) error {
	if version != "" {
		err := updateVersion(version, operatorPath, operatorCharts, crdCharts)
		if err != nil {
			return err
		}
	}

	err := updateWebhooks(operatorPath, operatorCharts)
	if err != nil {
		return err
	}

	err = updateClusterRole(operatorPath, operatorCharts)
	if err != nil {
		return err
	}

	// Changelog is generated from old CRDs.
	// Reads them first, and then compares with updated files.
	commitChangelog, err := updateChangelog(operatorPath, crdCharts)
	if err != nil {
		return err
	}

	err = copyCRDs(operatorPath, crdCharts)
	if err != nil {
		return err
	}

	return commitChangelog()
}
