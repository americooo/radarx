package modules

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// bucketSuffixes are appended to the target's base name to build candidate
// bucket names. Kept short (~10) so a scheduled run stays cheap and polite —
// this is discovery-by-guessing, not brute force.
var bucketSuffixes = []string{
	"",
	"-backup",
	"-assets",
	"-static",
	"-dev",
	"-prod",
	"-data",
	"-files",
	"-media",
	"-uploads",
}

// bucketMaxConcurrency bounds how many candidate/provider checks run at
// once, mirroring internal/engine's "sem := make(chan struct{}, N)" good
// citizen pattern.
const bucketMaxConcurrency = 5

// cloudBucketProvider describes one cloud storage provider's public bucket
// URL scheme and how to tell "exists+open", "exists+closed" and "doesn't
// exist" apart from a plain GET response.
type cloudBucketProvider struct {
	name string
	url  func(bucket string) string
}

var cloudBucketProviders = []cloudBucketProvider{
	{
		name: "S3",
		url:  func(bucket string) string { return fmt.Sprintf("https://%s.s3.amazonaws.com", bucket) },
	},
	{
		name: "GCS",
		url:  func(bucket string) string { return fmt.Sprintf("https://storage.googleapis.com/%s", bucket) },
	},
}

// CloudBucketModule generates candidate cloud storage bucket names from a
// target's root domain and checks whether S3/GCS buckets by those names
// exist and, if so, whether they allow public listing. It never writes,
// deletes or otherwise touches bucket contents — a GET against the bucket
// root is the only request made.
//
// fetch is injectable so tests never touch real cloud infrastructure.
type CloudBucketModule struct {
	fetch func(ctx context.Context, url string) (statusCode int, body string, err error)
}

func init() {
	Register(&CloudBucketModule{
		fetch: defaultCloudBucketFetch,
	})
}

func (m *CloudBucketModule) Name() string       { return "cloud-bucket" }
func (m *CloudBucketModule) Category() Category { return CategoryDiscovery }
func (m *CloudBucketModule) Trigger() Trigger   { return TriggerScheduled }

// bucketCheck is one candidate/provider pairing's outcome.
type bucketCheck struct {
	provider string
	bucket   string
	exists   bool
	open     bool
}

// Run derives a base name from target.Root, generates candidate bucket
// names, and probes each against every provider in cloudBucketProviders.
// asset is ignored (zero-value) — TriggerScheduled modules operate on the
// whole target, not one asset.
func (m *CloudBucketModule) Run(ctx context.Context, target model.Target, asset model.Asset, state State) (<-chan model.Finding, error) {
	out := make(chan model.Finding, 1)

	go func() {
		defer close(out)

		base := bucketBaseName(target.Root)
		if base == "" {
			return
		}

		candidates := make([]string, 0, len(bucketSuffixes))
		for _, suffix := range bucketSuffixes {
			candidates = append(candidates, base+suffix)
		}

		checks := m.probeAll(ctx, candidates)

		for _, c := range checks {
			if !c.exists {
				continue
			}

			seenKey := "seen:" + c.provider + ":" + c.bucket
			if _, ok, err := state.Get(seenKey); err == nil && ok {
				continue
			}

			bucketURL := providerURL(c.provider, c.bucket)

			var finding model.Finding
			if c.open {
				finding = model.Finding{
					Module:      m.Name(),
					Severity:    model.SeverityHigh,
					Asset:       model.Asset{Kind: model.KindEndpoint, Key: bucketURL, Host: target.Root},
					Title:       fmt.Sprintf("Open %s bucket: %s", c.provider, c.bucket),
					Description: fmt.Sprintf("%s bucket %q exists and allows public listing of its contents.", c.provider, c.bucket),
					Evidence:    "HTTP 200, listing enabled",
					TakenAt:     time.Now().UTC(),
				}
			} else {
				finding = model.Finding{
					Module:      m.Name(),
					Severity:    model.SeverityInfo,
					Asset:       model.Asset{Kind: model.KindEndpoint, Key: bucketURL, Host: target.Root},
					Title:       fmt.Sprintf("%s bucket exists (access denied): %s", c.provider, c.bucket),
					Description: fmt.Sprintf("%s bucket %q exists but public listing is denied.", c.provider, c.bucket),
					Evidence:    "HTTP 403",
					TakenAt:     time.Now().UTC(),
				}
			}

			select {
			case out <- finding:
			case <-ctx.Done():
				return
			}

			if err := state.Set(seenKey, "1"); err != nil {
				continue
			}
		}
	}()

	return out, nil
}

// probeAll checks every candidate name against every provider, bounded to
// bucketMaxConcurrency concurrent requests.
func (m *CloudBucketModule) probeAll(ctx context.Context, candidates []string) []bucketCheck {
	sem := make(chan struct{}, bucketMaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var results []bucketCheck

	for _, candidate := range candidates {
		for _, provider := range cloudBucketProviders {
			candidate, provider := candidate, provider
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					return
				}
				defer func() { <-sem }()

				exists, open := m.probeOne(ctx, provider, candidate)
				if !exists {
					return
				}

				mu.Lock()
				results = append(results, bucketCheck{
					provider: provider.name,
					bucket:   candidate,
					exists:   exists,
					open:     open,
				})
				mu.Unlock()
			}()
		}
	}

	wg.Wait()
	return results
}

// probeOne fetches one candidate bucket's URL for one provider and
// classifies the response: (exists=false) means "no such bucket" (404),
// (exists=true, open=true) means public listing is possible, and
// (exists=true, open=false) means the bucket exists but access is denied.
func (m *CloudBucketModule) probeOne(ctx context.Context, provider cloudBucketProvider, bucket string) (exists, open bool) {
	status, body, err := m.fetch(ctx, provider.url(bucket))
	if err != nil {
		return false, false
	}

	switch status {
	case http.StatusNotFound:
		return false, false
	case http.StatusOK:
		if strings.Contains(body, "<ListBucketResult") || strings.Contains(body, `"items"`) {
			return true, true
		}
		return true, false
	case http.StatusForbidden:
		return true, false
	default:
		return false, false
	}
}

// providerURL returns the public URL for a bucket under the named provider,
// matching the scheme used to probe it.
func providerURL(provider, bucket string) string {
	for _, p := range cloudBucketProviders {
		if p.name == provider {
			return p.url(bucket)
		}
	}
	return bucket
}

// bucketBaseName derives a plausible "company name" from a root domain:
// example.com -> example, www.example.co.uk -> example. It simply takes the
// first label of the domain, which covers the common case without needing a
// public-suffix list.
func bucketBaseName(root string) string {
	root = strings.ToLower(strings.TrimSpace(root))
	root = strings.TrimPrefix(root, "www.")
	if root == "" {
		return ""
	}

	labels := strings.Split(root, ".")
	base := labels[0]
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	return base
}

// cloudBucketClient mirrors takeoverClient's pattern: no automatic
// redirects, short timeout, TLS verification disabled since recon
// frequently hits hosts with self-signed or mismatched certs.
// internal/modules never imports internal/engine, so this is a small local
// copy of the same pattern rather than a shared dependency.
func cloudBucketClient() *http.Client {
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

func defaultCloudBucketFetch(ctx context.Context, url string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("User-Agent", "RadarX/0.1 (+asset-monitor)")

	client := cloudBucketClient()
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
