package main

import "testing"

func TestCheckInScope(t *testing.T) {
	roots := []string{"example.com"}

	if err := checkInScope(roots, "api.example.com"); err != nil {
		t.Errorf("expected api.example.com to be in scope, got error: %v", err)
	}
	if err := checkInScope(roots, "evil.com"); err == nil {
		t.Error("expected evil.com to be rejected as out of scope")
	}
}

func TestCheckInScopeMissingFile(t *testing.T) {
	if err := checkInScope(nil, "example.com"); err == nil {
		t.Error("expected error when scope has no entries")
	}
}
