package engine

import (
	"context"
	"sync"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// ScanOptions tunes a scan cycle.
type ScanOptions struct {
	Workers     int  // concurrency for subdomain resolution
	UseCertLogs bool // also pull subdomains from Certificate Transparency (crt.sh)
	ScanPorts   bool // run a light TCP connect port scan on each live host
}

// Scan runs one full recon cycle for a target and returns a snapshot.
// Pipeline: gather candidate subdomains (DNS brute + optional CT) -> resolve ->
// for each live host, probe HTTP and inspect TLS cert concurrently. All
// operations are read-only and authorized-scope only.
func Scan(ctx context.Context, t model.Target, opts ScanOptions) model.Snapshot {
	snap := model.Snapshot{
		TargetID: t.ID,
		Root:     t.Root,
		TakenAt:  time.Now().UTC(),
	}

	// 1. Subdomain discovery: DNS brute-force, optionally augmented with
	//    Certificate Transparency (passive, higher yield).
	subs := EnumerateSubdomains(ctx, t.Root, t.Wordlist, opts.Workers)
	if opts.UseCertLogs {
		if ctSubs, err := EnumerateCertTransparency(ctx, t.Root); err == nil {
			// CT hosts aren't confirmed to resolve; resolve them before trusting.
			subs = mergeSubdomains(subs, resolveHosts(ctx, ctSubs, opts.Workers))
		}
	}

	// Always include the root domain as a candidate host.
	hosts := map[string]struct{}{t.Root: {}}
	for _, a := range subs {
		snap.Assets = append(snap.Assets, a)
		hosts[a.Host] = struct{}{}
	}

	// 2. Enrich each host with HTTP probe + TLS cert, concurrently.
	client := probeClient()
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // bound concurrent enrichment

	for host := range hosts {
		host := host
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if ep, ok := ProbeHTTP(ctx, client, host); ok {
				mu.Lock()
				snap.Assets = append(snap.Assets, ep)
				mu.Unlock()
			}
			if cert, ok := InspectCert(ctx, host); ok {
				mu.Lock()
				snap.Assets = append(snap.Assets, cert)
				mu.Unlock()
			}
			if opts.ScanPorts {
				for _, pa := range ScanPorts(ctx, host, nil, 50) {
					mu.Lock()
					snap.Assets = append(snap.Assets, pa)
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	return snap
}
