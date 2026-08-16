package modules

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// dnsRecordTypes is the fixed set of record types this module watches, in
// the order they're queried — deterministic order makes test expectations
// (and log output) easy to reason about.
var dnsRecordTypes = []string{"A", "AAAA", "MX", "TXT", "NS"}

// DNSMonitorModule watches a target's root-domain DNS records (A, AAAA, MX,
// TXT, NS) for changes between scan cycles — a purely passive signal read
// straight from DNS, no active scanning. Records are queried at the domain
// level (target.Root), not per-subdomain, so asset is ignored.
//
// The lookup* fields are injectable so tests never touch real DNS.
type DNSMonitorModule struct {
	lookupA    func(ctx context.Context, host string) ([]string, error)
	lookupAAAA func(ctx context.Context, host string) ([]string, error)
	lookupMX   func(ctx context.Context, host string) ([]string, error)
	lookupTXT  func(ctx context.Context, host string) ([]string, error)
	lookupNS   func(ctx context.Context, host string) ([]string, error)
}

func init() {
	Register(&DNSMonitorModule{
		lookupA:    defaultLookupA,
		lookupAAAA: defaultLookupAAAA,
		lookupMX:   defaultLookupMX,
		lookupTXT:  defaultLookupTXT,
		lookupNS:   defaultLookupNS,
	})
}

func (m *DNSMonitorModule) Name() string       { return "dns-monitor" }
func (m *DNSMonitorModule) Category() Category { return CategoryDiscovery }
func (m *DNSMonitorModule) Trigger() Trigger   { return TriggerScheduled }

// Run queries A, AAAA, MX, TXT and NS records for target.Root and diffs each
// against what was stored last time. asset is ignored (zero-value) —
// TriggerScheduled modules operate on the whole target, not one asset.
func (m *DNSMonitorModule) Run(ctx context.Context, target model.Target, asset model.Asset, state State) (<-chan model.Finding, error) {
	out := make(chan model.Finding, 1)

	go func() {
		defer close(out)

		if target.Root == "" {
			return
		}

		for _, recordType := range dnsRecordTypes {
			values, err := m.lookup(ctx, recordType, target.Root)
			if err != nil {
				// One record type failing (e.g. no MX record) is normal,
				// not an error — skip it and keep checking the rest.
				continue
			}

			sorted := append([]string(nil), values...)
			sort.Strings(sorted)
			newVal := strings.Join(sorted, ",")

			key := "dns:" + recordType
			oldVal, ok, err := state.Get(key)
			if err != nil {
				continue
			}

			if !ok {
				// First time seeing this record type — establish the
				// baseline, don't report a finding.
				_ = state.Set(key, newVal)
				continue
			}

			if oldVal == newVal {
				continue
			}

			finding := model.Finding{
				Module:      m.Name(),
				Severity:    severityFor(recordType),
				Asset:       model.Asset{Kind: model.KindSubdomain, Key: target.Root, Host: target.Root},
				Title:       fmt.Sprintf("%s DNS records changed", recordType),
				Description: fmt.Sprintf("%s records for %s changed from %q to %q", recordType, target.Root, oldVal, newVal),
				Evidence:    fmt.Sprintf("old=%s new=%s", oldVal, newVal),
				TakenAt:     time.Now().UTC(),
			}
			select {
			case out <- finding:
			case <-ctx.Done():
				return
			}

			_ = state.Set(key, newVal)
		}
	}()

	return out, nil
}

// lookup dispatches to the injected lookup func for recordType.
func (m *DNSMonitorModule) lookup(ctx context.Context, recordType, host string) ([]string, error) {
	switch recordType {
	case "A":
		return m.lookupA(ctx, host)
	case "AAAA":
		return m.lookupAAAA(ctx, host)
	case "MX":
		return m.lookupMX(ctx, host)
	case "TXT":
		return m.lookupTXT(ctx, host)
	case "NS":
		return m.lookupNS(ctx, host)
	default:
		return nil, fmt.Errorf("unknown DNS record type %q", recordType)
	}
}

// severityFor ranks MX/NS changes higher than A/AAAA/TXT: a new MX or NS
// record can mean a new third-party service now controls mail or delegation
// for the domain, worth a closer look; A/AAAA/TXT churn is routine.
func severityFor(recordType string) model.Severity {
	switch recordType {
	case "MX", "NS":
		return model.SeverityMedium
	default:
		return model.SeverityInfo
	}
}

func defaultLookupA(ctx context.Context, host string) ([]string, error) {
	resolver := &net.Resolver{}
	addrs, err := resolver.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.To4() != nil {
			out = append(out, addr)
		}
	}
	return out, nil
}

func defaultLookupAAAA(ctx context.Context, host string) ([]string, error) {
	resolver := &net.Resolver{}
	addrs, err := resolver.LookupHost(ctx, host)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.To4() == nil {
			out = append(out, addr)
		}
	}
	return out, nil
}

func defaultLookupMX(ctx context.Context, host string) ([]string, error) {
	resolver := &net.Resolver{}
	records, err := resolver.LookupMX(ctx, host)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(records))
	for _, r := range records {
		out = append(out, fmt.Sprintf("%d %s", r.Pref, strings.TrimSuffix(r.Host, ".")))
	}
	return out, nil
}

func defaultLookupTXT(ctx context.Context, host string) ([]string, error) {
	resolver := &net.Resolver{}
	return resolver.LookupTXT(ctx, host)
}

func defaultLookupNS(ctx context.Context, host string) ([]string, error) {
	resolver := &net.Resolver{}
	records, err := resolver.LookupNS(ctx, host)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(records))
	for _, r := range records {
		out = append(out, strings.TrimSuffix(r.Host, "."))
	}
	return out, nil
}
