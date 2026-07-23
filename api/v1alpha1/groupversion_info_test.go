// Copyright (c) 2024 Aiven, Helsinki, Finland. https://aiven.io/

package v1alpha1

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// TestSchemeRegistersAllCRDKinds guards the central addKnownTypes list against omissions.
// Every kind that has a generated CRD must resolve in the scheme built by AddToScheme.
func TestSchemeRegistersAllCRDKinds(t *testing.T) {
	const crdDir = "../../config/crd/bases"

	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}

	entries, err := os.ReadDir(crdDir)
	if err != nil {
		t.Fatalf("read CRD dir %s: %v", crdDir, err)
	}

	checked := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}

		b, err := os.ReadFile(filepath.Join(crdDir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}

		var crd struct {
			Spec struct {
				Group string `json:"group"`
				Names struct {
					Kind string `json:"kind"`
				} `json:"names"`
			} `json:"spec"`
		}
		if err := yaml.Unmarshal(b, &crd); err != nil {
			t.Fatalf("unmarshal %s: %v", e.Name(), err)
		}

		if crd.Spec.Group != GroupVersion.Group {
			continue
		}
		if crd.Spec.Names.Kind == "" {
			t.Fatalf("%s: CRD has empty spec.names.kind", e.Name())
		}

		gvk := GroupVersion.WithKind(crd.Spec.Names.Kind)
		if !scheme.Recognizes(gvk) {
			t.Errorf("kind %q has a CRD (%s) but is not registered in addKnownTypes", crd.Spec.Names.Kind, e.Name())
		}
		checked++
	}

	if checked == 0 {
		t.Fatalf("no CRDs found under %s; check the path", crdDir)
	}
}
