package modules

import (
	"context"
	"strings"
	"testing"

	"github.com/americooo/radarx/internal/model"
)

func htmlWithScripts(scripts ...string) []byte {
	body := "<html><head>"
	for _, s := range scripts {
		body += `<script src="` + s + `"></script>`
	}
	body += "</head><body></body></html>"
	return []byte(body)
}

func TestJSDiffBaselineNoFindingButHashSaved(t *testing.T) {
	pageURL := "https://example.com/"
	jsA := "https://example.com/app.js"
	jsB := "https://example.com/vendor.js"

	m := &JSDiffModule{
		fetchURL: func(ctx context.Context, url string) (int, []byte, error) {
			switch url {
			case pageURL:
				return 200, htmlWithScripts(jsA, jsB), nil
			case jsA:
				return 200, []byte("console.log('app v1');"), nil
			case jsB:
				return 200, []byte("console.log('vendor v1');"), nil
			}
			t.Fatalf("unexpected fetch: %s", url)
			return 0, nil, nil
		},
	}

	store := newFakeSettingsStore()
	state := NewState(store, "test")

	a := model.Asset{Kind: model.KindEndpoint, Key: pageURL, Host: "example.com"}

	ch, err := m.Run(context.Background(), model.Target{}, a, state)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	var findings []model.Finding
	for f := range ch {
		findings = append(findings, f)
	}

	if len(findings) != 0 {
		t.Fatalf("expected no findings on baseline scan, got %d: %+v", len(findings), findings)
	}
	if _, ok, _ := state.Get("hash:" + jsA); !ok {
		t.Fatalf("expected hash for %s to be saved after baseline scan", jsA)
	}
	if _, ok, _ := state.Get("hash:" + jsB); !ok {
		t.Fatalf("expected hash for %s to be saved after baseline scan", jsB)
	}
}

func TestJSDiffUnchangedContentNoFinding(t *testing.T) {
	pageURL := "https://example.com/"
	jsA := "https://example.com/app.js"
	jsContent := []byte("console.log('app v1');")

	m := &JSDiffModule{
		fetchURL: func(ctx context.Context, url string) (int, []byte, error) {
			switch url {
			case pageURL:
				return 200, htmlWithScripts(jsA), nil
			case jsA:
				return 200, jsContent, nil
			}
			t.Fatalf("unexpected fetch: %s", url)
			return 0, nil, nil
		},
	}

	store := newFakeSettingsStore()
	state := NewState(store, "test")
	a := model.Asset{Kind: model.KindEndpoint, Key: pageURL, Host: "example.com"}

	// First scan: baseline.
	ch, err := m.Run(context.Background(), model.Target{}, a, state)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	var findings []model.Finding
	for f := range ch {
		findings = append(findings, f)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings on baseline scan, got %d", len(findings))
	}

	// Second scan: identical content.
	ch, err = m.Run(context.Background(), model.Target{}, a, state)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	findings = nil
	for f := range ch {
		findings = append(findings, f)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings when JS content is unchanged, got %d: %+v", len(findings), findings)
	}
}

func TestJSDiffChangedContentEmitsFinding(t *testing.T) {
	pageURL := "https://example.com/"
	jsA := "https://example.com/app.js"

	currentContent := []byte("console.log('app v1');")
	m := &JSDiffModule{
		fetchURL: func(ctx context.Context, url string) (int, []byte, error) {
			switch url {
			case pageURL:
				return 200, htmlWithScripts(jsA), nil
			case jsA:
				return 200, currentContent, nil
			}
			t.Fatalf("unexpected fetch: %s", url)
			return 0, nil, nil
		},
	}

	store := newFakeSettingsStore()
	state := NewState(store, "test")
	a := model.Asset{Kind: model.KindEndpoint, Key: pageURL, Host: "example.com"}

	// First scan: baseline.
	ch, err := m.Run(context.Background(), model.Target{}, a, state)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for range ch {
	}

	// Second scan: still unchanged.
	ch, err = m.Run(context.Background(), model.Target{}, a, state)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	for range ch {
	}

	// Third scan: content changed.
	currentContent = []byte("console.log('app v2 - totally different');")
	ch, err = m.Run(context.Background(), model.Target{}, a, state)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	var findings []model.Finding
	for f := range ch {
		findings = append(findings, f)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding after content change, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Severity != model.SeverityLow {
		t.Fatalf("expected Low severity, got %s", f.Severity)
	}
	if f.Module != "js-diff" {
		t.Fatalf("expected module js-diff, got %s", f.Module)
	}
	if f.Asset.Key != jsA {
		t.Fatalf("expected finding asset key %s, got %s", jsA, f.Asset.Key)
	}
	if !containsAll(f.Evidence, "old sha256=", "new sha256=") {
		t.Fatalf("expected evidence to contain both old and new sha256, got %q", f.Evidence)
	}
}

func TestJSDiffResolvesRelativeURL(t *testing.T) {
	pageURL := "https://example.com/some/page.html"
	relativeSrc := "/app.js"
	expectedFullURL := "https://example.com/app.js"

	var fetchedJSURL string
	m := &JSDiffModule{
		fetchURL: func(ctx context.Context, url string) (int, []byte, error) {
			switch url {
			case pageURL:
				return 200, htmlWithScripts(relativeSrc), nil
			case expectedFullURL:
				fetchedJSURL = url
				return 200, []byte("console.log('hi');"), nil
			}
			t.Fatalf("unexpected fetch: %s", url)
			return 0, nil, nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: pageURL, Host: "example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings on baseline scan, got %d", len(findings))
	}
	if fetchedJSURL != expectedFullURL {
		t.Fatalf("expected relative src to resolve to %s, got %s", expectedFullURL, fetchedJSURL)
	}
}

func TestJSDiffSkipsNonEndpointAssets(t *testing.T) {
	calls := 0
	m := &JSDiffModule{
		fetchURL: func(ctx context.Context, url string) (int, []byte, error) {
			calls++
			return 200, []byte("<html></html>"), nil
		},
	}

	a := model.Asset{Kind: model.KindSubdomain, Key: "example.com", Host: "example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for non-endpoint asset kind, got %d", len(findings))
	}
	if calls != 0 {
		t.Fatalf("expected fetchURL not to be called for non-endpoint asset, got %d calls", calls)
	}
}

func TestJSDiffModuleMetadata(t *testing.T) {
	m := &JSDiffModule{}
	if m.Name() != "js-diff" {
		t.Fatalf("unexpected name: %s", m.Name())
	}
	if m.Category() != CategoryChange {
		t.Fatalf("unexpected category: %s", m.Category())
	}
	if m.Trigger() != TriggerNewAssetsOnly {
		t.Fatalf("unexpected trigger: %s", m.Trigger())
	}
}

func TestJSDiffRegisteredByDefault(t *testing.T) {
	found := false
	for _, m := range All() {
		if m.Name() == "js-diff" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected js-diff module to be registered via init()")
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
