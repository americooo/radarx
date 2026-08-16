package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/americooo/radarx/internal/engine"
	"github.com/americooo/radarx/internal/model"
)

// App is the Wails-bound backend. It owns the lifecycle of the currently
// running scan (if any) so the frontend "Stop" button has something to
// cancel — internal/engine itself knows nothing about this struct or Wails.
type App struct {
	ctx context.Context

	mu     sync.Mutex
	cancel context.CancelFunc // cancels the in-flight scan, nil if none running
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved so we can
// call the runtime methods (event emission, etc).
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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
func (a *App) StartScan(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return errors.New("target is required")
	}

	a.mu.Lock()
	if a.cancel != nil {
		a.mu.Unlock()
		return errors.New("a scan is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.mu.Unlock()

	target := model.Target{
		ID:      root,
		Root:    root,
		AddedAt: time.Now().UTC(),
	}

	go func() {
		defer func() {
			a.mu.Lock()
			a.cancel = nil
			a.mu.Unlock()
		}()

		var scanErr error
		for ev := range engine.ScanStream(ctx, target, engine.ScanOptions{Workers: 40}) {
			if ev.Asset != nil {
				runtime.EventsEmit(a.ctx, "scan:asset", ev.Asset)
			}
			if ev.Done {
				scanErr = ev.Err
			}
		}

		done := scanDoneEvent{Cancelled: errors.Is(scanErr, context.Canceled)}
		if scanErr != nil && !done.Cancelled {
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
