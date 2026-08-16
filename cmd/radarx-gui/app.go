package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/americooo/radarx/internal/diff"
	"github.com/americooo/radarx/internal/engine"
	"github.com/americooo/radarx/internal/model"
	"github.com/americooo/radarx/internal/notify"
	"github.com/americooo/radarx/internal/report"
	"github.com/americooo/radarx/internal/scheduler"
	"github.com/americooo/radarx/internal/scope"
	"github.com/americooo/radarx/internal/store"
)

// App is the Wails-bound backend. It owns the lifecycle of the currently
// running scan (if any) so the frontend "Stop" button has something to
// cancel — internal/engine itself knows nothing about this struct or Wails.
type App struct {
	ctx context.Context

	mu     sync.Mutex
	cancel context.CancelFunc // cancels the in-flight scan, nil if none running

	// schedMu/schedCancel track the background continuous-monitoring loop
	// (internal/scheduler). Kept separate from mu/cancel above: a manual
	// one-off scan (StartScan/StopScan) and background monitoring
	// (StartMonitoring/StopMonitoring) are independent and can run at the
	// same time without contending on the same lock.
	schedMu     sync.Mutex
	schedCancel context.CancelFunc

	st store.Store // nil if the store failed to open (methods below check this)
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// storePath returns ~/.radarx/radarx.db — must stay identical to
// cmd/radarx/main.go's openStore() so the CLI and GUI share one database.
func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".radarx", "radarx.db"), nil
}

// scopePath returns ~/.radarx/scope.txt — must stay identical to
// cmd/radarx/main.go's scopePath() so both share one scope file.
func scopePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".radarx", "scope.txt"), nil
}

// openAppStore opens the shared SQLite store at storePath().
func openAppStore() (store.Store, error) {
	p, err := storePath()
	if err != nil {
		return nil, err
	}
	return store.NewSQLiteStore(p)
}

// startup is called when the app starts. The context is saved so we can
// call the runtime methods (event emission, etc). The store is opened here;
// if it fails, a is left with a nil store and every store-backed method
// below returns a clear error instead of panicking.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	st, err := openAppStore()
	if err != nil {
		log.Printf("radarx-gui: failed to open store: %v", err)
		return
	}
	a.st = st
}

// scanDoneEvent is the payload sent on "scan:done".
type scanDoneEvent struct {
	Err       string `json:"err,omitempty"`
	Cancelled bool   `json:"cancelled"`
}

// StartScan kicks off a scan of root in the background and returns
// immediately — it does not block the UI thread. Progress streams to the
// frontend via the "scan:asset" event (one per discovered model.Asset) and
// terminates with a single "scan:done" event.
//
// Before anything touches the network, root must pass the same scope gate
// the CLI enforces (internal/scope.CheckPath) — this is the one safety rule
// the GUI is never allowed to bypass.
func (a *App) StartScan(root string) error {
	root = strings.ToLower(strings.TrimSpace(root))
	if root == "" {
		return errors.New("target is required")
	}

	sp, err := scopePath()
	if err != nil {
		return err
	}
	if err := scope.CheckPath(sp, root); err != nil {
		return fmt.Errorf("scope violation: %w", err)
	}

	a.mu.Lock()
	if a.cancel != nil {
		a.mu.Unlock()
		return errors.New("a scan is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.mu.Unlock()

	// Reuse the existing target record (if any) so LastScanAt/label/interval
	// survive, mirroring cmdScan's "load or create ephemeral" behaviour.
	id := model.SlugID(root)
	target := model.Target{ID: id, Root: root, Enabled: true, IntervalM: 60, AddedAt: time.Now().UTC()}
	if a.st != nil {
		if existing, ok := a.findTarget(id); ok {
			target = existing
		}
	}

	go func() {
		defer func() {
			a.mu.Lock()
			a.cancel = nil
			a.mu.Unlock()
		}()

		snap := model.Snapshot{TargetID: target.ID, Root: target.Root, TakenAt: time.Now().UTC()}
		var scanErr error
		for ev := range engine.ScanStream(ctx, target, engine.ScanOptions{Workers: 40}) {
			if ev.Asset != nil {
				snap.Assets = append(snap.Assets, *ev.Asset)
				runtime.EventsEmit(a.ctx, "scan:asset", ev.Asset)
			}
			if ev.Done {
				scanErr = ev.Err
			}
		}

		cancelled := errors.Is(scanErr, context.Canceled)
		if a.st != nil && !cancelled && scanErr == nil {
			if err := a.st.SaveSnapshot(snap); err != nil {
				log.Printf("radarx-gui: SaveSnapshot failed: %v", err)
			}
			target.LastScanAt = snap.TakenAt
			if err := a.st.SaveTarget(target); err != nil {
				log.Printf("radarx-gui: SaveTarget failed: %v", err)
			}
		}

		done := scanDoneEvent{Cancelled: cancelled}
		if scanErr != nil && !cancelled {
			done.Err = scanErr.Error()
		}
		runtime.EventsEmit(a.ctx, "scan:done", done)
	}()

	return nil
}

// StopScan cancels the in-flight scan, if any. This is the only mechanism
// for stopping a scan — every engine call respects ctx cancellation.
func (a *App) StopScan() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancel == nil {
		return fmt.Errorf("no scan is running")
	}
	a.cancel()
	return nil
}

// StartMonitoring starts the background scheduler (internal/scheduler) which
// periodically rescans every enabled target on its configured interval,
// diffs against the last snapshot, and pushes any signal through
// notify.Default() (console + desktop, plus Telegram if configured). It
// returns immediately; the scheduler runs until StopMonitoring is called or
// the app shuts down (see OnShutdown in main.go).
func (a *App) StartMonitoring() error {
	if a.st == nil {
		return errors.New("store is not available")
	}

	a.schedMu.Lock()
	defer a.schedMu.Unlock()
	if a.schedCancel != nil {
		return errors.New("monitoring is already running")
	}

	sp, err := scopePath()
	if err != nil {
		return err
	}

	sched := scheduler.New(a.st, notify.Default(), scheduler.Options{ScopePath: sp})

	ctx, cancel := context.WithCancel(context.Background())
	a.schedCancel = cancel

	go func() {
		if err := sched.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("radarx-gui: scheduler stopped: %v", err)
		}
	}()

	return nil
}

// StopMonitoring cancels the background scheduler started by StartMonitoring.
func (a *App) StopMonitoring() error {
	a.schedMu.Lock()
	defer a.schedMu.Unlock()

	if a.schedCancel == nil {
		return errors.New("monitoring is not running")
	}
	a.schedCancel()
	a.schedCancel = nil
	return nil
}

// IsMonitoring reports whether the background scheduler is currently
// running, so the frontend can render an accurate toggle state.
func (a *App) IsMonitoring() bool {
	a.schedMu.Lock()
	defer a.schedMu.Unlock()
	return a.schedCancel != nil
}

// ExportReport writes a HackerOne-ready markdown asset inventory for the
// target's latest snapshot to ~/.radarx/reports/<targetID>-<timestamp>.md
// and returns the path written.
func (a *App) ExportReport(targetID string) (string, error) {
	if a.st == nil {
		return "", errors.New("store is not available")
	}

	snap, ok, err := a.st.LatestSnapshot(targetID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("no scan yet for this target")
	}

	t, found := a.findTarget(targetID)
	if !found {
		t = model.Target{ID: targetID, Root: snap.Root}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".radarx", "reports")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	name := fmt.Sprintf("%s-%s.md", targetID, time.Now().UTC().Format(time.RFC3339))
	name = strings.ReplaceAll(name, ":", "-") // RFC3339 colons are awkward in filenames on Windows
	path := filepath.Join(dir, name)

	md := report.Inventory(t, snap)
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// findTarget looks up a target by id in the store. Callers must ensure
// a.st is non-nil.
func (a *App) findTarget(id string) (model.Target, bool) {
	targets, err := a.st.ListTargets()
	if err != nil {
		return model.Target{}, false
	}
	for _, t := range targets {
		if t.ID == id {
			return t, true
		}
	}
	return model.Target{}, false
}

// ListTargets returns every target known to the store, ordered by root
// (see store.Store.ListTargets).
func (a *App) ListTargets() ([]model.Target, error) {
	if a.st == nil {
		return nil, errors.New("store is not available")
	}
	return a.st.ListTargets()
}

// AddTarget saves a new (or updated) target. root must already be covered
// by the operator's scope file — if it isn't, this returns an error the
// frontend recognizes and turns into an explicit authorization prompt
// (see AuthorizeTarget) rather than silently adding an unauthorized domain.
func (a *App) AddTarget(root, label string, intervalMinutes int) error {
	if a.st == nil {
		return errors.New("store is not available")
	}
	root = strings.ToLower(strings.TrimSpace(root))
	if root == "" {
		return errors.New("root domain is required")
	}

	sp, err := scopePath()
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(sp); os.IsNotExist(statErr) {
		return fmt.Errorf("not authorized: %s is not in scope yet, call AuthorizeTarget first", root)
	}
	sc, err := scope.Load(sp)
	if err != nil {
		return fmt.Errorf("not authorized: %s is not in scope yet, call AuthorizeTarget first", root)
	}
	if !sc.Allows(root) {
		return fmt.Errorf("not authorized: %s is not in scope yet, call AuthorizeTarget first", root)
	}

	t := model.Target{
		ID:        model.SlugID(root),
		Root:      root,
		Label:     label,
		Enabled:   true,
		IntervalM: intervalMinutes,
		AddedAt:   time.Now().UTC(),
	}
	return a.st.SaveTarget(t)
}

// AuthorizeTarget records root as an explicitly authorized scope entry.
// It must only be called after the operator has confirmed authorization in
// the UI — this function itself does no confirming, it only persists the
// decision (creating ~/.radarx/scope.txt if it doesn't exist yet, appending
// to it otherwise), mirroring the CLI's `radarx add` scope-confirmation flow.
func (a *App) AuthorizeTarget(root string) error {
	root = strings.ToLower(strings.TrimSpace(root))
	if root == "" {
		return errors.New("root domain is required")
	}

	sp, err := scopePath()
	if err != nil {
		return err
	}

	if _, statErr := os.Stat(sp); os.IsNotExist(statErr) {
		if err := os.MkdirAll(filepath.Dir(sp), 0o755); err != nil {
			return err
		}
		return os.WriteFile(sp, []byte(root+"\n"), 0o644)
	}

	sc, err := scope.Load(sp)
	if err == nil && sc.Allows(root) {
		return nil // already authorized, nothing to do
	}

	f, err := os.OpenFile(sp, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(root + "\n")
	return err
}

// GetScopeRoots returns the operator's currently authorized root domains.
// A missing scope file is not an error — it returns an empty slice so the
// Settings screen can render a "nothing configured yet" state.
func (a *App) GetScopeRoots() ([]string, error) {
	sp, err := scopePath()
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(sp); os.IsNotExist(statErr) {
		return []string{}, nil
	}
	sc, err := scope.Load(sp)
	if err != nil {
		return []string{}, nil
	}
	return sc.Roots(), nil
}

// GetLatestSnapshot returns the most recent snapshot for a target. If the
// target has never been scanned, it returns a zero-value Snapshot and an
// error explaining that, so the frontend can show a clear "not scanned yet"
// state instead of an empty table with no explanation.
func (a *App) GetLatestSnapshot(targetID string) (model.Snapshot, error) {
	if a.st == nil {
		return model.Snapshot{}, errors.New("store is not available")
	}
	snap, ok, err := a.st.LatestSnapshot(targetID)
	if err != nil {
		return model.Snapshot{}, err
	}
	if !ok {
		return model.Snapshot{}, fmt.Errorf("no snapshot yet for %s — run a scan first", targetID)
	}
	if snap.Assets == nil {
		// A nil slice marshals to JSON `null`, not `[]` — the frontend does
		// `snapshot.assets.length` and would crash on that. Never hand it a
		// nil slice for a field it's going to iterate/measure.
		snap.Assets = []model.Asset{}
	}
	return snap, nil
}

// GetDiff compares the two most recent snapshots for a target. If fewer
// than two snapshots exist yet, it returns an error explaining there isn't
// enough history to diff.
func (a *App) GetDiff(targetID string) (model.DiffResult, error) {
	if a.st == nil {
		return model.DiffResult{}, errors.New("store is not available")
	}
	snaps, err := a.st.ListSnapshots(targetID)
	if err != nil {
		return model.DiffResult{}, err
	}
	if len(snaps) < 2 {
		return model.DiffResult{}, fmt.Errorf("not enough scan history yet for %s (need at least 2 scans)", targetID)
	}
	prev, latest := snaps[len(snaps)-2], snaps[len(snaps)-1]
	d := diff.Compare(prev, latest)
	if d.Changes == nil {
		// Same nil-slice-marshals-to-null trap as GetLatestSnapshot: a diff
		// with no changes (the common case) must still be a JSON `[]` so
		// result.changes.length in DiffView.tsx doesn't throw on `null`.
		d.Changes = []model.Change{}
	}
	return d, nil
}

// GetTelegramStatus reports whether Telegram notifications are configured
// via environment variables. This is read-only — configuring/saving
// Telegram credentials from the GUI is out of scope for this phase.
func (a *App) GetTelegramStatus() bool {
	return os.Getenv("RADARX_TG_TOKEN") != "" && os.Getenv("RADARX_TG_CHAT_ID") != ""
}
