package modules

import (
	"context"
	"net/http"
	"testing"

	"github.com/americooo/radarx/internal/model"
)

func TestRateLimitSensitiveURLNoProtectionSignal(t *testing.T) {
	calls := 0
	m := &RateLimitModule{
		fetch: func(ctx context.Context, url string) (int, http.Header, string, error) {
			calls++
			return 200, http.Header{}, "<html>welcome back</html>", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://example.com/login"}
	findings := drainFindings(t, m, a)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Severity != model.SeverityMedium {
		t.Fatalf("expected Medium severity, got %s", f.Severity)
	}
	if f.Module != "rate-limit-signal" {
		t.Fatalf("expected module name rate-limit-signal, got %s", f.Module)
	}
	if f.Asset.Key != a.Key {
		t.Fatalf("expected finding asset to match input asset, got %+v", f.Asset)
	}
	if calls != 2 {
		t.Fatalf("expected fetch to be called exactly 2 times, got %d", calls)
	}
}

func TestRateLimitSensitiveURLRetryAfterHeader(t *testing.T) {
	calls := 0
	m := &RateLimitModule{
		fetch: func(ctx context.Context, url string) (int, http.Header, string, error) {
			calls++
			if calls == 1 {
				h := http.Header{}
				h.Set("Retry-After", "30")
				return 200, h, "ok", nil
			}
			return 200, http.Header{}, "ok", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://example.com/login"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings when Retry-After header present, got %d: %+v", len(findings), findings)
	}
	if calls != 2 {
		t.Fatalf("expected fetch to be called exactly 2 times, got %d", calls)
	}
}

func TestRateLimitSensitiveURL429Status(t *testing.T) {
	calls := 0
	m := &RateLimitModule{
		fetch: func(ctx context.Context, url string) (int, http.Header, string, error) {
			calls++
			if calls == 1 {
				return http.StatusTooManyRequests, http.Header{}, "slow down", nil
			}
			return 200, http.Header{}, "ok", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://example.com/admin"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings when 429 status observed, got %d: %+v", len(findings), findings)
	}
	if calls != 2 {
		t.Fatalf("expected fetch to be called exactly 2 times, got %d", calls)
	}
}

func TestRateLimitSensitiveURLCaptchaInBody(t *testing.T) {
	calls := 0
	m := &RateLimitModule{
		fetch: func(ctx context.Context, url string) (int, http.Header, string, error) {
			calls++
			return 200, http.Header{}, "<html>Please complete the CAPTCHA to continue</html>", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://example.com/signin"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings when captcha indicator present in body, got %d: %+v", len(findings), findings)
	}
	if calls != 2 {
		t.Fatalf("expected fetch to be called exactly 2 times, got %d", calls)
	}
}

func TestRateLimitSkipsNonSensitiveURL(t *testing.T) {
	m := &RateLimitModule{
		fetch: func(ctx context.Context, url string) (int, http.Header, string, error) {
			t.Fatal("fetch should not be called for a non-sensitive URL")
			return 0, nil, "", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://example.com/about"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for non-sensitive URL, got %d: %+v", len(findings), findings)
	}
}

func TestRateLimitSkipsNonEndpointAssets(t *testing.T) {
	m := &RateLimitModule{
		fetch: func(ctx context.Context, url string) (int, http.Header, string, error) {
			t.Fatal("fetch should not be called for a non-endpoint asset")
			return 0, nil, "", nil
		},
	}

	a := model.Asset{Kind: model.KindSubdomain, Key: "login.example.com", Host: "login.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for non-endpoint asset kind, got %d: %+v", len(findings), findings)
	}
}

func TestRateLimitDedupsAfterFirstUnprotectedFinding(t *testing.T) {
	calls := 0
	m := &RateLimitModule{
		fetch: func(ctx context.Context, url string) (int, http.Header, string, error) {
			calls++
			return 200, http.Header{}, "welcome", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://example.com/login"}
	state := NewState(newFakeSettingsStore(), "rate-limit-signal")

	ctx := context.Background()

	ch1, err := m.Run(ctx, model.Target{}, a, state)
	if err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	var first []model.Finding
	for f := range ch1 {
		first = append(first, f)
	}
	if len(first) != 1 {
		t.Fatalf("expected 1 finding on first run, got %d: %+v", len(first), first)
	}

	ch2, err := m.Run(ctx, model.Target{}, a, state)
	if err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}
	var second []model.Finding
	for f := range ch2 {
		second = append(second, f)
	}
	if len(second) != 0 {
		t.Fatalf("expected no finding on second run for same asset (dedup), got %d: %+v", len(second), second)
	}
	if calls != 2 {
		t.Fatalf("expected fetch to be called exactly 2 times total (second run should be skipped), got %d", calls)
	}
}

func TestRateLimitModuleMetadata(t *testing.T) {
	m := &RateLimitModule{}
	if m.Name() != "rate-limit-signal" {
		t.Fatalf("unexpected name: %s", m.Name())
	}
	if m.Category() != CategoryVulnSignal {
		t.Fatalf("unexpected category: %s", m.Category())
	}
	if m.Trigger() != TriggerNewAssetsOnly {
		t.Fatalf("unexpected trigger: %s", m.Trigger())
	}
}

func TestRateLimitRegisteredByDefault(t *testing.T) {
	found := false
	for _, m := range All() {
		if m.Name() == "rate-limit-signal" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected rate-limit-signal module to be registered via init()")
	}
}
