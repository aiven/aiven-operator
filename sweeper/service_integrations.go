package main

import (
	"context"
	"fmt"

	avngen "github.com/aiven/go-client-codegen"
)

type serviceIntegrationEndpointsSweeper struct {
	client avngen.Client
}

func (sweeper *serviceIntegrationEndpointsSweeper) Name() string {
	return "Service integration endpoints"
}

// Sweep deletes all service integration endpoints in a project
func (sweeper *serviceIntegrationEndpointsSweeper) Sweep(ctx context.Context, projectName string) error {
	endpoints, err := sweeper.client.ServiceIntegrationEndpointList(ctx, projectName)
	if err != nil {
		return fmt.Errorf("error retrieving a list of integration endpoints: %w", err)
	}

	for _, s := range endpoints {
		if err := sweeper.client.ServiceIntegrationEndpointDelete(ctx, projectName, s.EndpointId); avngen.OmitNotFound(err) != nil {
			return fmt.Errorf("error deleting service integration endpoint %q: %w", s.EndpointName, err)
		}
	}

	return nil
}
