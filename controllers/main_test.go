// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package controllers

import (
	"os"
	"testing"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestMain(m *testing.M) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	// Keep the ServiceUser API lag retries fast.
	emptyPasswordRetryDelay = time.Millisecond
	notFoundRetryDelay = time.Millisecond
	os.Exit(m.Run())
}
