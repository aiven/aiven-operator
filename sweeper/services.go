package main

import (
	"context"
	"fmt"
	"strings"

	avngen "github.com/aiven/go-client-codegen"
	"github.com/aiven/go-client-codegen/handler/service"
)

type servicesSweeper struct {
	client        avngen.Client
	sweepPrefixes []string
}

func (sweeper *servicesSweeper) Name() string {
	return "services"
}

// Sweep deletes services in a project
func (sweeper *servicesSweeper) Sweep(ctx context.Context, projectName string) error {
	services, err := sweeper.client.ServiceList(ctx, projectName)
	if err != nil {
		return fmt.Errorf("error retrieving a list of services : %w", err)
	}

	for _, s := range services {
		// only delete services that have a specified prefix in their name
		if !hasPrefixAny(s.ServiceName, sweeper.sweepPrefixes) {
			continue
		}

		// if service termination_protection is on service cannot be deleted
		// update service and turn termination_protection off
		if s.TerminationProtection {
			terminationProtection := false
			_, err := sweeper.client.ServiceUpdate(ctx, projectName, s.ServiceName, &service.ServiceUpdateIn{
				TerminationProtection: &terminationProtection,
			})
			if err != nil {
				return fmt.Errorf("error disabling `termination_protection` for service %q: %w", s.ServiceName, err)
			}
		}

		if err := sweeper.client.ServiceDelete(ctx, projectName, s.ServiceName); avngen.OmitNotFound(err) != nil {
			return fmt.Errorf("error deleting service %s: %w", s.ServiceName, err)
		}
	}

	return nil
}

func hasPrefixAny(s string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
