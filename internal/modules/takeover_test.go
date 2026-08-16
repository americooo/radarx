package modules

import (
	"context"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
)

func drainFindings(t *testing.T, m Module, a model.Asset) []model.Finding {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := m.Run(ctx, a)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	var out []model.Finding
	for f := range ch {
		out = append(out, f)
	}
	return out
}

func TestTakeoverMatchedCNAMEAndSignature(t *testing.T) {
	m := &TakeoverModule{
		lookupCNAME: func(ctx context.Context, host string) (string, error) {
			return "dangling.s3.amazonaws.com", nil
		},
		fetch: func(ctx context.Context, url string) (int, string, error) {
			return 404, "<html>NoSuchBucket: the bucket does not exist</html>", nil
		},
	}

	a := model.Asset{Kind: model.KindSubdomain, Key: "assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Severity != model.SeverityHigh {
		t.Fatalf("expected High severity, got %s", f.Severity)
	}
	if f.Module != "subdomain-takeover" {
		t.Fatalf("expected module name subdomain-takeover, got %s", f.Module)
	}
	if f.Asset.Key != a.Key {
		t.Fatalf("expected finding asset to match input asset, got %+v", f.Asset)
	}
}

func TestTakeoverCNAMEMatchesButBodyDoesNot(t *testing.T) {
	m := &TakeoverModule{
		lookupCNAME: func(ctx context.Context, host string) (string, error) {
			return "dangling.s3.amazonaws.com", nil
		},
		fetch: func(ctx context.Context, url string) (int, string, error) {
			return 200, "<html>everything is fine here</html>", nil
		},
	}

	a := model.Asset{Kind: model.KindSubdomain, Key: "assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings when body signature doesn't match, got %d", len(findings))
	}
}

func TestTakeoverNoCNAME(t *testing.T) {
	m := &TakeoverModule{
		lookupCNAME: func(ctx context.Context, host string) (string, error) {
			return "", nil
		},
		fetch: func(ctx context.Context, url string) (int, string, error) {
			t.Fatal("fetch should not be called when there is no CNAME")
			return 0, "", nil
		},
	}

	a := model.Asset{Kind: model.KindSubdomain, Key: "assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings when CNAME is empty, got %d", len(findings))
	}
}

func TestTakeoverCNAMEDoesNotMatchAnyFingerprint(t *testing.T) {
	m := &TakeoverModule{
		lookupCNAME: func(ctx context.Context, host string) (string, error) {
			return "lb.internal.example.net", nil
		},
		fetch: func(ctx context.Context, url string) (int, string, error) {
			t.Fatal("fetch should not be called when CNAME matches no fingerprint")
			return 0, "", nil
		},
	}

	a := model.Asset{Kind: model.KindSubdomain, Key: "assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for non-matching CNAME, got %d", len(findings))
	}
}

func TestTakeoverSkipsNonSubdomainAssets(t *testing.T) {
	calls := 0
	m := &TakeoverModule{
		lookupCNAME: func(ctx context.Context, host string) (string, error) {
			calls++
			return "dangling.s3.amazonaws.com", nil
		},
		fetch: func(ctx context.Context, url string) (int, string, error) {
			return 404, "NoSuchBucket", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for non-subdomain asset kind, got %d", len(findings))
	}
	if calls != 0 {
		t.Fatalf("expected lookupCNAME not to be called for non-subdomain asset, got %d calls", calls)
	}
}

func TestTakeoverModuleMetadata(t *testing.T) {
	m := &TakeoverModule{}
	if m.Name() != "subdomain-takeover" {
		t.Fatalf("unexpected name: %s", m.Name())
	}
	if m.Category() != CategoryDiscovery {
		t.Fatalf("unexpected category: %s", m.Category())
	}
	if m.Trigger() != TriggerNewAssetsOnly {
		t.Fatalf("unexpected trigger: %s", m.Trigger())
	}
}

func TestTakeoverRegisteredByDefault(t *testing.T) {
	found := false
	for _, m := range All() {
		if m.Name() == "subdomain-takeover" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected subdomain-takeover module to be registered via init()")
	}
}
