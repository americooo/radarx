package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckInScope(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scope.txt")
	if err := os.WriteFile(path, []byte("example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := checkInScope(path, "api.example.com"); err != nil {
		t.Errorf("expected api.example.com to be in scope, got error: %v", err)
	}
	if err := checkInScope(path, "evil.com"); err == nil {
		t.Error("expected evil.com to be rejected as out of scope")
	}
}

func TestCheckInScopeMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.txt")
	if err := checkInScope(path, "example.com"); err == nil {
		t.Error("expected error when scope file doesn't exist")
	}
}
