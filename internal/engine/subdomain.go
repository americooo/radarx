package engine

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// DefaultWordlist is a small built-in list of common subdomain labels used when
// a target doesn't supply its own. Kept intentionally short; production users
// point RadarX at a real wordlist.
var DefaultWordlist = []string{
	"www", "api", "dev", "staging", "stage", "test", "admin", "portal",
	"app", "apps", "mail", "smtp", "webmail", "vpn", "remote", "git",
	"gitlab", "jenkins", "ci", "cdn", "static", "assets", "img", "media",
	"docs", "help", "support", "status", "dashboard", "internal", "beta",
	"demo", "shop", "store", "pay", "billing", "auth", "sso", "login",
	"blog", "news", "m", "mobile", "ftp", "db", "database", "grafana",
	"kibana", "prometheus", "monitor", "metrics", "s3", "storage", "backup",
}

// resolveResult carries a single label's resolution outcome.
type resolveResult struct {
	host string
	ips  []string
}

// EnumerateSubdomains brute-forces subdomains of root using DNS resolution.
// It is safe (read-only DNS lookups) and concurrency-bounded. Only hosts that
// actually resolve are returned as assets.
func EnumerateSubdomains(ctx context.Context, root string, wordlist []string, workers int) []model.Asset {
	if len(wordlist) == 0 {
		wordlist = DefaultWordlist
	}
	if workers <= 0 {
		workers = 40
	}

	resolver := &net.Resolver{}
	jobs := make(chan string)
	out := make(chan resolveResult)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for label := range jobs {
				host := label + "." + root
				lctx, cancel := context.WithTimeout(ctx, 4*time.Second)
				ips, err := resolver.LookupHost(lctx, host)
				cancel()
				if err == nil && len(ips) > 0 {
					out <- resolveResult{host: host, ips: dedupe(ips)}
				}
			}
		}()
	}

	// Feed jobs.
	go func() {
		defer close(jobs)
		for _, label := range wordlist {
			select {
			case <-ctx.Done():
				return
			case jobs <- label:
			}
		}
	}()

	// Close out once all workers finish.
	go func() {
		wg.Wait()
		close(out)
	}()

	var assets []model.Asset
	for r := range out {
		assets = append(assets, model.Asset{
			Kind: model.KindSubdomain,
			Key:  r.host,
			Host: r.host,
			IP:   r.ips,
		})
	}
	return assets
}

func dedupe(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	var out []string
	for _, v := range in {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}
