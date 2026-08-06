// Package store persists targets and snapshots. The MVP uses a zero-dependency
// JSON-on-disk layout so RadarX runs anywhere with no CGO and no external DB.
// The Store interface is the seam: swap in a SQLite implementation later
// without touching the engine or GUI.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// Store is the persistence contract RadarX depends on.
type Store interface {
	SaveTarget(t model.Target) error
	ListTargets() ([]model.Target, error)
	SaveSnapshot(s model.Snapshot) error
	LatestSnapshot(targetID string) (model.Snapshot, bool, error)
	// ListSnapshots returns all snapshots for a target, oldest first.
	ListSnapshots(targetID string) ([]model.Snapshot, error)
}

// JSONStore lays data out under a root directory:
//
//	<root>/targets/<id>.json
//	<root>/snapshots/<id>/<RFC3339>.json
type JSONStore struct {
	root string
}

// NewJSONStore creates the directory layout under root (e.g. ~/.radarx).
func NewJSONStore(root string) (*JSONStore, error) {
	for _, sub := range []string{"targets", "snapshots"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return nil, err
		}
	}
	return &JSONStore{root: root}, nil
}

func (s *JSONStore) SaveTarget(t model.Target) error {
	return writeJSON(filepath.Join(s.root, "targets", t.ID+".json"), t)
}

func (s *JSONStore) ListTargets() ([]model.Target, error) {
	dir := filepath.Join(s.root, "targets")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []model.Target
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		var t model.Target
		if err := readJSON(filepath.Join(dir, e.Name()), &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Root < out[j].Root })
	return out, nil
}

func (s *JSONStore) SaveSnapshot(snap model.Snapshot) error {
	dir := filepath.Join(s.root, "snapshots", snap.TargetID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := snap.TakenAt.UTC().Format("20060102T150405Z") + ".json"
	return writeJSON(filepath.Join(dir, name), snap)
}

// LatestSnapshot returns the most recent snapshot for a target. The bool is
// false (with no error) when the target has never been scanned.
func (s *JSONStore) LatestSnapshot(targetID string) (model.Snapshot, bool, error) {
	dir := filepath.Join(s.root, "snapshots", targetID)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return model.Snapshot{}, false, nil
	}
	if err != nil {
		return model.Snapshot{}, false, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return model.Snapshot{}, false, nil
	}
	sort.Strings(names) // lexicographic == chronological given our name format
	latest := names[len(names)-1]

	var snap model.Snapshot
	if err := readJSON(filepath.Join(dir, latest), &snap); err != nil {
		return model.Snapshot{}, false, err
	}
	return snap, true, nil
}

// ListSnapshots returns all snapshots for a target, oldest first.
func (s *JSONStore) ListSnapshots(targetID string) ([]model.Snapshot, error) {
	dir := filepath.Join(s.root, "snapshots", targetID)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // chronological given our name format
	out := make([]model.Snapshot, 0, len(names))
	for _, n := range names {
		var snap model.Snapshot
		if err := readJSON(filepath.Join(dir, n), &snap); err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, nil
}

func writeJSON(path string, v any) error {
	// Write-to-temp then rename for atomicity (never leave a half-written file).
	tmp := path + fmt.Sprintf(".tmp-%d", time.Now().UnixNano())
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
