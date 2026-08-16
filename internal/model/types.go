// Package model defines the core data structures RadarX uses to represent
// targets, discovered assets, point-in-time snapshots, and the diffs between
// them. Everything here is plain data (no behaviour) so it can be serialized
// to JSON today and to SQLite later without changes.
package model

import (
	"strings"
	"time"
)

// SlugID derives a stable Target.ID from a root domain, shared by the CLI and
// GUI so both write the same id to the same store (e.g. "csclub.uz" ->
// "csclub_uz").
func SlugID(root string) string {
	root = strings.ToLower(strings.TrimSpace(root))
	return strings.NewReplacer(".", "_", "/", "_", ":", "_").Replace(root)
}

// Target is a monitored scope root — a root domain the operator is authorized
// to monitor (e.g. "example.com"). RadarX only ever expands assets *within*
// a target's scope.
type Target struct {
	ID         string    `json:"id"`         // stable id (slug of root)
	Root       string    `json:"root"`       // root domain, e.g. example.com
	Label      string    `json:"label"`      // human name / program name
	Enabled    bool      `json:"enabled"`    // whether the scheduler picks it up
	IntervalM  int       `json:"interval_m"` // scan interval in minutes
	Wordlist   []string  `json:"wordlist"`   // optional subdomain wordlist
	AddedAt    time.Time `json:"added_at"`
	LastScanAt time.Time `json:"last_scan_at,omitempty"`
}

// AssetKind classifies what a discovered asset is.
type AssetKind string

const (
	KindSubdomain AssetKind = "subdomain"
	KindPort      AssetKind = "port"
	KindEndpoint  AssetKind = "endpoint"
	KindCert      AssetKind = "cert"
)

// Asset is a single discovered thing that lives under a Target's scope.
// The Key must be stable and unique within (TargetID, Kind) so diffing can
// match the same asset across snapshots.
type Asset struct {
	Kind AssetKind `json:"kind"`
	Key  string    `json:"key"` // stable identity, e.g. "api.example.com" or "api.example.com:443"

	// Enrichment (optional, may be empty depending on Kind):
	Host       string   `json:"host,omitempty"`
	IP         []string `json:"ip,omitempty"`
	Port       int      `json:"port,omitempty"`
	Scheme     string   `json:"scheme,omitempty"`      // http/https
	StatusCode int      `json:"status_code,omitempty"` // HTTP probe result
	Title      string   `json:"title,omitempty"`       // <title> of the page
	Server     string   `json:"server,omitempty"`      // Server header
	CertCN     string   `json:"cert_cn,omitempty"`     // TLS cert common name
	CertExpiry string   `json:"cert_expiry,omitempty"` // TLS notAfter (RFC3339)
}

// Snapshot is the full set of assets discovered for a Target in one scan cycle.
type Snapshot struct {
	TargetID string    `json:"target_id"`
	Root     string    `json:"root"`
	TakenAt  time.Time `json:"taken_at"`
	Assets   []Asset   `json:"assets"`
}

// ChangeType is how an asset changed between two snapshots.
type ChangeType string

const (
	ChangeNew      ChangeType = "new"      // appeared this cycle — the money event
	ChangeRemoved  ChangeType = "removed"  // disappeared
	ChangeModified ChangeType = "modified" // same key, enrichment changed
)

// Change is one diff entry: what happened to one asset.
type Change struct {
	Type   ChangeType `json:"type"`
	Kind   AssetKind  `json:"kind"`
	Key    string     `json:"key"`
	Before *Asset     `json:"before,omitempty"`
	After  *Asset     `json:"after,omitempty"`
	Fields []string   `json:"fields,omitempty"` // which fields changed (for Modified)
}

// DiffResult is the outcome of comparing two snapshots for one target.
type DiffResult struct {
	TargetID string    `json:"target_id"`
	Root     string    `json:"root"`
	At       time.Time `json:"at"`
	Changes  []Change  `json:"changes"`
}

// HasSignal reports whether the diff contains anything worth alerting on.
func (d DiffResult) HasSignal() bool { return len(d.Changes) > 0 }

// NewCount returns how many assets are brand new — the highest-priority signal.
func (d DiffResult) NewCount() int {
	n := 0
	for _, c := range d.Changes {
		if c.Type == ChangeNew {
			n++
		}
	}
	return n
}
