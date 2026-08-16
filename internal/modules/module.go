// Package modules defines RadarX's detection module system: a Module is a
// self-contained unit that inspects one asset and emits zero or more
// Findings. internal/modules never imports internal/engine — modules
// consume model.Asset, they don't run scans themselves. This mirrors the
// engine's own rule ("engine never imports other layers") in reverse:
// modules only depend on internal/model.
package modules

import (
	"context"

	"github.com/americooo/radarx/internal/model"
)

// Category classifies what kind of signal a module produces.
type Category string

const (
	CategoryDiscovery  Category = "discovery"
	CategoryChange     Category = "change"
	CategoryVulnSignal Category = "vuln_signal"
)

// Trigger controls which assets a module is run against.
type Trigger string

const (
	TriggerAllAssets     Trigger = "all_assets"
	TriggerNewAssetsOnly Trigger = "new_assets_only"
	TriggerScheduled     Trigger = "scheduled"
)

// State lets a module remember something between runs (e.g. "the JS hash I
// saw last time", "the CT log entries I've already reported") — scoped to
// that one module, backed by the SQLite settings table under the hood.
// Values are opaque strings; a module that needs structure marshals its own
// JSON before calling Set.
type State interface {
	Get(key string) (value string, ok bool, err error)
	Set(key, value string) error
}

// Module is a self-contained detection unit. internal/modules never imports
// internal/engine — modules consume model.Asset, they don't run scans.
type Module interface {
	Name() string
	Category() Category
	Trigger() Trigger
	// Run checks one asset and emits zero or more findings, closing the
	// channel when done (mirrors engine.ScanStream's channel discipline —
	// respect ctx cancellation, always close, never leak a goroutine).
	//
	// target is the asset's owning Target — most modules only need asset,
	// but TriggerScheduled modules operate at the domain level (e.g. "query
	// CT logs for target.Root") and are invoked once per scan cycle with a
	// zero-value asset; they should read target instead. state is this
	// module's private, persistent key/value store for remembering prior
	// results across runs.
	Run(ctx context.Context, target model.Target, asset model.Asset, state State) (<-chan model.Finding, error)
}

var registry []Module

// Register adds a module to the global registry. Call from each module's
// init() so registration is a side effect of importing the package — no
// central "list all modules" file to keep in sync.
func Register(m Module) { registry = append(registry, m) }

// All returns every registered module.
func All() []Module { return registry }
