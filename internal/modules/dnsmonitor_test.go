package modules

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
)

func drainDNSFindings(t *testing.T, m *DNSMonitorModule, target model.Target, state State) []model.Finding {
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

func baselineDNSModule() *DNSMonitorModule {
	return &DNSMonitorModule{
		lookupA: func(ctx context.Context, host string) ([]string, error) {
			return []string{"1.2.3.4"}, nil
		},
		lookupAAAA: func(ctx context.Context, host string) ([]string, error) {
			return []string{"2001:db8::1"}, nil
		},
		lookupMX: func(ctx context.Context, host string) ([]string, error) {
			return []string{"10 mail.example.com"}, nil
		},
		lookupTXT: func(ctx context.Context, host string) ([]string, error) {
			return []string{"v=spf1 include:_spf.example.com ~all"}, nil
		},
		lookupNS: func(ctx context.Context, host string) ([]string, error) {
			return []string{"ns1.example.com", "ns2.example.com"}, nil
		},
	}
}

func TestDNSMonitorFirstRunEstablishesBaselineNoFindings(t *testing.T) {
	m := baselineDNSModule()
	target := model.Target{Root: "example.com"}
	store := newFakeSettingsStore()
	state := NewState(store, "test")

	findings := drainDNSFindings(t, m, target, state)
	if len(findings) != 0 {
		t.Fatalf("expected no findings on first (baseline) run, got %d: %+v", len(findings), findings)
	}

	for _, recordType := range []string{"A", "AAAA", "MX", "TXT", "NS"} {
		if _, ok, err := state.Get("dns:" + recordType); err != nil || !ok {
			t.Fatalf("expected dns:%s to be stored after baseline run, ok=%v err=%v", recordType, ok, err)
		}
	}
}

func TestDNSMonitorARecordChangeProducesInfoFinding(t *testing.T) {
	m := baselineDNSModule()
	target := model.Target{Root: "example.com"}
	store := newFakeSettingsStore()
	state := NewState(store, "test")

	// Baseline run.
	drainDNSFindings(t, m, target, state)

	// Second run: only the A record changes.
	m.lookupA = func(ctx context.Context, host string) ([]string, error) {
		return []string{"5.6.7.8"}, nil
	}

	findings := drainDNSFindings(t, m, target, state)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for changed A record, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Module != "dns-monitor" {
		t.Fatalf("expected module dns-monitor, got %s", f.Module)
	}
	if f.Severity != model.SeverityInfo {
		t.Fatalf("expected Info severity for A record change, got %s", f.Severity)
	}
	if f.Title != "A DNS records changed" {
		t.Fatalf("unexpected title: %s", f.Title)
	}
	if f.Asset.Kind != model.KindSubdomain || f.Asset.Key != target.Root || f.Asset.Host != target.Root {
		t.Fatalf("unexpected finding asset: %+v", f.Asset)
	}
	if f.Evidence != "old=1.2.3.4 new=5.6.7.8" {
		t.Fatalf("unexpected evidence: %s", f.Evidence)
	}
}

func TestDNSMonitorMXRecordChangeProducesMediumFinding(t *testing.T) {
	m := baselineDNSModule()
	target := model.Target{Root: "example.com"}
	store := newFakeSettingsStore()
	state := NewState(store, "test")

	drainDNSFindings(t, m, target, state)

	m.lookupMX = func(ctx context.Context, host string) ([]string, error) {
		return []string{"10 mail.newvendor.com"}, nil
	}

	findings := drainDNSFindings(t, m, target, state)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for changed MX record, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Severity != model.SeverityMedium {
		t.Fatalf("expected Medium severity for MX record change, got %s", f.Severity)
	}
	if f.Title != "MX DNS records changed" {
		t.Fatalf("unexpected title: %s", f.Title)
	}
}

func TestDNSMonitorNoChangeProducesNoFindings(t *testing.T) {
	m := baselineDNSModule()
	target := model.Target{Root: "example.com"}
	store := newFakeSettingsStore()
	state := NewState(store, "test")

	drainDNSFindings(t, m, target, state)
	findings := drainDNSFindings(t, m, target, state)
	if len(findings) != 0 {
		t.Fatalf("expected no findings when nothing changed, got %d: %+v", len(findings), findings)
	}
}

func TestDNSMonitorOneRecordTypeErrorDoesNotStopOthers(t *testing.T) {
	m := baselineDNSModule()
	m.lookupMX = func(ctx context.Context, host string) ([]string, error) {
		return nil, errors.New("dns: no such host")
	}

	target := model.Target{Root: "example.com"}
	store := newFakeSettingsStore()
	state := NewState(store, "test")

	// Baseline run: MX fails, everything else should still be stored.
	findings := drainDNSFindings(t, m, target, state)
	if len(findings) != 0 {
		t.Fatalf("expected no findings on baseline run, got %d: %+v", len(findings), findings)
	}
	if _, ok, _ := state.Get("dns:MX"); ok {
		t.Fatal("expected dns:MX to not be stored when lookupMX errors")
	}
	for _, recordType := range []string{"A", "AAAA", "TXT", "NS"} {
		if _, ok, err := state.Get("dns:" + recordType); err != nil || !ok {
			t.Fatalf("expected dns:%s to be stored despite MX error, ok=%v err=%v", recordType, ok, err)
		}
	}

	// Second run: A record changes, MX still errors — should still see the
	// A finding and no crash/hang from the MX error.
	m.lookupA = func(ctx context.Context, host string) ([]string, error) {
		return []string{"9.9.9.9"}, nil
	}

	findings = drainDNSFindings(t, m, target, state)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (A change) despite MX error, got %d: %+v", len(findings), findings)
	}
	if findings[0].Title != "A DNS records changed" {
		t.Fatalf("unexpected finding title: %s", findings[0].Title)
	}
}

func TestDNSMonitorEmptyRootProducesNoFindings(t *testing.T) {
	m := baselineDNSModule()
	target := model.Target{Root: ""}
	state := NewState(newFakeSettingsStore(), "test")

	findings := drainDNSFindings(t, m, target, state)
	if len(findings) != 0 {
		t.Fatalf("expected no findings for empty target root, got %d", len(findings))
	}
}

func TestDNSMonitorModuleMetadata(t *testing.T) {
	m := &DNSMonitorModule{}
	if m.Name() != "dns-monitor" {
		t.Fatalf("unexpected name: %s", m.Name())
	}
	if m.Category() != CategoryDiscovery {
		t.Fatalf("unexpected category: %s", m.Category())
	}
	if m.Trigger() != TriggerScheduled {
		t.Fatalf("unexpected trigger: %s", m.Trigger())
	}
}

func TestDNSMonitorRegisteredByDefault(t *testing.T) {
	found := false
	for _, m := range All() {
		if m.Name() == "dns-monitor" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected dns-monitor module to be registered via init()")
	}
}
