package main

import (
	"flag"
	"log"
	"strings"

	"github.com/aiven/aiven-go-client/tools/exp/dist"
)

const destination = "./api/v1alpha1/userconfig"

func main() {
	var serviceList, integrationList string
	flag.StringVar(&serviceList, "services", "", "Comma separated service list of names to generate for")
	flag.StringVar(&integrationList, "integrations", "", "Comma separated integrations list of names to generate for")
	flag.Parse()

	if serviceList+integrationList == "" {
		log.Fatal("--service or --integrations must be provided")
	}

	if serviceList != "" {
		err := generate(destination+"/service", dist.ServiceTypes, strings.Split(serviceList, ","))
		if err != nil {
			log.Fatal(err)
		}
	}

	if integrationList != "" {
		err := generate(destination+"/integration", dist.IntegrationTypes, strings.Split(integrationList, ","))
		if err != nil {
			log.Fatal(err)
		}
	}
}
