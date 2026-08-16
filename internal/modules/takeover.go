package modules

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// fingerprint describes one dangling-CNAME-prone service: a CNAME pointing
// there is a takeover candidate if the response body also matches the
// service's "nothing here" signature.
type fingerprint struct {
	service       string
	cnameContains []string
	bodySignature string
}

// takeoverFingerprints is not exhaustive — it covers commonly abused
// services. Extend it as new takeover-prone providers show up.
var takeoverFingerprints = []fingerprint{
	{"Amazon S3", []string{".s3.amazonaws.com", ".s3-website"}, "NoSuchBucket"},
	{"GitHub Pages", []string{".github.io"}, "There isn't a GitHub Pages site here"},
	{"Heroku", []string{".herokudns.com", ".herokuapp.com"}, "no such app"},
	{"Azure Web Apps", []string{".azurewebsites.net"}, "404 Web Site not found"},
	{"Fastly", []string{".fastly.net"}, "Fastly error: unknown domain"},
	{"Shopify", []string{".myshopify.com"}, "Sorry, this shop is currently unavailable"},
	{"Zendesk", []string{".zendesk.com"}, "Help Center Closed"},
	{"Tumblr", []string{".tumblr.com"}, "There's nothing here"},
	{"WordPress.com", []string{".wordpress.com"}, "Do you want to register"},
	{"Surge.sh", []string{".surge.sh"}, "project not found"},
	{"Pantheon", []string{".pantheonsite.io"}, "The gods are wise"},
	{"Netlify", []string{".netlify.app"}, "Not Found - Request ID"},
}

// TakeoverModule detects dangling CNAMEs pointing at services known to be
// abusable for subdomain takeover: this is a read-only signal, it never
// attempts to actually claim the dangling resource.
//
// lookupCNAME and fetch are injectable so tests never touch real DNS/HTTP.
type TakeoverModule struct {
	lookupCNAME func(ctx context.Context, host string) (string, error)
	fetch       func(ctx context.Context, url string) (statusCode int, body string, err error)
}

func init() {
	Register(&TakeoverModule{
		lookupCNAME: defaultLookupCNAME,
		fetch:       defaultFetch,
	})
}

func (m *TakeoverModule) Name() string       { return "subdomain-takeover" }
func (m *TakeoverModule) Category() Category { return CategoryDiscovery }
func (m *TakeoverModule) Trigger() Trigger   { return TriggerNewAssetsOnly }

// Run checks one asset's CNAME against the fingerprint table and, on a
// match, fetches the host over HTTPS then HTTP looking for the service's
// "nothing here" signature in the body.
func (m *TakeoverModule) Run(ctx context.Context, asset model.Asset) (<-chan model.Finding, error) {
	out := make(chan model.Finding, 1)

	go func() {
		defer close(out)

		if asset.Kind != model.KindSubdomain || asset.Host == "" {
			return
		}

		cname, err := m.lookupCNAME(ctx, asset.Host)
		if err != nil || cname == "" {
			return
		}

		fp, matched := matchFingerprint(cname)
		if !matched {
			return
		}

		for _, scheme := range []string{"https", "http"} {
			url := scheme + "://" + asset.Host
			_, body, err := m.fetch(ctx, url)
			if err != nil {
				continue
			}
			if strings.Contains(body, fp.bodySignature) {
				snippet := body
				if len(snippet) > 200 {
					snippet = snippet[:200] + "…"
				}
				finding := model.Finding{
					Module:      m.Name(),
					Severity:    model.SeverityHigh,
					Asset:       asset,
					Title:       fmt.Sprintf("Possible %s subdomain takeover", fp.service),
					Description: fmt.Sprintf("%s has a CNAME (%s) pointing at %s, and the response body matches that service's unclaimed-resource signature.", asset.Host, cname, fp.service),
					Evidence:    fmt.Sprintf("CNAME=%s body=%q", cname, snippet),
					TakenAt:     time.Now().UTC(),
				}
				select {
				case out <- finding:
				case <-ctx.Done():
				}
				return
			}
		}
	}()

	return out, nil
}

func matchFingerprint(cname string) (fingerprint, bool) {
	lc := strings.ToLower(cname)
	for _, fp := range takeoverFingerprints {
		for _, needle := range fp.cnameContains {
			if strings.Contains(lc, needle) {
				return fp, true
			}
		}
	}
	return fingerprint{}, false
}

func defaultLookupCNAME(ctx context.Context, host string) (string, error) {
	resolver := &net.Resolver{}
	cname, err := resolver.LookupCNAME(ctx, host)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(cname, "."), nil
}

// takeoverClient mirrors internal/engine's httpprobe.go client pattern: no
// automatic redirects (we want to see the first response), short timeout,
// and TLS verification disabled since recon frequently hits hosts with
// self-signed or mismatched certs. internal/modules never imports
// internal/engine, so this is a small local copy of the same pattern
// rather than a shared dependency.
func takeoverClient() *http.Client {
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

func defaultFetch(ctx context.Context, url string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", "RadarX/0.1 (+asset-monitor)")

	client := takeoverClient()
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
