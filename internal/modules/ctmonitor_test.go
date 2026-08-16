package modules

import (
	"context"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
)

func drainCTFindings(t *testing.T, m *CTMonitorModule, target model.Target, state State) []model.Finding {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := m.Run(ctx, target, model.Asset{}, state)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	var out []model.Finding
	for f := range ch {
		out = append(out, f)
	}
	return out
}

func TestCTMonitorNewCertificatesProduceFindings(t *testing.T) {
	body := []byte(`[
		{"id": 1001, "issuer_name": "Let's Encrypt", "common_name": "api.example.com", "name_value": "api.example.com"},
		{"id": 1002, "issuer_name": "Let's Encrypt", "common_name": "www.example.com", "name_value": "www.example.com\nexample.com"}
	]`)

	m := &CTMonitorModule{
		fetch: func(ctx context.Context, url string) ([]byte, error) {
			return body, nil
		},
	}

	target := model.Target{Root: "example.com"}
	state := NewState(newFakeSettingsStore(), "test")

	findings := drainCTFindings(t, m, target, state)

	// id=1001 -> 1 host, id=1002 -> 2 hosts (www.example.com, example.com)
	if len(findings) != 3 {
		t.Fatalf("expected 3 findings, got %d: %+v", len(findings), findings)
	}
	for _, f := range findings {
		if f.Module != "ct-monitor" {
			t.Fatalf("expected module ct-monitor, got %s", f.Module)
		}
		if f.Severity != model.SeverityInfo {
			t.Fatalf("expected Info severity, got %s", f.Severity)
		}
		if f.Asset.Kind != model.KindSubdomain {
			t.Fatalf("expected KindSubdomain asset, got %s", f.Asset.Kind)
		}
		if f.Title != "New certificate observed in CT logs" {
			t.Fatalf("unexpected title: %s", f.Title)
		}
	}
}

func TestCTMonitorDedupesAcrossRuns(t *testing.T) {
	body := []byte(`[
		{"id": 2001, "issuer_name": "Let's Encrypt", "common_name": "api.example.com", "name_value": "api.example.com"}
	]`)

	fetchCalls := 0
	m := &CTMonitorModule{
		fetch: func(ctx context.Context, url string) ([]byte, error) {
			fetchCalls++
			return body, nil
		},
	}

	target := model.Target{Root: "example.com"}
	store := newFakeSettingsStore()
	state := NewState(store, "test")

	first := drainCTFindings(t, m, target, state)
	if len(first) != 1 {
		t.Fatalf("expected 1 finding on first run, got %d: %+v", len(first), first)
	}

	// Second run against the same state (crt.sh returns the same entries again).
	second := drainCTFindings(t, m, target, state)
	if len(second) != 0 {
		t.Fatalf("expected 0 findings on second run (already-seen cert id), got %d: %+v", len(second), second)
	}

	if fetchCalls != 2 {
		t.Fatalf("expected fetch to be called once per Run call, got %d", fetchCalls)
	}
}

func TestCTMonitorEmptyResponseProducesNoFindingsNoError(t *testing.T) {
	m := &CTMonitorModule{
		fetch: func(ctx context.Context, url string) ([]byte, error) {
			return []byte(`[]`), nil
		},
	}

	target := model.Target{Root: "example.com"}
	state := NewState(newFakeSettingsStore(), "test")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := m.Run(ctx, target, model.Asset{}, state)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	var findings []model.Finding
	for f := range ch {
		findings = append(findings, f)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings for empty crt.sh response, got %d", len(findings))
	}
}

func TestCTMonitorTrulyEmptyBodyProducesNoFindingsNoError(t *testing.T) {
	m := &CTMonitorModule{
		fetch: func(ctx context.Context, url string) ([]byte, error) {
			return []byte(``), nil
		},
	}

	target := model.Target{Root: "example.com"}
	state := NewState(newFakeSettingsStore(), "test")

	findings := drainCTFindings(t, m, target, state)
	if len(findings) != 0 {
		t.Fatalf("expected no findings for empty body, got %d", len(findings))
	}
}

func TestCTMonitorModuleMetadata(t *testing.T) {
	m := &CTMonitorModule{}
	if m.Name() != "ct-monitor" {
		t.Fatalf("unexpected name: %s", m.Name())
	}
	if m.Category() != CategoryDiscovery {
		t.Fatalf("unexpected category: %s", m.Category())
	}
	if m.Trigger() != TriggerScheduled {
		t.Fatalf("unexpected trigger: %s", m.Trigger())
	}
}

func TestCTMonitorRegisteredByDefault(t *testing.T) {
	found := false
	for _, m := range All() {
		if m.Name() == "ct-monitor" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected ct-monitor module to be registered via init()")
	}
}
