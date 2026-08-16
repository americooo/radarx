package modules

import (
	"context"
	"sync"

	"github.com/americooo/radarx/internal/model"
)

// Run executes every registered module against the appropriate asset set for
// its Trigger:
//
//   - TriggerNewAssetsOnly modules run once per asset in newAssets (from a
//     diff's ChangeNew entries).
//   - TriggerAllAssets modules run once per asset in allAssets.
//   - TriggerScheduled modules run exactly once per call (not per-asset),
//     with a zero-value model.Asset — they operate on target as a whole
//     (e.g. "query CT logs for target.Root"), not a specific discovered
//     asset.
//
// Findings from all modules fan into one channel, closed once every module
// has finished. Modules run concurrently, and each module's per-asset checks
// also run concurrently (bounded) — mirrors internal/engine/scanner.go's
// semaphore pattern. Each module gets its own namespaced State (see
// NewState) for remembering results across calls.
func Run(ctx context.Context, target model.Target, newAssets, allAssets []model.Asset, store SettingsStore) <-chan model.Finding {
	out := make(chan model.Finding)

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		for _, m := range All() {
			m := m
			state := NewState(store, m.Name())

			if m.Trigger() == TriggerScheduled {
				wg.Add(1)
				go func() {
					defer wg.Done()
					runOnce(ctx, m, target, model.Asset{}, state, out)
				}()
				continue
			}

			var assets []model.Asset
			if m.Trigger() == TriggerNewAssetsOnly {
				assets = newAssets
			} else {
				assets = allAssets
			}
			if len(assets) == 0 {
				continue
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				runPerAsset(ctx, m, target, assets, state, out)
			}()
		}

		wg.Wait()
	}()

	return out
}

// runPerAsset runs one module against every asset in its scope, bounding
// per-asset concurrency, and forwards every finding it emits into out.
func runPerAsset(ctx context.Context, m Module, target model.Target, assets []model.Asset, state State, out chan<- model.Finding) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20) // bound concurrent per-asset checks

	for _, a := range assets {
		a := a
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			runOnce(ctx, m, target, a, state, out)
		}()
	}
	wg.Wait()
}

// runOnce runs a module a single time (one asset check, or one whole-target
// pass for TriggerScheduled modules) and forwards its findings into out.
func runOnce(ctx context.Context, m Module, target model.Target, asset model.Asset, state State, out chan<- model.Finding) {
	findings, err := m.Run(ctx, target, asset, state)
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
}
