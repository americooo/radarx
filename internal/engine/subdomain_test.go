package engine

import (
	"context"
	"reflect"
	"testing"

	"github.com/americooo/radarx/internal/model"
)

func TestDedupe(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", nil, nil},
		{"no dupes", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"dupes preserve first-seen order", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"all same", []string{"x", "x", "x"}, []string{"x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := dedupe(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("dedupe(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestEnumerateSubdomainsCancelledContext ensures the DNS brute-force loop
// respects an already-cancelled context and returns without hanging or
// touching the network (jobs are never sent once ctx.Done() fires).
func TestEnumerateSubdomainsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assets := EnumerateSubdomains(ctx, "example.com", []string{"www", "api"}, 4)
	if len(assets) != 0 {
		t.Fatalf("expected no assets with a cancelled context, got %d", len(assets))
	}
}

func TestMergeSubdomains(t *testing.T) {
	a := []model.Asset{
		{Kind: model.KindSubdomain, Key: "www.example.com", Host: "www.example.com"},
		{Kind: model.KindSubdomain, Key: "api.example.com", Host: "api.example.com"},
	}
	b := []model.Asset{
		{Kind: model.KindSubdomain, Key: "api.example.com", Host: "api.example.com"},
		{Kind: model.KindSubdomain, Key: "dev.example.com", Host: "dev.example.com"},
	}

	merged := mergeSubdomains(a, b)
	if len(merged) != 3 {
		t.Fatalf("expected 3 deduplicated hosts, got %d: %+v", len(merged), merged)
	}

	seen := make(map[string]bool)
	for _, asset := range merged {
		if seen[asset.Host] {
			t.Fatalf("duplicate host %q in merged result", asset.Host)
		}
		seen[asset.Host] = true
	}
	for _, host := range []string{"www.example.com", "api.example.com", "dev.example.com"} {
		if !seen[host] {
			t.Fatalf("expected host %q in merged result", host)
		}
	}
}

func TestMergeSubdomainsEmpty(t *testing.T) {
	merged := mergeSubdomains(nil, nil)
	if len(merged) != 0 {
		t.Fatalf("expected empty result, got %d", len(merged))
	}
}
