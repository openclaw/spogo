//go:build windows

package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReplaceConfigReturnsPermanentFailure(t *testing.T) {
	dir := t.TempDir()
	err := replaceConfigFile(filepath.Join(dir, "missing"), filepath.Join(dir, "config"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected original missing-source error, got %v", err)
	}
}

func TestReplaceConfigWaitsForWindowsReader(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "new"), filepath.Join(dir, "config")
	for path, contents := range map[string]string{source: "new", destination: "old"} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	reader, err := os.Open(destination)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	done := make(chan error, 1)
	go func() { done <- replaceConfigFile(source, destination) }()
	select {
	case err := <-done:
		t.Fatalf("replacement finished while reader held the file: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "new" {
		t.Fatalf("replacement contents: %q, %v", data, err)
	}
}
