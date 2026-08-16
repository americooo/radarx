// Package modules: rate-limit signal. This is NOT an attack module — it
// never brute-forces or bombards a target. It sends exactly two ordinary GET
// requests to an endpoint that *looks* sensitive (login/auth/password/admin
// in its path) and checks whether the response carries any sign of
// rate-limiting or bot protection (Retry-After, X-RateLimit-*/RateLimit-*
// headers, HTTP 429, or a CAPTCHA/rate-limit hint in the body). If neither
// response shows any such sign, that absence is itself the signal: "this
// sensitive endpoint doesn't appear to be protected against repeated
// requests." The module deliberately caps itself at two requests per asset
// and never retries or escalates.
package modules

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// sensitivePathKeywords marks an endpoint as "looks sensitive" — worth the
// two-request check — without touching every endpoint RadarX has ever seen
// (that would be noisy and closer to blanket probing than a targeted
// signal).
var sensitivePathKeywords = []string{
	"login", "signin", "sign-in", "auth", "password", "reset", "admin",
}

// RateLimitModule flags endpoints that look sensitive (login/auth/password/
// reset/admin in the path) but show no observable rate-limit or CAPTCHA
// protection across two plain GET requests. fetch is injectable so tests
// never touch the network, and its call count is the enforcement mechanism
// for "never more than two requests."
type RateLimitModule struct {
	fetch func(ctx context.Context, url string) (statusCode int, headers http.Header, body string, err error)
}

func init() {
	Register(&RateLimitModule{
		fetch: defaultRateLimitFetch,
	})
}

func (m *RateLimitModule) Name() string       { return "rate-limit-signal" }
func (m *RateLimitModule) Category() Category { return CategoryVulnSignal }
func (m *RateLimitModule) Trigger() Trigger   { return TriggerNewAssetsOnly }

// Run checks one endpoint asset. It only acts on endpoints whose path looks
// sensitive, sends exactly two GET requests, and emits a Medium finding only
// if neither response shows any rate-limit or CAPTCHA signal. If a
// protection signal is present, or the asset was already flagged before,
// Run emits nothing.
func (m *RateLimitModule) Run(ctx context.Context, target model.Target, asset model.Asset, state State) (<-chan model.Finding, error) {
	out := make(chan model.Finding, 1)

	go func() {
		defer close(out)

		if asset.Kind != model.KindEndpoint || asset.Key == "" {
			return
		}
		if !looksSensitive(asset.Key) {
			return
		}

		stateKey := "checked:" + asset.Key
		if prev, ok, err := state.Get(stateKey); err == nil && ok && prev == "unprotected" {
			// Already reported once for this URL — don't spam repeat findings.
			return
		}

		protected := false
		for i := 0; i < 2; i++ {
			status, headers, body, err := m.fetch(ctx, asset.Key)
			if err != nil {
				// A failed request tells us nothing about protection either
				// way; skip it rather than treat it as a signal.
				continue
			}
			if hasProtectionSignal(status, headers, body) {
				protected = true
			}
		}

		if protected {
			_ = state.Set(stateKey, "protected")
			return
		}

		_ = state.Set(stateKey, "unprotected")

		finding := model.Finding{
			Module:      m.Name(),
			Severity:    model.SeverityMedium,
			Asset:       asset,
			Title:       "No rate-limit protection observed",
			Description: fmt.Sprintf("%s responded to repeated requests with no Retry-After/X-RateLimit headers, 429 status, or CAPTCHA indicator", asset.Key),
			Evidence:    "2 test requests, no protection signal",
			TakenAt:     time.Now().UTC(),
		}
		select {
		case out <- finding:
		case <-ctx.Done():
		}
	}()

	return out, nil
}

// looksSensitive reports whether url's path contains one of
// sensitivePathKeywords, case-insensitively.
func looksSensitive(url string) bool {
	lc := strings.ToLower(url)
	for _, kw := range sensitivePathKeywords {
		if strings.Contains(lc, kw) {
			return true
		}
	}
	return false
}

// hasProtectionSignal reports whether a single response shows any evidence
// of rate-limiting or bot protection.
func hasProtectionSignal(status int, headers http.Header, body string) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	if headers.Get("Retry-After") != "" {
		return true
	}
	for h := range headers {
		lh := strings.ToLower(h)
		if strings.HasPrefix(lh, "x-ratelimit-") || strings.HasPrefix(lh, "ratelimit-") {
			return true
		}
	}
	lb := strings.ToLower(body)
	for _, needle := range []string{"captcha", "rate limit", "too many"} {
		if strings.Contains(lb, needle) {
			return true
		}
	}
	return false
}

// rateLimitClient mirrors takeoverClient (internal/modules never imports
// internal/engine, so each module keeps its own small local HTTP client
// rather than sharing one).
func rateLimitClient() *http.Client {
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

func defaultRateLimitFetch(ctx context.Context, url string) (int, http.Header, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, "", err
	}
	req.Header.Set("User-Agent", "RadarX/0.1 (+asset-monitor)")

	client := rateLimitClient()
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return resp.StatusCode, resp.Header, "", err
	}
	return resp.StatusCode, resp.Header, string(body), nil
}
