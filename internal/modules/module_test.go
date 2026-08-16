package modules

import (
	"context"
	"sync"
	"testing"

	"github.com/americooo/radarx/internal/model"
)

// fakeModule is a minimal Module used to exercise the registry and
// orchestrator without touching any real detection logic. The orchestrator
// runs Run concurrently across assets, so seen is guarded by a mutex.
type fakeModule struct {
	name    string
	trigger Trigger
	emit    func(asset model.Asset) []model.Finding

	mu   sync.Mutex
	seen []model.Asset
}

func (f *fakeModule) Name() string       { return f.name }
func (f *fakeModule) Category() Category { return CategoryDiscovery }
func (f *fakeModule) Trigger() Trigger   { return f.trigger }

// Seen returns a snapshot of the assets Run has been called with so far.
func (f *fakeModule) Seen() []model.Asset {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]model.Asset(nil), f.seen...)
}

func (f *fakeModule) Run(ctx context.Context, target model.Target, asset model.Asset, state State) (<-chan model.Finding, error) {
	f.mu.Lock()
	f.seen = append(f.seen, asset)
	f.mu.Unlock()
	out := make(chan model.Finding, 4)
	go func() {
		defer close(out)
		if f.emit == nil {
			return
		}
		for _, fnd := range f.emit(asset) {
			select {
			case out <- fnd:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

func TestRegisterAndAll(t *testing.T) {
	before := len(All())

	m := &fakeModule{name: "test-registry-module", trigger: TriggerAllAssets}
	Register(m)

	after := All()
	if len(after) != before+1 {
		t.Fatalf("expected registry to grow by 1, got %d -> %d", before, len(after))
	}

	found := false
	for _, reg := range after {
		if reg.Name() == "test-registry-module" {
			found = true
		}
	}
	if !found {
		t.Fatal("registered module not found in All()")
	}
}
