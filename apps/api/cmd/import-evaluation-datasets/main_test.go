package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPrintsLocalOnlyHelpWithoutConfiguration(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	status := run([]string{"--help"}, os.Getwd, os.LookupEnv, &stdout, &stderr)
	if status != 0 {
		t.Fatalf("run(--help) status = %d, want zero", status)
	}
	if !strings.Contains(stdout.String(), "does not fetch sources") || stderr.Len() != 0 {
		t.Fatalf("help output is not a safe local-only contract: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestProjectAssetReaderStaysWithinRepositoryRoot(t *testing.T) {
	repositoryRoot, err := findRepositoryRootFromTestDirectory()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}
	reader := projectAssetReader{repositoryRoot: repositoryRoot}
	asset, err := reader.OpenAsset("data/corpora/brazil-lgpd/evaluation/brazil-lgpd-v1.jsonl")
	if err != nil {
		t.Fatalf("OpenAsset() error = %v", err)
	}
	defer asset.Close()
	if _, err := io.ReadAll(io.LimitReader(asset, 1)); err != nil {
		t.Fatalf("read local asset: %v", err)
	}
	if _, err := reader.OpenAsset("../AGENTS.md"); err == nil {
		t.Fatal("OpenAsset() accepted a path outside the repository root")
	}
}

func findRepositoryRootFromTestDirectory() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findRepositoryRoot(filepath.Clean(directory))
}
