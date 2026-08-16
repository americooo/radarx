package modules

import (
	"context"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// withRegistry runs fn with the global registry temporarily replaced,
// restoring the original afterwards, so orchestrator tests don't depend on
// (or interfere with) modules registered by other files' init()s.
func withRegistry(t *testing.T, mods []Module, fn func()) {
	t.Helper()
	orig := registry
	registry = mods
	defer func() { registry = orig }()
	fn()
}

func asset(host string) model.Asset {
	return model.Asset{Kind: model.KindSubdomain, Key: host, Host: host}
}

func TestOrchestratorRoutesByTrigger(t *testing.T) {
	newOnly := &fakeModule{
		name:    "new-only",
		trigger: TriggerNewAssetsOnly,
		emit: func(a model.Asset) []model.Finding {
			return []model.Finding{{Module: "new-only", Asset: a, TakenAt: time.Now()}}
		},
	}
	allAssetsMod := &fakeModule{
		name:    "all-assets",
		trigger: TriggerAllAssets,
		emit: func(a model.Asset) []model.Finding {
			return []model.Finding{{Module: "all-assets", Asset: a, TakenAt: time.Now()}}
		},
	}
	scheduled := &fakeModule{
		name:    "scheduled",
		trigger: TriggerScheduled,
	}

	newAssets := []model.Asset{asset("new.example.com")}
	allAssets := []model.Asset{asset("new.example.com"), asset("old.example.com")}

	withRegistry(t, []Module{newOnly, allAssetsMod, scheduled}, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var findings []model.Finding
		for f := range Run(ctx, newAssets, allAssets) {
			findings = append(findings, f)
		}

		newOnlySeen := newOnly.Seen()
		if len(newOnlySeen) != 1 || newOnlySeen[0].Host != "new.example.com" {
			t.Fatalf("expected new-only module to see exactly the new asset, got %+v", newOnlySeen)
		}
		if seen := allAssetsMod.Seen(); len(seen) != 2 {
			t.Fatalf("expected all-assets module to see all assets, got %+v", seen)
		}
		if seen := scheduled.Seen(); len(seen) != 2 {
			t.Fatalf("expected scheduled module to currently see all assets, got %+v", seen)
		}

		if len(findings) != 3 { // 1 from new-only, 2 from all-assets
			t.Fatalf("expected 3 findings total, got %d: %+v", len(findings), findings)
		}
	})
}

func TestOrchestratorEmptyAssetsSkipsModule(t *testing.T) {
	m := &fakeModule{name: "new-only", trigger: TriggerNewAssetsOnly}

	withRegistry(t, []Module{m}, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var findings []model.Finding
		for f := range Run(ctx, nil, []model.Asset{asset("old.example.com")}) {
			findings = append(findings, f)
		}
		if len(findings) != 0 {
			t.Fatalf("expected no findings, got %d", len(findings))
		}
		if seen := m.Seen(); len(seen) != 0 {
			t.Fatalf("expected module to see no assets when newAssets is empty, got %+v", seen)
		}
	})
}
