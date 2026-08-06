// Package scheduler runs scans in the background: for each enabled target it
// fires a scan cycle on the target's interval, diffs against the last stored
// snapshot, persists the new one, and pushes any signal to the notifier.
//
// This is what turns RadarX from a manual scanner into continuous monitoring.
package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/americooo/radarx/internal/diff"
	"github.com/americooo/radarx/internal/engine"
	"github.com/americooo/radarx/internal/model"
	"github.com/americooo/radarx/internal/notify"
	"github.com/americooo/radarx/internal/store"
)

// Options configures a Scheduler run.
type Options struct {
	Workers   int           // subdomain enum concurrency per scan
	MinPeriod time.Duration // safety floor on how often any target may scan
	Logger    *log.Logger
}

// Scheduler owns the background scan loop.
type Scheduler struct {
	store    store.Store
	notifier notify.Notifier
	opts     Options
	log      *log.Logger

	mu      sync.Mutex
	nextDue map[string]time.Time // targetID -> next scan time
}

// New builds a Scheduler.
func New(st store.Store, n notify.Notifier, opts Options) *Scheduler {
	if opts.Workers <= 0 {
		opts.Workers = 40
	}
	if opts.MinPeriod <= 0 {
		opts.MinPeriod = 5 * time.Minute
	}
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}
	return &Scheduler{
		store:    st,
		notifier: n,
		opts:     opts,
		log:      opts.Logger,
		nextDue:  make(map[string]time.Time),
	}
}

// Run blocks until ctx is cancelled. Every tick it checks which enabled targets
// are due and scans them (each in its own goroutine so a slow target doesn't
// stall the others).
func (s *Scheduler) Run(ctx context.Context) error {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()

	s.log.Printf("scheduler started (min-period %s)", s.opts.MinPeriod)
	s.sweep(ctx) // scan whatever is due immediately on startup

	for {
		select {
		case <-ctx.Done():
			s.log.Println("scheduler stopping")
			return ctx.Err()
		case <-tick.C:
			s.sweep(ctx)
		}
	}
}

// sweep scans all targets that are currently due.
func (s *Scheduler) sweep(ctx context.Context) {
	targets, err := s.store.ListTargets()
	if err != nil {
		s.log.Printf("list targets: %v", err)
		return
	}

	now := time.Now()
	var wg sync.WaitGroup
	for _, t := range targets {
		if !t.Enabled {
			continue
		}
		if !s.due(t, now) {
			continue
		}
		t := t
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scanOne(ctx, t)
		}()
	}
	wg.Wait()
}

// due reports whether a target should scan now, honouring its interval and the
// global minimum period.
func (s *Scheduler) due(t model.Target, now time.Time) bool {
	period := time.Duration(t.IntervalM) * time.Minute
	if period < s.opts.MinPeriod {
		period = s.opts.MinPeriod
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	next, tracked := s.nextDue[t.ID]
	if !tracked {
		// First time we've seen this target this run. Use its stored LastScanAt
		// so restarts don't rescan everything immediately.
		if t.LastScanAt.IsZero() || now.Sub(t.LastScanAt) >= period {
			return true
		}
		s.nextDue[t.ID] = t.LastScanAt.Add(period)
		return false
	}
	return !now.Before(next)
}

func (s *Scheduler) scanOne(ctx context.Context, t model.Target) {
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	start := time.Now()
	snap := engine.Scan(scanCtx, t, engine.ScanOptions{Workers: s.opts.Workers})

	prev, hadPrev, err := s.store.LatestSnapshot(t.ID)
	if err != nil {
		s.log.Printf("[%s] load prev: %v", t.Root, err)
	}

	d := diff.Compare(prev, snap)

	if err := s.store.SaveSnapshot(snap); err != nil {
		s.log.Printf("[%s] save snapshot: %v", t.Root, err)
	}
	t.LastScanAt = snap.TakenAt
	if err := s.store.SaveTarget(t); err != nil {
		s.log.Printf("[%s] save target: %v", t.Root, err)
	}

	// Schedule the next run.
	period := time.Duration(t.IntervalM) * time.Minute
	if period < s.opts.MinPeriod {
		period = s.opts.MinPeriod
	}
	s.mu.Lock()
	s.nextDue[t.ID] = time.Now().Add(period)
	s.mu.Unlock()

	s.log.Printf("[%s] scanned %d assets in %s — %d new, %d changes",
		t.Root, len(snap.Assets), time.Since(start).Round(time.Millisecond),
		d.NewCount(), len(d.Changes))

	// Only alert on the first run's signal if there genuinely is signal; on a
	// true baseline (no previous snapshot) we stay quiet to avoid a flood.
	if hadPrev && d.HasSignal() {
		if err := s.notifier.Notify(d); err != nil {
			s.log.Printf("[%s] notify: %v", t.Root, err)
		}
	}
}
