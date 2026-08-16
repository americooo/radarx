package modules

import (
	"context"
	"sync"
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

// fakeSettingsStore is an in-memory SettingsStore for tests — no SQLite, no
// disk I/O, just a guarded map.
type fakeSettingsStore struct {
	mu   sync.Mutex
	data map[string]string
}

func newFakeSettingsStore() *fakeSettingsStore {
	return &fakeSettingsStore{data: make(map[string]string)}
}

func (s *fakeSettingsStore) GetSetting(key string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.data[key]
	return v, ok, nil
}

func (s *fakeSettingsStore) SetSetting(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
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
		emit: func(a model.Asset) []model.Finding {
			return []model.Finding{{Module: "scheduled", Asset: a, TakenAt: time.Now()}}
		},
	}

	newAssets := []model.Asset{asset("new.example.com")}
	allAssets := []model.Asset{asset("new.example.com"), asset("old.example.com")}
	target := model.Target{ID: "t1", Root: "example.com"}

	withRegistry(t, []Module{newOnly, allAssetsMod, scheduled}, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var findings []model.Finding
		for f := range Run(ctx, target, newAssets, allAssets, newFakeSettingsStore()) {
			findings = append(findings, f)
		}

		newOnlySeen := newOnly.Seen()
		if len(newOnlySeen) != 1 || newOnlySeen[0].Host != "new.example.com" {
			t.Fatalf("expected new-only module to see exactly the new asset, got %+v", newOnlySeen)
		}
		if seen := allAssetsMod.Seen(); len(seen) != 2 {
			t.Fatalf("expected all-assets module to see all assets, got %+v", seen)
		}
		// TriggerScheduled modules run exactly once per cycle, with a
		// zero-value asset — they're meant to work off target, not a
		// specific discovered asset.
		if seen := scheduled.Seen(); len(seen) != 1 || seen[0].Host != "" || seen[0].Key != "" {
			t.Fatalf("expected scheduled module to run once with a zero-value asset, got %+v", seen)
		}

		if len(findings) != 4 { // 1 new-only + 2 all-assets + 1 scheduled
			t.Fatalf("expected 4 findings total, got %d: %+v", len(findings), findings)
		}
	})
}

func TestOrchestratorEmptyAssetsSkipsModule(t *testing.T) {
	m := &fakeModule{name: "new-only", trigger: TriggerNewAssetsOnly}
	target := model.Target{ID: "t1", Root: "example.com"}

	withRegistry(t, []Module{m}, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var findings []model.Finding
		for f := range Run(ctx, target, nil, []model.Asset{asset("old.example.com")}, newFakeSettingsStore()) {
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

func TestModuleStateIsNamespacedPerModule(t *testing.T) {
	store := newFakeSettingsStore()
	a := NewState(store, "module-a")
	b := NewState(store, "module-b")

	if err := a.Set("hash", "aaa"); err != nil {
		t.Fatal(err)
	}
	if err := b.Set("hash", "bbb"); err != nil {
		t.Fatal(err)
	}

	got, ok, err := a.Get("hash")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got != "aaa" {
		t.Fatalf("expected module-a's own value, got %q (ok=%v)", got, ok)
	}

	got, ok, err = b.Get("hash")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got != "bbb" {
		t.Fatalf("expected module-b's own value, got %q (ok=%v)", got, ok)
	}

	if _, ok, _ := a.Get("nonexistent"); ok {
		t.Fatal("expected missing key to report ok=false")
	}
}
