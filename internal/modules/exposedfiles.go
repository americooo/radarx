package modules

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// exposedFilePaths is intentionally short — this module is a "good citizen":
// a handful of well-known sensitive paths, not a directory brute-force list.
var exposedFilePaths = []string{
	"/.git/config",
	"/.git/HEAD",
	"/.env",
	"/.env.local",
	"/.env.production",
	"/config.php.bak",
	"/backup.zip",
	"/backup.sql",
	"/.DS_Store",
	"/wp-config.php.bak",
	"/.aws/credentials",
}

// envAssignmentRe matches an uppercase KEY=value line, the shape of a real
// .env file — distinguishes an actual dotenv from a soft-404 page that
// happens to return HTTP 200.
var envAssignmentRe = regexp.MustCompile(`(?m)^[A-Z_]+=.+$`)

// exposedFilesMaxConcurrency bounds how many of a single asset's candidate
// paths are checked at once — keeps this module a "good citizen" even though
// it's not fully sequential.
const exposedFilesMaxConcurrency = 4

// ExposedFilesModule probes an endpoint's base host for common sensitive
// files (.git, .env, backups, credentials) left exposed by misconfiguration.
// This is a read-only signal: it fetches the file, it never modifies or
// exploits anything.
//
// fetch is injectable so tests never touch real HTTP.
type ExposedFilesModule struct {
	fetch func(ctx context.Context, url string) (statusCode int, body string, err error)
}

func init() {
	Register(&ExposedFilesModule{
		fetch: defaultExposedFilesFetch,
	})
}

func (m *ExposedFilesModule) Name() string       { return "exposed-files" }
func (m *ExposedFilesModule) Category() Category { return CategoryVulnSignal }
func (m *ExposedFilesModule) Trigger() Trigger   { return TriggerNewAssetsOnly }

// Run checks one endpoint asset's base host (scheme+host, no path) for a
// short list of commonly exposed sensitive files.
func (m *ExposedFilesModule) Run(ctx context.Context, target model.Target, asset model.Asset, state State) (<-chan model.Finding, error) {
	out := make(chan model.Finding, 1)

	go func() {
		defer close(out)

		if asset.Kind != model.KindEndpoint {
			return
		}

		parsed, err := url.Parse(asset.Key)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return
		}
		base := parsed.Scheme + "://" + parsed.Host

		var wg sync.WaitGroup
		var mu sync.Mutex
		findings := make([]model.Finding, 0, len(exposedFilePaths))
		sem := make(chan struct{}, exposedFilesMaxConcurrency)

		for _, path := range exposedFilePaths {
			path := path
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					return
				}
				defer func() { <-sem }()

				fullURL := base + path
				status, body, err := m.fetch(ctx, fullURL)
				if err != nil {
					return
				}
				if !isExposedFileSignal(path, status, body) {
					return
				}

				finding := model.Finding{
					Module:   m.Name(),
					Severity: model.SeverityHigh,
					Asset: model.Asset{
						Kind: model.KindEndpoint,
						Key:  fullURL,
						Host: asset.Host,
					},
					Title:       fmt.Sprintf("Exposed file: %s", path),
					Description: fmt.Sprintf("%s responded with what looks like a real, sensitive file at %s — this should not be publicly reachable.", base, path),
					Evidence:    fmt.Sprintf("HTTP %d, %d bytes", status, len(body)),
					TakenAt:     time.Now().UTC(),
				}

				mu.Lock()
				findings = append(findings, finding)
				mu.Unlock()
			}()
		}

		wg.Wait()

		for _, f := range findings {
			select {
			case out <- f:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// isExposedFileSignal applies the soft-404 heuristics: a bare 200 with a
// small/empty body is not trusted, and .git/.env paths additionally require
// content matching their known real-file shape.
func isExposedFileSignal(path string, status int, body string) bool {
	if status != http.StatusOK {
		return false
	}
	if strings.TrimSpace(body) == "" || len(body) < 20 {
		return false
	}

	switch path {
	case "/.git/HEAD":
		return strings.Contains(body, "ref:")
	case "/.git/config":
		return strings.Contains(body, "[core]") || strings.Contains(body, "[remote")
	case "/.env", "/.env.local", "/.env.production":
		return envAssignmentRe.MatchString(body)
	default:
		return len(body) > 10
	}
}

// exposedFilesClient mirrors takeoverClient's pattern (see takeover.go):
// no automatic redirects, short timeout, TLS verification disabled since
// recon frequently hits hosts with self-signed or mismatched certs.
// internal/modules never imports internal/engine, so this is a small local
// copy rather than a shared dependency.
func exposedFilesClient() *http.Client {
	return &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // recon: cert is inspected, not trusted
			DisableKeepAlives: true,
			MaxIdleConns:      10,
		},
	}
}

func defaultExposedFilesFetch(ctx context.Context, url string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", "RadarX/0.1 (+asset-monitor)")

	client := exposedFilesClient()
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(body), nil
}
