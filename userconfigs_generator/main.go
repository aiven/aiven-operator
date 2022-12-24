package main

import (
	"flag"
	"log"
	"strings"

	"github.com/aiven/aiven-go-client/tools/exp/dist"
)

const destination = "./api/v1alpha1/userconfigs"

func main() {
	var serviceList string
	flag.StringVar(&serviceList, "services", "", "Comma separated service list of names to generate for")
	flag.Parse()

	// flags package does not provide validation
	if serviceList == "" {
		log.Fatal("--services i required")
	}

	err := generate(destination, dist.ServiceTypes, strings.Split(serviceList, ","))
	if err != nil {
		log.Fatal(err)
	}
}
