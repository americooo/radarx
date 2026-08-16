package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// TestScanStreamCancelledContext verifies that ScanStream, given an
// already-cancelled context, finishes quickly (no network access needed —
// every DNS/HTTP/dial call bails out immediately on ctx.Done()) and emits
// exactly one terminal event with Done=true.
func TestScanStreamCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the scan even starts

	target := model.Target{ID: "t1", Root: "example.com"}

	done := make(chan struct{})
	var events []ScanEvent
	go func() {
		defer close(done)
		for ev := range ScanStream(ctx, target, ScanOptions{Workers: 4}) {
			events = append(events, ev)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ScanStream did not finish quickly with an already-cancelled context")
	}

	if len(events) != 1 {
		t.Fatalf("expected exactly 1 terminal event, got %d: %+v", len(events), events)
	}
	ev := events[0]
	if !ev.Done {
		t.Fatalf("expected terminal event to have Done=true, got %+v", ev)
	}
	if ev.Asset != nil {
		t.Fatalf("expected no asset on the terminal event, got %+v", ev.Asset)
	}
	if ev.Err == nil || !errors.Is(ev.Err, context.Canceled) {
		t.Fatalf("expected terminal event Err to be context.Canceled, got %v", ev.Err)
	}
}

// TestScanUsesScanStream sanity-checks that Scan (the batch wrapper) also
// returns promptly and with an empty snapshot when given an already-cancelled
// context, since it's just draining ScanStream.
func TestScanUsesScanStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	target := model.Target{ID: "t1", Root: "example.com"}

	done := make(chan model.Snapshot, 1)
	go func() {
		done <- Scan(ctx, target, ScanOptions{Workers: 4})
	}()

	select {
	case snap := <-done:
		if len(snap.Assets) != 0 {
			t.Fatalf("expected no assets from a cancelled scan, got %d", len(snap.Assets))
		}
		if snap.TargetID != target.ID || snap.Root != target.Root {
			t.Fatalf("snapshot metadata mismatch: %+v", snap)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Scan did not finish quickly with an already-cancelled context")
	}
}
