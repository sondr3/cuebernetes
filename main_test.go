package main

import (
	"flag"
	"io/fs"
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
		{
			name: "infrastructure kustomization",
			file: "testdata/clusters/infrastructure.cue",
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

func TestFindCueFiles(t *testing.T) {
	files, err := findCueFiles("testdata/apps")
	if err != nil {
		t.Fatalf("findCueFiles returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(files), files)
	}
	if files[0] != "testdata/apps/podinfo.cue" {
		t.Errorf("expected testdata/apps/podinfo.cue, got %s", files[0])
	}
}

func TestFindCueFilesRecursive(t *testing.T) {
	files, err := findCueFiles("testdata")
	if err != nil {
		t.Fatalf("findCueFiles returned error: %v", err)
	}
	if len(files) != 6 {
		t.Fatalf("expected 6 files, got %d: %v", len(files), files)
	}
}

func TestFindCueFilesNonexistent(t *testing.T) {
	_, err := findCueFiles("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
}

func TestWalkDirIgnores(t *testing.T) {
	tests := []struct {
		name    string
		entry   string
		isDir   bool
		wantErr error
	}{
		{"hidden dir", ".git", true, filepath.SkipDir},
		{"cue.mod", "cue.mod", true, filepath.SkipDir},
		{"normal dir", "src", true, nil},
		{"normal file", "main.go", false, nil},
		{"dot current", ".", true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := fakeDirEntry{name: tt.entry, isDir: tt.isDir}
			got := walkDirIgnores(entry)
			if got != tt.wantErr {
				t.Errorf("walkDirIgnores(%q) = %v, want %v", tt.entry, got, tt.wantErr)
			}
		})
	}
}

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.isDir }
func (f fakeDirEntry) Type() fs.FileMode          { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

func TestParseFileError(t *testing.T) {
	h := Handler{Manifests: make(map[string][]Manifest)}
	err := h.parseFile("nonexistent.cue")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestWrite(t *testing.T) {
	h := Handler{Manifests: map[string][]Manifest{
		"test.cue": {
			{Name: "ns", Value: []byte("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: test\n")},
			{Name: "repo", Value: []byte("apiVersion: source.toolkit.fluxcd.io/v1\nkind: OCIRepository\n")},
		},
	}}

	dir := t.TempDir()
	out := filepath.Join(dir, "output")

	err := h.Write(out, false)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(out, "test.yaml"))
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(content), "apiVersion: v1") {
		t.Error("output missing first manifest")
	}
	if !strings.Contains(string(content), "kind: OCIRepository") {
		t.Error("output missing second manifest")
	}
}

func TestWriteSplit(t *testing.T) {
	h := Handler{Manifests: map[string][]Manifest{
		"test.cue": {
			{Name: "ns", Value: []byte("apiVersion: v1\nkind: Namespace\n")},
			{Name: "repo", Value: []byte("apiVersion: source.toolkit.fluxcd.io/v1\nkind: OCIRepository\n")},
		},
	}}

	dir := t.TempDir()
	out := filepath.Join(dir, "output")

	err := h.Write(out, true)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(out, "test-ns.yaml")); err != nil {
		t.Errorf("split file test-ns.yaml not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "test-repo.yaml")); err != nil {
		t.Errorf("split file test-repo.yaml not found: %v", err)
	}
}

func TestWriteNestedOutputDir(t *testing.T) {
	h := Handler{Manifests: map[string][]Manifest{
		"test.cue": {
			{Name: "ns", Value: []byte("apiVersion: v1\nkind: Namespace\n")},
		},
	}}

	dir := t.TempDir()
	out := filepath.Join(dir, "nested", "deep", "path")

	err := h.Write(out, false)
	if err != nil {
		t.Fatalf("Write with nested output dir returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(out, "test.yaml")); err != nil {
		t.Errorf("output file not found: %v", err)
	}
}

func TestHandleYamlNoTabs(t *testing.T) {
	h := Handler{Manifests: make(map[string][]Manifest)}
	err := h.parseFile("testdata/clusters/infrastructure.cue")
	if err != nil {
		t.Fatalf("parseFile returned error: %v", err)
	}

	manifests, ok := h.Manifests["testdata/clusters/infrastructure.cue"]
	if !ok {
		t.Fatal("no manifests found for infrastructure.cue")
	}

	for _, m := range manifests {
		if strings.Contains(string(m.Value), "\t") {
			t.Errorf("manifest %q contains tabs", m.Name)
		}
	}
}

func TestStringifyManifests(t *testing.T) {
	manifests := []Manifest{
		{Name: "ns", Value: []byte("apiVersion: v1\nkind: Namespace\n")},
		{Name: "repo", Value: []byte("apiVersion: source.toolkit.fluxcd.io/v1\nkind: OCIRepository\n")},
	}

	got := StringifyManifests("test.cue", manifests)

	if !strings.HasPrefix(got, "# DO NOT EDIT -- generated from test.cue\n") {
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
