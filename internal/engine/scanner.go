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

// ScanEvent is one increment of a streaming scan: either a newly discovered
// asset, or the terminal event (Done=true, Err set if the scan failed/was
// cancelled).
type ScanEvent struct {
	Asset *model.Asset
	Done  bool
	Err   error
}

// ScanStream runs a scan and emits each discovered asset as it's found,
// followed by exactly one terminal event with Done=true. The channel is
// always closed by ScanStream, even on ctx cancellation or error.
//
// Pipeline: gather candidate subdomains (DNS brute + optional CT) -> resolve ->
// for each live host, probe HTTP and inspect TLS cert concurrently. All
// operations are read-only and authorized-scope only.
//
// If ctx is cancelled mid-scan, assets already discovered are still emitted;
// only further work stops. The terminal event carries ctx.Err() in that case.
func ScanStream(ctx context.Context, t model.Target, opts ScanOptions) <-chan ScanEvent {
	events := make(chan ScanEvent)

	go func() {
		defer close(events)

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
		for i := range subs {
			a := subs[i]
			hosts[a.Host] = struct{}{}
			if !emit(ctx, events, &a) {
				finish(ctx, events)
				return
			}
		}

		// 2. Enrich each host with HTTP probe + TLS cert, concurrently.
		client := probeClient()
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
					emit(ctx, events, &ep)
				}
				if cert, ok := InspectCert(ctx, host); ok {
					emit(ctx, events, &cert)
				}
				if opts.ScanPorts {
					for _, pa := range ScanPorts(ctx, host, nil, 50) {
						pa := pa
						emit(ctx, events, &pa)
					}
				}
			}()
		}
		wg.Wait()

		finish(ctx, events)
	}()

	return events
}

// emit sends an asset event, returning false if ctx was cancelled instead.
func emit(ctx context.Context, events chan<- ScanEvent, a *model.Asset) bool {
	select {
	case events <- ScanEvent{Asset: a}:
		return true
	case <-ctx.Done():
		return false
	}
}

// finish sends the single terminal event, carrying ctx.Err() if the scan was
// cancelled.
func finish(ctx context.Context, events chan<- ScanEvent) {
	events <- ScanEvent{Done: true, Err: ctx.Err()}
}

// Scan runs one full recon cycle for a target and returns a snapshot. It is a
// thin wrapper around ScanStream that drains the stream into a single
// model.Snapshot — the streaming pipeline in ScanStream is the one source of
// scan logic.
func Scan(ctx context.Context, t model.Target, opts ScanOptions) model.Snapshot {
	snap := model.Snapshot{
		TargetID: t.ID,
		Root:     t.Root,
		TakenAt:  time.Now().UTC(),
	}
	for ev := range ScanStream(ctx, t, opts) {
		if ev.Asset != nil {
			snap.Assets = append(snap.Assets, *ev.Asset)
		}
	}
	return snap
}
