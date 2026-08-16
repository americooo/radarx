package modules

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
)

func drainCloudBucketFindings(t *testing.T, m *CloudBucketModule, target model.Target, state State) []model.Finding {
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

// singleCandidateFetch only "answers" for one exact bucket URL (S3, base
// name with no suffix) and returns 404 (bucket does not exist) for every
// other candidate — keeping tests fast and their assertions unambiguous
// without needing to fake all ~20 candidate/provider requests.
func singleCandidateFetch(matchURL string, status int, body string) func(ctx context.Context, url string) (int, string, error) {
	return func(ctx context.Context, url string) (int, string, error) {
		if url == matchURL {
			return status, body, nil
		}
		return 404, "", nil
	}
}

func TestCloudBucketOpenS3ProducesHighSeverityFinding(t *testing.T) {
	m := &CloudBucketModule{
		fetch: singleCandidateFetch(
			"https://example.s3.amazonaws.com",
			200,
			`<?xml version="1.0"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></ListBucketResult>`,
		),
	}

	target := model.Target{Root: "example.com"}
	state := NewState(newFakeSettingsStore(), "test")

	findings := drainCloudBucketFindings(t, m, target, state)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Severity != model.SeverityHigh {
		t.Fatalf("expected High severity, got %s", f.Severity)
	}
	if f.Module != "cloud-bucket" {
		t.Fatalf("expected module cloud-bucket, got %s", f.Module)
	}
	if f.Asset.Kind != model.KindEndpoint {
		t.Fatalf("expected KindEndpoint asset, got %s", f.Asset.Kind)
	}
	if f.Asset.Key != "https://example.s3.amazonaws.com" {
		t.Fatalf("unexpected asset key: %s", f.Asset.Key)
	}
	if !strings.Contains(f.Title, "Open S3 bucket") {
		t.Fatalf("unexpected title: %s", f.Title)
	}
}

func TestCloudBucketExistsButClosedProducesInfoFinding(t *testing.T) {
	m := &CloudBucketModule{
		fetch: singleCandidateFetch(
			"https://example.s3.amazonaws.com",
			403,
			`<?xml version="1.0"?><Error><Code>AccessDenied</Code></Error>`,
		),
	}

	target := model.Target{Root: "example.com"}
	state := NewState(newFakeSettingsStore(), "test")

	findings := drainCloudBucketFindings(t, m, target, state)

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Severity != model.SeverityInfo {
		t.Fatalf("expected Info severity, got %s", f.Severity)
	}
	if !strings.Contains(f.Title, "bucket exists (access denied)") {
		t.Fatalf("unexpected title: %s", f.Title)
	}
}

func TestCloudBucketNotFoundProducesNoFinding(t *testing.T) {
	m := &CloudBucketModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			return 404, "", nil
		},
	}

	target := model.Target{Root: "example.com"}
	state := NewState(newFakeSettingsStore(), "test")

	findings := drainCloudBucketFindings(t, m, target, state)

	if len(findings) != 0 {
		t.Fatalf("expected no findings when bucket does not exist, got %d: %+v", len(findings), findings)
	}
}

func TestCloudBucketDedupesAcrossRuns(t *testing.T) {
	m := &CloudBucketModule{
		fetch: singleCandidateFetch(
			"https://example.s3.amazonaws.com",
			200,
			`<ListBucketResult></ListBucketResult>`,
		),
	}

	target := model.Target{Root: "example.com"}
	store := newFakeSettingsStore()
	state := NewState(store, "test")

	first := drainCloudBucketFindings(t, m, target, state)
	if len(first) != 1 {
		t.Fatalf("expected 1 finding on first run, got %d: %+v", len(first), first)
	}

	second := drainCloudBucketFindings(t, m, target, state)
	if len(second) != 0 {
		t.Fatalf("expected 0 findings on second run (already-seen bucket), got %d: %+v", len(second), second)
	}
}

func TestCloudBucketBaseNameDerivedFromRootDomain(t *testing.T) {
	cases := map[string]string{
		"example.com":     "example",
		"www.example.com": "example",
		"sub.example.co":  "sub",
		"":                "",
	}
	for root, want := range cases {
		if got := bucketBaseName(root); got != want {
			t.Fatalf("bucketBaseName(%q) = %q, want %q", root, got, want)
		}
	}
}

func TestCloudBucketGeneratesCandidatesFromBaseName(t *testing.T) {
	var mu sync.Mutex
	var requestedURLs []string
	m := &CloudBucketModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			mu.Lock()
			requestedURLs = append(requestedURLs, url)
			mu.Unlock()
			return 404, "", nil
		},
	}

	target := model.Target{Root: "example.com"}
	state := NewState(newFakeSettingsStore(), "test")

	drainCloudBucketFindings(t, m, target, state)

	wantSubstrings := []string{
		"example.s3.amazonaws.com",
		"example-backup.s3.amazonaws.com",
		"example-assets.s3.amazonaws.com",
		"storage.googleapis.com/example",
		"storage.googleapis.com/example-dev",
	}
	for _, want := range wantSubstrings {
		found := false
		for _, u := range requestedURLs {
			if strings.Contains(u, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected a request URL containing %q, requested: %v", want, requestedURLs)
		}
	}
}

func TestCloudBucketEmptyRootProducesNoRequests(t *testing.T) {
	calls := 0
	m := &CloudBucketModule{
		fetch: func(ctx context.Context, url string) (int, string, error) {
			calls++
			return 404, "", nil
		},
	}

	target := model.Target{Root: ""}
	state := NewState(newFakeSettingsStore(), "test")

	findings := drainCloudBucketFindings(t, m, target, state)

	if len(findings) != 0 {
		t.Fatalf("expected no findings for empty root, got %d", len(findings))
	}
	if calls != 0 {
		t.Fatalf("expected no fetch calls for empty root, got %d", calls)
	}
}

func TestCloudBucketModuleMetadata(t *testing.T) {
	m := &CloudBucketModule{}
	if m.Name() != "cloud-bucket" {
		t.Fatalf("unexpected name: %s", m.Name())
	}
	if m.Category() != CategoryDiscovery {
		t.Fatalf("unexpected category: %s", m.Category())
	}
	if m.Trigger() != TriggerScheduled {
		t.Fatalf("unexpected trigger: %s", m.Trigger())
	}
}

func TestCloudBucketRegisteredByDefault(t *testing.T) {
	found := false
	for _, m := range All() {
		if m.Name() == "cloud-bucket" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected cloud-bucket module to be registered via init()")
	}
}
