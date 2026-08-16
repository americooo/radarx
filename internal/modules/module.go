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

// Module is a self-contained detection unit. internal/modules never imports
// internal/engine — modules consume model.Asset, they don't run scans.
type Module interface {
	Name() string
	Category() Category
	Trigger() Trigger
	// Run checks one asset and emits zero or more findings, closing the
	// channel when done (mirrors engine.ScanStream's channel discipline —
	// respect ctx cancellation, always close, never leak a goroutine).
	Run(ctx context.Context, asset model.Asset) (<-chan model.Finding, error)
}

var registry []Module

// Register adds a module to the global registry. Call from each module's
// init() so registration is a side effect of importing the package — no
// central "list all modules" file to keep in sync.
func Register(m Module) { registry = append(registry, m) }

// All returns every registered module.
func All() []Module { return registry }
