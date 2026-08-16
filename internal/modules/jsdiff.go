// Package modules: jsdiff.go watches the JS files referenced by an HTML
// endpoint for content changes — a signal no one-shot recon tool produces,
// since it only means something once you have a "before" to diff against.
package modules

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// maxHTMLBodyBytes bounds how much of the HTML page we read looking for
// <script src> tags — pages don't need to be huge to enumerate their scripts.
const maxHTMLBodyBytes = 512 * 1024

// maxJSBodyBytes bounds how much of each JS file we read to hash — enough
// for most bundles without risking unbounded memory on a pathological file.
const maxJSBodyBytes = 1024 * 1024

// maxJSFilesPerPage caps how many <script src> URLs we follow per page, so a
// page with dozens of scripts doesn't turn one asset into dozens of requests.
const maxJSFilesPerPage = 15

var (
	scriptSrcRe = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+)["']`)
	secretishRe = regexp.MustCompile(`(?i)(api[_-]?key|secret|token)["']?\s*[:=]\s*["']([a-zA-Z0-9_\-]{16,})["']`)
)

// JSDiffModule fetches an HTML endpoint, follows its <script src> links, and
// hashes each JS file's content — the first time a JS URL is seen it's just
// recorded as a baseline, every time after that a content hash change
// becomes a Finding.
//
// fetchURL is injectable so tests never touch the network.
type JSDiffModule struct {
	fetchURL func(ctx context.Context, url string) (statusCode int, body []byte, err error)
}

func init() {
	Register(&JSDiffModule{
		fetchURL: defaultJSDiffFetch,
	})
}

func (m *JSDiffModule) Name() string       { return "js-diff" }
func (m *JSDiffModule) Category() Category { return CategoryChange }
func (m *JSDiffModule) Trigger() Trigger   { return TriggerNewAssetsOnly }

// Run only operates on KindEndpoint assets (HTML pages) — it fetches the
// page, resolves every <script src> to an absolute URL, hashes each script's
// content, and emits a Finding when a previously-seen script's hash changes.
func (m *JSDiffModule) Run(ctx context.Context, target model.Target, asset model.Asset, state State) (<-chan model.Finding, error) {
	out := make(chan model.Finding, 1)

	go func() {
		defer close(out)

		if asset.Kind != model.KindEndpoint || asset.Key == "" {
			return
		}

		baseURL, err := url.Parse(asset.Key)
		if err != nil {
			return
		}

		_, htmlBody, err := m.fetchURL(ctx, asset.Key)
		if err != nil || len(htmlBody) == 0 {
			return
		}

		jsURLs := extractScriptURLs(baseURL, htmlBody)
		if len(jsURLs) > maxJSFilesPerPage {
			jsURLs = jsURLs[:maxJSFilesPerPage]
		}

		for _, jsURL := range jsURLs {
			select {
			case <-ctx.Done():
				return
			default:
			}

			_, jsBody, err := m.fetchURL(ctx, jsURL)
			if err != nil || len(jsBody) == 0 {
				continue
			}

			newHash := sha256Hex(jsBody)
			stateKey := "hash:" + jsURL

			oldHash, ok, err := state.Get(stateKey)
			if err != nil {
				continue
			}

			if !ok {
				// First time this JS URL is seen — record the baseline, no
				// Finding (avoids spamming on the very first scan).
				_ = state.Set(stateKey, newHash)
			} else if oldHash != newHash {
				finding := model.Finding{
					Module:      m.Name(),
					Severity:    model.SeverityLow,
					Asset:       model.Asset{Kind: model.KindEndpoint, Key: jsURL, Host: asset.Host},
					Title:       "JS file changed",
					Description: fmt.Sprintf("%s content changed since last scan", jsURL),
					Evidence:    fmt.Sprintf("old sha256=%s new sha256=%s", oldHash, newHash),
					TakenAt:     time.Now().UTC(),
				}
				select {
				case out <- finding:
				case <-ctx.Done():
					return
				}
				_ = state.Set(stateKey, newHash)
			}

			if secret, ok := findSecretPattern(jsBody); ok {
				finding := model.Finding{
					Module:      m.Name(),
					Severity:    model.SeverityMedium,
					Asset:       model.Asset{Kind: model.KindEndpoint, Key: jsURL, Host: asset.Host},
					Title:       "Potential secret pattern in JS",
					Description: fmt.Sprintf("%s contains a string matching a common secret/API key pattern", jsURL),
					Evidence:    secret,
					TakenAt:     time.Now().UTC(),
				}
				select {
				case out <- finding:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

// extractScriptURLs finds every <script src="..."> in body and resolves it
// to an absolute URL relative to base.
func extractScriptURLs(base *url.URL, body []byte) []string {
	matches := scriptSrcRe.FindAllSubmatch(body, -1)
	urls := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		raw := strings.TrimSpace(string(match[1]))
		if raw == "" {
			continue
		}
		ref, err := url.Parse(raw)
		if err != nil {
			continue
		}
		urls = append(urls, base.ResolveReference(ref).String())
	}
	return urls
}

// findSecretPattern looks for an obvious "key/secret/token = value" pattern
// in JS content — a low-effort heuristic, not a real secret scanner, so it's
// kept intentionally simple.
func findSecretPattern(body []byte) (evidence string, found bool) {
	m := secretishRe.FindSubmatch(body)
	if m == nil {
		return "", false
	}
	return string(m[0]), true
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// jsDiffClient mirrors internal/engine's httpprobe.go probeClient pattern
// (and takeover.go's takeoverClient): no automatic redirects, short timeout,
// TLS verification disabled since recon frequently hits hosts with
// self-signed or mismatched certs. internal/modules never imports
// internal/engine, so this is a small local copy of the same pattern rather
// than a shared dependency.
func jsDiffClient() *http.Client {
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

func defaultJSDiffFetch(ctx context.Context, target string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("User-Agent", "RadarX/0.1 (+asset-monitor)")

	client := jsDiffClient()
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	limit := maxHTMLBodyBytes
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "javascript") {
		limit = maxJSBodyBytes
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}
