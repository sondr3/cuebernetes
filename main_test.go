package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestParseFile(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{
			name: "named fields",
			file: "testdata/apps/podinfo.cue",
		},
		{
			name: "named fields with multiple imports",
			file: "testdata/infrastructure/controllers/cert-manager.cue",
		},
		{
			name: "unnamed namespace",
			file: "testdata/infrastructure/controllers/cert-manager-files/cert-manager-namespace.cue",
		},
		{
			name: "unnamed repo",
			file: "testdata/infrastructure/controllers/cert-manager-files/cert-manager-repo.cue",
		},
		{
			name: "unnamed helm release",
			file: "testdata/infrastructure/controllers/cert-manager-files/cert-manager-helm.cue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Handler{Manifests: make(map[string][]Manifest)}
			err := h.parseFile(tt.file)
			if err != nil {
				t.Fatalf("parseFile(%s) returned error: %v", tt.file, err)
			}

			manifests, ok := h.Manifests[tt.file]
			if !ok {
				t.Fatalf("no manifests found for %s", tt.file)
			}

			got := StringifyManifests(tt.file, manifests)
			golden := strings.TrimSuffix(tt.file, filepath.Ext(tt.file)) + ".golden"

			if *update {
				if err := os.WriteFile(golden, []byte(got), 0644); err != nil {
					t.Fatalf("failed to update golden file: %v", err)
				}
			}

			expected, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("failed to read golden file %s (run with -update to create): %v", golden, err)
			}

			if got != string(expected) {
				t.Errorf("output mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", tt.file, got, string(expected))
			}
		})
	}
}

func TestStringifyManifests(t *testing.T) {
	manifests := []Manifest{
		{Name: "ns", Value: []byte("apiVersion: v1\nkind: Namespace\n")},
		{Name: "repo", Value: []byte("apiVersion: source.toolkit.fluxcd.io/v1\nkind: OCIRepository\n")},
	}

	got := StringifyManifests("test.cue", manifests)

	if !strings.HasPrefix(got, "# generated from test.cue -- DO NOT EDIT\n") {
		t.Errorf("missing header comment")
	}
	if !strings.Contains(got, "---\n") {
		t.Errorf("missing separator between manifests")
	}
	if !strings.Contains(got, "apiVersion: v1") {
		t.Errorf("missing first manifest")
	}
	if !strings.Contains(got, "kind: OCIRepository") {
		t.Errorf("missing second manifest")
	}
}
