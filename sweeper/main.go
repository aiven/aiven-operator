package main

import (
	"context"
	"log"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/kelseyhightower/envconfig"
)

type sweepConfig struct {
	Token         string   `envconfig:"AIVEN_TOKEN" required:"true"`
	Project       string   `envconfig:"AIVEN_PROJECT_NAME" required:"true"`
	SweepPrefixes []string `envconfig:"AIVEN_SWEEP_PREFIX" default:"test"`
	DebugLogging  bool     `envconfig:"ENABLE_DEBUG_LOGGING"`
}

type sweeper interface {
	Name() string
	Sweep(ctx context.Context, projectName string) error
}

func main() {
	envVars := new(sweepConfig)
	ctx := context.Background()
	err := envconfig.Process("", envVars)
	if err != nil {
		log.Fatalf("error processing environment variables: %v\n", err)
	}

	// generate a new client
	client, err := avngen.NewClient(avngen.TokenOpt(envVars.Token), avngen.DebugOpt(envVars.DebugLogging))
	if err != nil {
		log.Fatalf("error creating aiven client: %v\n", err)
	}

	sweepers := []sweeper{
		&servicesSweeper{client: client, sweepPrefixes: envVars.SweepPrefixes},
		&vpcsSweeper{client},
		&serviceIntegrationEndpointsSweeper{client},
	}

	for _, sweeper := range sweepers {
		log.Printf("Sweeping %s\n", sweeper.Name())

		err := sweeper.Sweep(ctx, envVars.Project)
		if err != nil {
			log.Fatalf("error sweeping %s: %v\n", sweeper.Name(), err)
		}
	}
}
