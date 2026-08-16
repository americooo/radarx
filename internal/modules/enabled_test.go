package modules

import (
	"context"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
)

func TestIsEnabledDefaultsToTrue(t *testing.T) {
	store := newFakeSettingsStore()
	if !IsEnabled(store, "some-module") {
		t.Fatal("expected a module with no stored preference to default to enabled")
	}
}

func TestSetEnabledFalseDisables(t *testing.T) {
	store := newFakeSettingsStore()
	if err := SetEnabled(store, "some-module", false); err != nil {
		t.Fatal(err)
	}
	if IsEnabled(store, "some-module") {
		t.Fatal("expected module to be disabled after SetEnabled(false)")
	}
}

func TestSetEnabledTrueReenables(t *testing.T) {
	store := newFakeSettingsStore()
	if err := SetEnabled(store, "some-module", false); err != nil {
		t.Fatal(err)
	}
	if err := SetEnabled(store, "some-module", true); err != nil {
		t.Fatal(err)
	}
	if !IsEnabled(store, "some-module") {
		t.Fatal("expected module to be re-enabled after SetEnabled(true)")
	}
}

func TestOrchestratorSkipsDisabledModule(t *testing.T) {
	m := &fakeModule{
		name:    "disable-me",
		trigger: TriggerAllAssets,
		emit: func(a model.Asset) []model.Finding {
			return []model.Finding{{Module: "disable-me"}}
		},
	}
	target := model.Target{ID: "t1", Root: "example.com"}
	store := newFakeSettingsStore()
	if err := SetEnabled(store, "disable-me", false); err != nil {
		t.Fatal(err)
	}

	withRegistry(t, []Module{m}, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var findings []model.Finding
		for f := range Run(ctx, target, nil, []model.Asset{asset("host.example.com")}, store) {
			findings = append(findings, f)
		}
		if len(findings) != 0 {
			t.Fatalf("expected disabled module to produce no findings, got %d", len(findings))
		}
		if seen := m.Seen(); len(seen) != 0 {
			t.Fatalf("expected disabled module's Run to never be called, got %+v", seen)
		}
	})
}
