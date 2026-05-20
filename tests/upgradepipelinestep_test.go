//go:build grafana

package tests

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/aiven/go-client-codegen/handler/upgradepipeline"
	"github.com/stretchr/testify/require"

	"github.com/aiven/aiven-operator/api/v1alpha1"
)

func TestUpgradePipelineStep(t *testing.T) {
	t.Parallel()
	defer recoverPanic(t)

	// GIVEN
	ctx, cancel := testCtx()
	defer cancel()

	sourceName := randName("ups-src")
	destinationName := randName("ups-dst")
	stepName := randName("ups")
	adoptedStepName := randName("ups-adopt")

	sourceYml, err := loadExampleYaml("grafana.yaml", map[string]string{
		"metadata.name":                  sourceName,
		"spec.project":                   cfg.Project,
		"spec.cloudName":                 cfg.PrimaryCloudName,
		"spec.connInfoSecretTarget.name": sourceName,
	})
	require.NoError(t, err)
	destinationYml, err := loadExampleYaml("grafana.yaml", map[string]string{
		"metadata.name":                  destinationName,
		"spec.project":                   cfg.Project,
		"spec.cloudName":                 cfg.PrimaryCloudName,
		"spec.connInfoSecretTarget.name": destinationName,
	})
	require.NoError(t, err)

	s := NewSession(ctx, k8sClient)
	defer s.Destroy(t)

	require.NoError(t, s.Apply(sourceYml+"---\n"+destinationYml))

	source := new(v1alpha1.Grafana)
	require.NoError(t, s.GetRunning(source, sourceName))

	destination := new(v1alpha1.Grafana)
	require.NoError(t, s.GetRunning(destination, destinationName))

	// WHEN
	stepYml, err := loadExampleYaml("upgradepipelinestep.yaml", map[string]string{
		"metadata.name":                stepName,
		"spec.organizationId":          cfg.AccountID,
		"spec.sourceProjectName":       cfg.Project,
		"spec.sourceServiceName":       sourceName,
		"spec.destinationProjectName":  cfg.Project,
		"spec.destinationServiceName":  destinationName,
		"spec.autoValidationDelayDays": "REMOVE",
	})
	require.NoError(t, err)
	require.NoError(t, s.Apply(stepYml))

	step := new(v1alpha1.UpgradePipelineStep)
	require.NoError(t, s.GetRunning(step, stepName))

	// THEN
	require.NotEmpty(t, step.Status.ID)

	stepAvn, err := avnGen.UpgradePipelineStepGet(ctx, cfg.AccountID, step.Status.ID)
	require.NoError(t, err)
	require.Equal(t, cfg.Project, stepAvn.SourceProjectName)
	require.Equal(t, sourceName, stepAvn.SourceServiceName)
	require.Equal(t, cfg.Project, stepAvn.DestinationProjectName)
	require.Equal(t, destinationName, stepAvn.DestinationServiceName)
	require.Equal(t, 7, stepAvn.AutoValidationDelayDays)

	updatedStepYml, err := loadExampleYaml("upgradepipelinestep.yaml", map[string]string{
		"metadata.name":                stepName,
		"spec.organizationId":          cfg.AccountID,
		"spec.sourceProjectName":       cfg.Project,
		"spec.sourceServiceName":       sourceName,
		"spec.destinationProjectName":  cfg.Project,
		"spec.destinationServiceName":  destinationName,
		"spec.autoValidationDelayDays": "3",
	})
	require.NoError(t, err)
	require.NoError(t, s.Apply(updatedStepYml))

	updatedStep := new(v1alpha1.UpgradePipelineStep)
	require.NoError(t, s.GetRunning(updatedStep, stepName))
	require.Eventually(t, func() bool {
		out, err := avnGen.UpgradePipelineStepGet(ctx, cfg.AccountID, updatedStep.Status.ID)
		if err != nil {
			return false
		}

		return out.AutoValidationDelayDays == 3
	}, 15*time.Second, 1*time.Second)

	require.NoError(t, s.Delete(updatedStep, func() error {
		_, err := avnGen.UpgradePipelineStepGet(ctx, cfg.AccountID, updatedStep.Status.ID)
		return err
	}))
	require.Eventually(t, func() bool {
		out, err := avnGen.UpgradePipelineStepList(ctx, cfg.AccountID)
		if err != nil {
			return false
		}

		count := 0
		for _, step := range out.Steps {
			if step.SourceProjectName == cfg.Project &&
				step.SourceServiceName == sourceName &&
				step.DestinationProjectName == cfg.Project &&
				step.DestinationServiceName == destinationName {
				count++
			}
		}

		return count == 0
	}, 15*time.Second, 1*time.Second)

	adoptedInitialDelay := 5
	adoptedUpdatedDelay := 2
	adopted, err := avnGen.UpgradePipelineStepCreate(ctx, cfg.AccountID, &upgradepipeline.UpgradePipelineStepCreateIn{
		AutoValidationDelayDays: &adoptedInitialDelay,
		DestinationProjectName:  cfg.Project,
		DestinationServiceName:  destinationName,
		SourceProjectName:       cfg.Project,
		SourceServiceName:       sourceName,
	})
	require.NoError(t, err)
	require.NotEmpty(t, adopted.StepId)
	adoptedStepID := adopted.StepId
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), deleteTimeout)
		defer cleanupCancel()

		_ = avnGen.UpgradePipelineStepDelete(cleanupCtx, cfg.AccountID, adoptedStepID)
	}()

	adoptedStepYml, err := loadExampleYaml("upgradepipelinestep.yaml", map[string]string{
		"metadata.name":                adoptedStepName,
		"spec.organizationId":          cfg.AccountID,
		"spec.sourceProjectName":       cfg.Project,
		"spec.sourceServiceName":       sourceName,
		"spec.destinationProjectName":  cfg.Project,
		"spec.destinationServiceName":  destinationName,
		"spec.autoValidationDelayDays": strconv.Itoa(adoptedInitialDelay),
	})
	require.NoError(t, err)
	require.NoError(t, s.Apply(adoptedStepYml))

	adoptedStep := new(v1alpha1.UpgradePipelineStep)
	require.NoError(t, s.GetRunning(adoptedStep, adoptedStepName))
	require.Equal(t, adoptedStepID, adoptedStep.Status.ID)
	require.Eventually(t, func() bool {
		out, err := avnGen.UpgradePipelineStepList(ctx, cfg.AccountID)
		if err != nil {
			return false
		}

		count := 0
		for _, step := range out.Steps {
			if step.SourceProjectName == cfg.Project &&
				step.SourceServiceName == sourceName &&
				step.DestinationProjectName == cfg.Project &&
				step.DestinationServiceName == destinationName {
				count++
			}
		}

		return count == 1
	}, 15*time.Second, 1*time.Second)

	updatedAdoptedStepYml, err := loadExampleYaml("upgradepipelinestep.yaml", map[string]string{
		"metadata.name":                adoptedStepName,
		"spec.organizationId":          cfg.AccountID,
		"spec.sourceProjectName":       cfg.Project,
		"spec.sourceServiceName":       sourceName,
		"spec.destinationProjectName":  cfg.Project,
		"spec.destinationServiceName":  destinationName,
		"spec.autoValidationDelayDays": strconv.Itoa(adoptedUpdatedDelay),
	})
	require.NoError(t, err)
	require.NoError(t, s.Apply(updatedAdoptedStepYml))

	updatedAdoptedStep := new(v1alpha1.UpgradePipelineStep)
	require.NoError(t, s.GetRunning(updatedAdoptedStep, adoptedStepName))
	require.Eventually(t, func() bool {
		out, err := avnGen.UpgradePipelineStepGet(ctx, cfg.AccountID, updatedAdoptedStep.Status.ID)
		if err != nil {
			return false
		}

		return out.AutoValidationDelayDays == adoptedUpdatedDelay
	}, 15*time.Second, 1*time.Second)

	require.NoError(t, s.Delete(updatedAdoptedStep, func() error {
		_, err := avnGen.UpgradePipelineStepGet(ctx, cfg.AccountID, updatedAdoptedStep.Status.ID)
		return err
	}))
}
