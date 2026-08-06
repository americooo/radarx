package engine

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/americooo/radarx/internal/model"
)

var titleRe = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

// resolveHosts filters a set of candidate hosts down to those that actually
// resolve in DNS, enriching each with its IPs. Used to verify CT-sourced hosts
// (which may include stale or never-deployed names) before trusting them.
func resolveHosts(ctx context.Context, candidates []model.Asset, workers int) []model.Asset {
	if workers <= 0 {
		workers = 40
	}
	resolver := &net.Resolver{}
	jobs := make(chan model.Asset)
	out := make(chan model.Asset)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for a := range jobs {
				lctx, cancel := context.WithTimeout(ctx, 4*time.Second)
				ips, err := resolver.LookupHost(lctx, a.Host)
				cancel()
				if err == nil && len(ips) > 0 {
					a.IP = dedupe(ips)
					out <- a
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, a := range candidates {
			select {
			case <-ctx.Done():
				return
			case jobs <- a:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(out)
	}()

	var live []model.Asset
	for a := range out {
		live = append(live, a)
	}
	return live
}

// probeClient is a hardened HTTP client for recon: no redirects followed
// automatically (we want to see the first response), short timeouts, and TLS
// verification disabled because recon frequently hits hosts with self-signed
// or mismatched certs — we record the cert separately, we don't trust it.
func probeClient() *http.Client {
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

// ProbeHTTP tries https then http for a host and returns an endpoint asset if
// the host answers. Returns (asset, true) on a response, (_, false) otherwise.
func ProbeHTTP(ctx context.Context, client *http.Client, host string) (model.Asset, bool) {
	for _, scheme := range []string{"https", "http"} {
		url := scheme + "://" + host
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "RadarX/0.1 (+asset-monitor)")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		resp.Body.Close()

		return model.Asset{
			Kind:       model.KindEndpoint,
			Key:        url,
			Host:       host,
			Scheme:     scheme,
			StatusCode: resp.StatusCode,
			Title:      extractTitle(body),
			Server:     resp.Header.Get("Server"),
		}, true
	}
	return model.Asset{}, false
}

func extractTitle(body []byte) string {
	m := titleRe.FindSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	t := strings.TrimSpace(string(m[1]))
	t = strings.Join(strings.Fields(t), " ") // collapse whitespace
	if len(t) > 120 {
		t = t[:120] + "…"
	}
	return t
}
