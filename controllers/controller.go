package controllers

import (
	"context"
	"fmt"
	"github.com/aiven/aiven-go-client"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Controller struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	AivenClient *aiven.Client
}

// GetAivenClient retrieves an Aiven client
func (c *Controller) InitAivenClient(req ctrl.Request, ctx context.Context, log logr.Logger) error {
	if c.AivenClient != nil {
		return nil
	}

	var token string

	// Check if aiven-token secret exists
	secret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: "aiven-token", Namespace: req.Namespace}, secret)
	if err != nil {
		log.Error(err, "aiven-token secret is missing, required by the Aiven client")
		return err
	}

	if v, ok := secret.Data["token"]; ok {
		token = string(v)
	} else {
		return fmt.Errorf("cannot initialize Aiven client, kubernetes secret has no `token` key")
	}

	c.AivenClient, err = aiven.NewTokenClient(token, "k8s-operator/")
	if err != nil {
		return err
	}

	return nil
}

// contains checks if string slice contains an element
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// labelsForProject returns the labels for selecting the resources
// belonging to the given project CR name.
func labelsForProject(name string) map[string]string {
	return map[string]string{"app": "project", "project_cr": name}
}
