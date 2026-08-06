// Package web serves RadarX's local dashboard: a single-page view of every
// monitored target, its current assets, and recent changes. It uses only the
// Go standard library (net/http + embed) so there is no build toolchain, no
// node, and no external dependency — true local-first.
//
// The dashboard is read-only and meant to be bound to localhost.
package web

import (
	"embed"
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/americooo/radarx/internal/diff"
	"github.com/americooo/radarx/internal/model"
	"github.com/americooo/radarx/internal/store"
)

//go:embed assets/*
var assets embed.FS

// Server serves the dashboard from a Store.
type Server struct {
	store store.Store
}

// New builds a dashboard server.
func New(st store.Store) *Server { return &Server{store: st} }

// Handler returns the HTTP handler (routes + static assets).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/state", s.handleState)
	mux.Handle("/", http.FileServer(http.FS(assets)))
	// Serve index at root.
	mux.HandleFunc("/index.html", s.serveIndex)
	mux.HandleFunc("/dashboard", s.serveIndex)
	return withRoot(mux, s)
}

func withRoot(mux *http.ServeMux, s *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			s.serveIndex(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := assets.ReadFile("assets/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// stateResponse is the JSON payload the frontend polls.
type stateResponse struct {
	GeneratedAt time.Time    `json:"generated_at"`
	Targets     []targetView `json:"targets"`
}

type targetView struct {
	ID          string         `json:"id"`
	Root        string         `json:"root"`
	Label       string         `json:"label"`
	Enabled     bool           `json:"enabled"`
	IntervalM   int            `json:"interval_m"`
	LastScanAt  string         `json:"last_scan_at"`
	AssetCounts map[string]int `json:"asset_counts"`
	TotalAssets int            `json:"total_assets"`
	Assets      []model.Asset  `json:"assets"`
	Recent      []model.Change `json:"recent_changes"`
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	targets, err := s.store.ListTargets()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := stateResponse{GeneratedAt: time.Now()}
	for _, t := range targets {
		tv := targetView{
			ID:          t.ID,
			Root:        t.Root,
			Label:       t.Label,
			Enabled:     t.Enabled,
			IntervalM:   t.IntervalM,
			AssetCounts: map[string]int{},
		}
		if !t.LastScanAt.IsZero() {
			tv.LastScanAt = t.LastScanAt.Format(time.RFC3339)
		}

		snaps, _ := s.store.ListSnapshots(t.ID)
		if n := len(snaps); n > 0 {
			latest := snaps[n-1]
			tv.Assets = latest.Assets
			tv.TotalAssets = len(latest.Assets)
			for _, a := range latest.Assets {
				tv.AssetCounts[string(a.Kind)]++
			}
			// Recent changes = diff of last two snapshots.
			if n >= 2 {
				d := diff.Compare(snaps[n-2], latest)
				tv.Recent = d.Changes
			}
		}
		resp.Targets = append(resp.Targets, tv)
	}

	sort.Slice(resp.Targets, func(i, j int) bool {
		return resp.Targets[i].Root < resp.Targets[j].Root
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
