package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// crtshEntry is one row from crt.sh's JSON output.
type crtshEntry struct {
	NameValue string `json:"name_value"`
}

// EnumerateCertTransparency queries crt.sh for certificates issued to the root
// domain and extracts subdomains from them. This is passive (it never touches
// the target) and typically finds far more hosts than DNS brute-forcing —
// including internal-looking names that were briefly exposed in a cert.
//
// Returned hosts are NOT yet verified to resolve; the caller enriches them.
func EnumerateCertTransparency(ctx context.Context, root string) ([]model.Asset, error) {
	url := fmt.Sprintf("https://crt.sh/?q=%%25.%s&output=json", root)

	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RadarX/0.1 (+asset-monitor)")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crt.sh returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, err
	}

	var entries []crtshEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parse crt.sh json: %w", err)
	}

	seen := make(map[string]struct{})
	var assets []model.Asset
	suffix := "." + root
	for _, e := range entries {
		// name_value may contain multiple newline-separated names.
		for _, name := range strings.Split(e.NameValue, "\n") {
			host := strings.ToLower(strings.TrimSpace(name))
			host = strings.TrimPrefix(host, "*.") // ignore wildcard prefix
			if host == "" {
				continue
			}
			if host != root && !strings.HasSuffix(host, suffix) {
				continue // stay strictly in scope
			}
			if _, ok := seen[host]; ok {
				continue
			}
			seen[host] = struct{}{}
			assets = append(assets, model.Asset{
				Kind: model.KindSubdomain,
				Key:  host,
				Host: host,
			})
		}
	}
	return assets, nil
}

// mergeSubdomainSources combines DNS-brute and CT results, de-duplicating by
// host. Kept here so the scanner can call one function.
func mergeSubdomains(sets ...[]model.Asset) []model.Asset {
	seen := make(map[string]struct{})
	var out []model.Asset
	for _, set := range sets {
		for _, a := range set {
			if _, ok := seen[a.Host]; ok {
				continue
			}
			seen[a.Host] = struct{}{}
			out = append(out, a)
		}
	}
	return out
}

// ctTimeout is exposed for callers that want to bound CT lookups independently.
var ctTimeout = 20 * time.Second
