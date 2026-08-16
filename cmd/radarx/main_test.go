package main

import (
	"testing"

	"github.com/americooo/radarx/internal/model"
)

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

func TestNewAssetsFrom(t *testing.T) {
	newHost := model.Asset{Key: "new.example.com"}
	changedHost := model.Asset{Key: "changed.example.com"}
	removedHost := model.Asset{Key: "removed.example.com"}

	d := model.DiffResult{Changes: []model.Change{
		{Type: model.ChangeNew, After: &newHost},
		{Type: model.ChangeModified, After: &changedHost},
		{Type: model.ChangeRemoved, Before: &removedHost},
	}}

	got := newAssetsFrom(d)
	if len(got) != 1 {
		t.Fatalf("expected 1 new asset, got %d", len(got))
	}
	if got[0].Key != "new.example.com" {
		t.Errorf("expected new.example.com, got %s", got[0].Key)
	}
}

func TestNewAssetsFromEmpty(t *testing.T) {
	if got := newAssetsFrom(model.DiffResult{}); got != nil {
		t.Errorf("expected nil for empty diff, got %v", got)
	}
}
