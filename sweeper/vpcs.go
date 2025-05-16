package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	avngen "github.com/aiven/go-client-codegen"
)

type vpcsSweeper struct {
	client avngen.Client
}

func (sweeper *vpcsSweeper) Name() string {
	return "VPCs"
}

// Sweep deletes VPCs within a project
func (sweeper *vpcsSweeper) Sweep(ctx context.Context, projectName string) error {
	vpcs, err := sweeper.client.VpcList(ctx, projectName)
	if err != nil {
		return fmt.Errorf("error retrieving a list of VPCs: %w", err)
	}

	for _, v := range vpcs {
		// If VPC is being deleted, skip it
		if v.State == "DELETING" {
			continue
		}
		// VPCs cannot be deleted if there is a service in it, or if it is moving out of it
		// (e.g. service was deleted from the VPC). Thus, we need to use a retry mechanism to delete the VPC
		err := waitForTaskToComplete(ctx, func() (bool, error) {
			if _, vpcDeleteErr := sweeper.client.VpcDelete(ctx, projectName, v.ProjectVpcId); vpcDeleteErr != nil {
				if isCriticalVpcDeleteError(vpcDeleteErr) {
					return false, fmt.Errorf("error fetching VPC %s: %w", v.ProjectVpcId, vpcDeleteErr)
				}
				log.Printf("VPC in cloud %q (ID: %s) is not ready for deletion yet", v.CloudName, v.ProjectVpcId)
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			return fmt.Errorf("error deleting VPC in cloud %q (ID: %s): %w", v.CloudName, v.ProjectVpcId, err)
		}
	}

	return nil
}

// isCriticalVpcDeleteError returns true if the given error has any status code other than 409
func isCriticalVpcDeleteError(err error) bool {
	var e avngen.Error

	return errors.As(err, &e) && e.Status != http.StatusConflict
}

// waitForTaskToCompleteInterval is the interval to wait before running a task again
const waitForTaskToCompleteInterval = time.Second * 10

// waitForTaskToComplete waits for a task to complete
func waitForTaskToComplete(ctx context.Context, f func() (bool, error)) (err error) {
	retry := false

outer:
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context timeout while retrying operation, error=%q", ctx.Err().Error())
		case <-time.After(waitForTaskToCompleteInterval):
			retry, err = f()
			if err != nil {
				return err
			}
			if !retry {
				break outer
			}
		}
	}

	return nil
}
