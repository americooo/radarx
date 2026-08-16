package modules

import (
	"context"
	"sync"

	"github.com/americooo/radarx/internal/model"
)

// Run executes every registered module against the appropriate asset set
// for its Trigger: TriggerNewAssetsOnly modules only see newAssets (from a
// diff's ChangeNew entries), TriggerAllAssets and TriggerScheduled modules
// see every asset in the current snapshot. Findings from all modules fan
// into one channel, closed once every module has finished. Modules run
// concurrently, and each module's per-asset checks also run concurrently
// (bounded) — mirrors internal/engine/scanner.go's semaphore pattern.
//
// TriggerScheduled modules currently run every cycle against allAssets, same
// as TriggerAllAssets; time-based scheduling (e.g. "query CT logs every N
// hours") is deferred to the module that needs it.
func Run(ctx context.Context, newAssets, allAssets []model.Asset) <-chan model.Finding {
	out := make(chan model.Finding)

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		for _, m := range All() {
			var assets []model.Asset
			switch m.Trigger() {
			case TriggerNewAssetsOnly:
				assets = newAssets
			default: // TriggerAllAssets, TriggerScheduled
				assets = allAssets
			}
			if len(assets) == 0 {
				continue
			}

			m := m
			wg.Add(1)
			go func() {
				defer wg.Done()
				runModule(ctx, m, assets, out)
			}()
		}

		wg.Wait()
	}()

	return out
}

// runModule runs one module against every asset in its scope, bounding
// per-asset concurrency, and forwards every finding it emits into out.
func runModule(ctx context.Context, m Module, assets []model.Asset, out chan<- model.Finding) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // bound concurrent per-asset checks

	for _, a := range assets {
		a := a
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			findings, err := m.Run(ctx, a)
			if err != nil {
				return
			}
			for f := range findings {
				select {
				case out <- f:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	wg.Wait()
}
