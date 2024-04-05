package main

import (
	"flag"
	"log"
	"strings"

	"github.com/aiven/go-api-schemas/pkg/dist"
)

const destination = "./api/v1alpha1/userconfig"

func main() {
	var serviceList, integrationList, integrationEndpointList string
	flag.StringVar(&serviceList, "services", "", "Comma separated list of service names to generate")
	flag.StringVar(&integrationList, "integrations", "", "Comma separated list of integration names to generate")
	flag.StringVar(&integrationEndpointList, "integration-endpoints", "", "Comma separated list of integration endpoint names to generate")
	flag.Parse()

	if serviceList+integrationList+integrationEndpointList == "" {
		log.Fatal("--service, --integrations or --integration-endpoints must be provided")
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

	if integrationEndpointList != "" {
		err := generate(destination+"/integrationendpoints", dist.IntegrationEndpointTypes, strings.Split(integrationEndpointList, ","))
		if err != nil {
			log.Fatal(err)
		}
	}
}
