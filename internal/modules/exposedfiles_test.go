package modules

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/americooo/radarx/internal/model"
)

func TestExposedFilesGitHEADFound(t *testing.T) {
	m := &ExposedFilesModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			if strings.HasSuffix(url, "/.git/HEAD") {
				return 200, "ref: refs/heads/main\n", nil
			}
			return 404, "not found", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Module != "exposed-files" {
		t.Fatalf("expected module name exposed-files, got %s", f.Module)
	}
	if f.Severity != model.SeverityHigh {
		t.Fatalf("expected High severity, got %s", f.Severity)
	}
	if f.Asset.Key != "https://assets.example.com/.git/HEAD" {
		t.Fatalf("unexpected finding asset key: %s", f.Asset.Key)
	}
	if f.Asset.Host != a.Host {
		t.Fatalf("expected finding asset host to match input asset, got %+v", f.Asset)
	}
}

func TestExposedFilesEnvFound(t *testing.T) {
	m := &ExposedFilesModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			if strings.HasSuffix(url, "/.env") {
				return 200, "DB_PASSWORD=secret123\n", nil
			}
			return 404, "not found", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Asset.Key != "https://assets.example.com/.env" {
		t.Fatalf("unexpected finding asset key: %s", f.Asset.Key)
	}
	if !strings.Contains(f.Title, "/.env") {
		t.Fatalf("expected title to mention /.env, got %q", f.Title)
	}
}

func TestExposedFiles404NoFinding(t *testing.T) {
	m := &ExposedFilesModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			return 404, "Not Found", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for 404 responses, got %d: %+v", len(findings), findings)
	}
}

func TestExposedFilesSoft404NoFinding(t *testing.T) {
	m := &ExposedFilesModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			if strings.HasSuffix(url, "/.git/HEAD") {
				// Soft-404: server returns 200 for everything, including
				// missing paths, so the body has no "ref:" — not a real HEAD.
				return 200, "<html><body>Page not found, sorry!</body></html>", nil
			}
			return 404, "not found", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for soft-404 responses, got %d: %+v", len(findings), findings)
	}
}

func TestExposedFilesEmptyBodyNoFinding(t *testing.T) {
	m := &ExposedFilesModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			return 200, "", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for empty body, got %d: %+v", len(findings), findings)
	}
}

func TestExposedFilesBackupFileFound(t *testing.T) {
	m := &ExposedFilesModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			if strings.HasSuffix(url, "/backup.sql") {
				return 200, "-- MySQL dump 10.13\nCREATE TABLE users (...);\n", nil
			}
			return 404, "not found", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	if findings[0].Asset.Key != "https://assets.example.com/backup.sql" {
		t.Fatalf("unexpected finding asset key: %s", findings[0].Asset.Key)
	}
}

func TestExposedFilesSkipsNonEndpointAssets(t *testing.T) {
	calls := 0
	m := &ExposedFilesModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			calls++
			return 200, "ref: refs/heads/main\n", nil
		},
	}

	a := model.Asset{Kind: model.KindSubdomain, Key: "assets.example.com", Host: "assets.example.com"}
	findings := drainFindings(t, m, a)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for non-endpoint asset kind, got %d", len(findings))
	}
	if calls != 0 {
		t.Fatalf("expected fetch not to be called for non-endpoint asset, got %d calls", calls)
	}
}

func TestExposedFilesChecksExpectedPathsOnly(t *testing.T) {
	seen := make(map[string]bool)
	var mu sync.Mutex
	m := &ExposedFilesModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			mu.Lock()
			seen[url] = true
			mu.Unlock()
			return 404, "not found", nil
		},
	}

	a := model.Asset{Kind: model.KindEndpoint, Key: "https://assets.example.com", Host: "assets.example.com"}
	drainFindings(t, m, a)

	if len(seen) != len(exposedFilePaths) {
		t.Fatalf("expected exactly %d requests, got %d: %v", len(exposedFilePaths), len(seen), seen)
	}
	for _, path := range exposedFilePaths {
		if !seen["https://assets.example.com"+path] {
			t.Fatalf("expected a request for %s, got %v", path, seen)
		}
	}
}

func TestExposedFilesModuleMetadata(t *testing.T) {
	m := &ExposedFilesModule{}
	if m.Name() != "exposed-files" {
		t.Fatalf("unexpected name: %s", m.Name())
	}
	if m.Category() != CategoryVulnSignal {
		t.Fatalf("unexpected category: %s", m.Category())
	}
	if m.Trigger() != TriggerNewAssetsOnly {
		t.Fatalf("unexpected trigger: %s", m.Trigger())
	}
}

func TestExposedFilesRegisteredByDefault(t *testing.T) {
	found := false
	for _, m := range All() {
		if m.Name() == "exposed-files" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected exposed-files module to be registered via init()")
	}
}
