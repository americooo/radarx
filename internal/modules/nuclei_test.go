package modules

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/americooo/radarx/internal/model"
)

func TestNucleiSkipsWhenBinaryNotFound(t *testing.T) {
	runCalls := 0
	m := &NucleiModule{
		lookPath: func(file string) (string, error) {
			return "", errors.New("executable file not found in $PATH")
		},
		runNuclei: func(ctx context.Context, targetURL string) (io.Reader, error) {
			runCalls++
			return strings.NewReader(""), nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://sub.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings when nuclei is missing, got %d", len(findings))
	}
	if runCalls != 0 {
		t.Fatalf("expected runNuclei not to be called when nuclei binary is missing, got %d calls", runCalls)
	}
}

func TestNucleiParsesJSONLFindings(t *testing.T) {
	jsonl := `{"template-id":"exposed-panel","info":{"name":"Exposed Admin Panel","severity":"high","description":"An admin panel is exposed."},"matched-at":"https://sub.example.com/admin"}
{"template-id":"tech-detect","info":{"name":"Tech Fingerprint","severity":"info","description":"Detected technology."},"matched-at":"https://sub.example.com"}
{"template-id":"cve-2023-xxxx","info":{"name":"Critical RCE","severity":"critical","description":"Remote code execution."},"matched-at":"https://sub.example.com/api"}
`

	m := &NucleiModule{
		lookPath: func(file string) (string, error) { return "/usr/bin/nuclei", nil },
		runNuclei: func(ctx context.Context, targetURL string) (io.Reader, error) {
			return strings.NewReader(jsonl), nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://sub.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 3 {
		t.Fatalf("expected 3 findings, got %d: %+v", len(findings), findings)
	}

	wantSeverities := []model.Severity{model.SeverityHigh, model.SeverityInfo, model.SeverityCritical}
	for i, f := range findings {
		if f.Module != "nuclei" {
			t.Fatalf("expected module nuclei, got %s", f.Module)
		}
		if f.Severity != wantSeverities[i] {
			t.Fatalf("finding %d: expected severity %s, got %s", i, wantSeverities[i], f.Severity)
		}
		if f.Asset.Key != a.Key {
			t.Fatalf("finding %d: expected asset key %s, got %s", i, a.Key, f.Asset.Key)
		}
	}

	if findings[0].Title != "Exposed Admin Panel" {
		t.Fatalf("expected title 'Exposed Admin Panel', got %s", findings[0].Title)
	}
	if findings[0].Evidence != "https://sub.example.com/admin" {
		t.Fatalf("expected evidence to be matched-at, got %s", findings[0].Evidence)
	}
}

func TestNucleiEmptyJSONLProducesNoFindings(t *testing.T) {
	m := &NucleiModule{
		lookPath: func(file string) (string, error) { return "/usr/bin/nuclei", nil },
		runNuclei: func(ctx context.Context, targetURL string) (io.Reader, error) {
			return strings.NewReader(""), nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://sub.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for empty nuclei output, got %d", len(findings))
	}
}

func TestNucleiSkipsNonEndpointAssets(t *testing.T) {
	runCalls := 0
	m := &NucleiModule{
		lookPath: func(file string) (string, error) { return "/usr/bin/nuclei", nil },
		runNuclei: func(ctx context.Context, targetURL string) (io.Reader, error) {
			runCalls++
			return strings.NewReader(""), nil
		},
	}

	a := model.Asset{Kind: model.KindSubdomain, Key: "sub.example.com", Host: "sub.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for non-endpoint asset kind, got %d", len(findings))
	}
	if runCalls != 0 {
		t.Fatalf("expected runNuclei not to be called for non-endpoint asset, got %d calls", runCalls)
	}
}

func TestNucleiModuleMetadata(t *testing.T) {
	m := &NucleiModule{}
	if m.Name() != "nuclei" {
		t.Fatalf("unexpected name: %s", m.Name())
	}
	if m.Category() != CategoryVulnSignal {
		t.Fatalf("unexpected category: %s", m.Category())
	}
	if m.Trigger() != TriggerNewAssetsOnly {
		t.Fatalf("unexpected trigger: %s", m.Trigger())
	}
}

func TestNucleiRegisteredByDefault(t *testing.T) {
	found := false
	for _, m := range All() {
		if m.Name() == "nuclei" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected nuclei module to be registered via init()")
	}
}

func TestMapSeverityUnknownDefaultsToInfo(t *testing.T) {
	if got := mapSeverity("unknown"); got != model.SeverityInfo {
		t.Fatalf("expected unknown severity to map to info, got %s", got)
	}
	if got := mapSeverity(""); got != model.SeverityInfo {
		t.Fatalf("expected empty severity to map to info, got %s", got)
	}
}
