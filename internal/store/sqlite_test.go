package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/diff"
	"github.com/americooo/radarx/internal/model"
)

func openTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "radarx.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSQLiteSaveTargetRoundTrip(t *testing.T) {
	s := openTestStore(t)

	target := model.Target{
		ID:        "example_com",
		Root:      "example.com",
		Label:     "Acme BBP",
		Enabled:   true,
		IntervalM: 60,
		Wordlist:  []string{"api", "admin", "www"},
		AddedAt:   time.Now().UTC().Truncate(time.Second),
	}
	if err := s.SaveTarget(target); err != nil {
		t.Fatalf("SaveTarget: %v", err)
	}

	got, err := s.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 target, got %d", len(got))
	}
	gt := got[0]
	if gt.ID != target.ID || gt.Root != target.Root || gt.Label != target.Label ||
		gt.Enabled != target.Enabled || gt.IntervalM != target.IntervalM {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", gt, target)
	}
	if len(gt.Wordlist) != 3 || gt.Wordlist[0] != "api" {
		t.Fatalf("wordlist round-trip mismatch: got %v", gt.Wordlist)
	}
	if !gt.AddedAt.Equal(target.AddedAt) {
		t.Fatalf("added_at mismatch: got %v, want %v", gt.AddedAt, target.AddedAt)
	}
	if !gt.LastScanAt.IsZero() {
		t.Fatalf("expected zero LastScanAt, got %v", gt.LastScanAt)
	}
}

func TestSQLiteSaveTargetUpsert(t *testing.T) {
	s := openTestStore(t)

	t1 := model.Target{ID: "x", Root: "x.com", Label: "v1", Enabled: true, IntervalM: 30, AddedAt: time.Now().UTC()}
	if err := s.SaveTarget(t1); err != nil {
		t.Fatalf("SaveTarget: %v", err)
	}
	t1.Label = "v2"
	t1.IntervalM = 45
	t1.LastScanAt = time.Now().UTC().Truncate(time.Second)
	if err := s.SaveTarget(t1); err != nil {
		t.Fatalf("SaveTarget (update): %v", err)
	}

	got, err := s.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 target after upsert, got %d", len(got))
	}
	if got[0].Label != "v2" || got[0].IntervalM != 45 {
		t.Fatalf("expected upsert to apply, got %+v", got[0])
	}
	if !got[0].LastScanAt.Equal(t1.LastScanAt) {
		t.Fatalf("last_scan_at mismatch: got %v, want %v", got[0].LastScanAt, t1.LastScanAt)
	}
}

func TestSQLiteListTargetsOrderedByRoot(t *testing.T) {
	s := openTestStore(t)

	for _, root := range []string{"zeta.com", "alpha.com", "mid.com"} {
		must(t, s.SaveTarget(model.Target{ID: root, Root: root, AddedAt: time.Now().UTC()}))
	}

	got, err := s.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	want := []string{"alpha.com", "mid.com", "zeta.com"}
	if len(got) != len(want) {
		t.Fatalf("expected %d targets, got %d", len(want), len(got))
	}
	for i, w := range want {
		if got[i].Root != w {
			t.Fatalf("expected order %v, got %v", want, rootsOf(got))
		}
	}
}

func TestSQLiteSnapshotRoundTrip(t *testing.T) {
	s := openTestStore(t)
	must(t, s.SaveTarget(model.Target{ID: "t1", Root: "t1.com", AddedAt: time.Now().UTC()}))

	snap := model.Snapshot{
		TargetID: "t1",
		Root:     "t1.com",
		TakenAt:  time.Now().UTC().Truncate(time.Second),
		Assets: []model.Asset{
			{Kind: model.KindSubdomain, Key: "api.t1.com", Host: "api.t1.com", IP: []string{"1.2.3.4", "5.6.7.8"}},
			{Kind: model.KindEndpoint, Key: "https://api.t1.com", Host: "api.t1.com", Scheme: "https",
				StatusCode: 200, Title: "Welcome", Server: "nginx"},
			{Kind: model.KindCert, Key: "api.t1.com:443", Host: "api.t1.com", Port: 443,
				CertCN: "api.t1.com", CertExpiry: "2027-01-01T00:00:00Z"},
		},
	}
	if err := s.SaveSnapshot(snap); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	got, ok, err := s.LatestSnapshot("t1")
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if !ok {
		t.Fatal("expected LatestSnapshot to find a snapshot")
	}
	if got.TargetID != snap.TargetID || got.Root != snap.Root {
		t.Fatalf("snapshot header mismatch: got %+v", got)
	}
	if !got.TakenAt.Equal(snap.TakenAt) {
		t.Fatalf("taken_at mismatch: got %v, want %v", got.TakenAt, snap.TakenAt)
	}
	if len(got.Assets) != 3 {
		t.Fatalf("expected 3 assets, got %d", len(got.Assets))
	}

	sub := got.Assets[0]
	if sub.Kind != model.KindSubdomain || sub.Key != "api.t1.com" || len(sub.IP) != 2 ||
		sub.IP[0] != "1.2.3.4" || sub.IP[1] != "5.6.7.8" {
		t.Fatalf("subdomain asset round-trip mismatch: %+v", sub)
	}

	ep := got.Assets[1]
	if ep.Kind != model.KindEndpoint || ep.StatusCode != 200 || ep.Title != "Welcome" || ep.Server != "nginx" {
		t.Fatalf("endpoint asset round-trip mismatch: %+v", ep)
	}

	cert := got.Assets[2]
	if cert.Kind != model.KindCert || cert.CertCN != "api.t1.com" || cert.Port != 443 {
		t.Fatalf("cert asset round-trip mismatch: %+v", cert)
	}
}

func TestSQLiteLatestSnapshotNoneYet(t *testing.T) {
	s := openTestStore(t)
	_, ok, err := s.LatestSnapshot("nonexistent")
	if err != nil {
		t.Fatalf("LatestSnapshot: %v", err)
	}
	if ok {
		t.Fatal("expected ok=false for target with no snapshots")
	}
}

func TestSQLiteListSnapshotsOldestFirst(t *testing.T) {
	s := openTestStore(t)
	must(t, s.SaveTarget(model.Target{ID: "t1", Root: "t1.com", AddedAt: time.Now().UTC()}))

	base := time.Now().UTC().Truncate(time.Second)
	for i, delta := range []time.Duration{2 * time.Hour, 0, 1 * time.Hour} {
		snap := model.Snapshot{
			TargetID: "t1",
			Root:     "t1.com",
			TakenAt:  base.Add(delta),
			Assets:   []model.Asset{{Kind: model.KindSubdomain, Key: "gen", Host: "gen"}},
		}
		if err := s.SaveSnapshot(snap); err != nil {
			t.Fatalf("SaveSnapshot #%d: %v", i, err)
		}
	}

	got, err := s.ListSnapshots("t1")
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].TakenAt.After(got[i].TakenAt) {
			t.Fatalf("snapshots not oldest-first: %v then %v", got[i-1].TakenAt, got[i].TakenAt)
		}
	}
	// oldest first should have base+0, base+1h, base+2h
	if !got[0].TakenAt.Equal(base) || !got[1].TakenAt.Equal(base.Add(time.Hour)) || !got[2].TakenAt.Equal(base.Add(2*time.Hour)) {
		t.Fatalf("unexpected ordering: %v, %v, %v", got[0].TakenAt, got[1].TakenAt, got[2].TakenAt)
	}
}

func TestSQLiteMultipleTargetsIsolateSnapshots(t *testing.T) {
	s := openTestStore(t)
	must(t, s.SaveTarget(model.Target{ID: "a", Root: "a.com", AddedAt: time.Now().UTC()}))
	must(t, s.SaveTarget(model.Target{ID: "b", Root: "b.com", AddedAt: time.Now().UTC()}))

	must(t, s.SaveSnapshot(model.Snapshot{TargetID: "a", Root: "a.com", TakenAt: time.Now().UTC(),
		Assets: []model.Asset{{Kind: model.KindSubdomain, Key: "x.a.com", Host: "x.a.com"}}}))
	must(t, s.SaveSnapshot(model.Snapshot{TargetID: "b", Root: "b.com", TakenAt: time.Now().UTC(),
		Assets: []model.Asset{{Kind: model.KindSubdomain, Key: "y.b.com", Host: "y.b.com"}, {Kind: model.KindSubdomain, Key: "z.b.com", Host: "z.b.com"}}}))

	aSnaps, err := s.ListSnapshots("a")
	if err != nil {
		t.Fatalf("ListSnapshots(a): %v", err)
	}
	if len(aSnaps) != 1 || len(aSnaps[0].Assets) != 1 {
		t.Fatalf("expected target a to have 1 snapshot with 1 asset, got %+v", aSnaps)
	}

	bSnaps, err := s.ListSnapshots("b")
	if err != nil {
		t.Fatalf("ListSnapshots(b): %v", err)
	}
	if len(bSnaps) != 1 || len(bSnaps[0].Assets) != 2 {
		t.Fatalf("expected target b to have 1 snapshot with 2 assets, got %+v", bSnaps)
	}
}

// TestSQLiteMigrationFromJSONStore verifies that pointing NewSQLiteStore at a
// directory that already has a legacy JSONStore layout (but no SQLite DB yet)
// imports the existing targets and snapshots exactly once.
func TestSQLiteMigrationFromJSONStore(t *testing.T) {
	root := t.TempDir()

	jsonStore, err := NewJSONStore(root)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}

	target := model.Target{
		ID: "legacy_com", Root: "legacy.com", Label: "Legacy",
		Enabled: true, IntervalM: 60, AddedAt: time.Now().UTC().Truncate(time.Second),
	}
	if err := jsonStore.SaveTarget(target); err != nil {
		t.Fatalf("jsonStore.SaveTarget: %v", err)
	}

	snap := model.Snapshot{
		TargetID: target.ID,
		Root:     target.Root,
		TakenAt:  time.Now().UTC().Truncate(time.Second),
		Assets: []model.Asset{
			{Kind: model.KindSubdomain, Key: "api.legacy.com", Host: "api.legacy.com", IP: []string{"9.9.9.9"}},
		},
	}
	if err := jsonStore.SaveSnapshot(snap); err != nil {
		t.Fatalf("jsonStore.SaveSnapshot: %v", err)
	}

	// Now open a SQLiteStore rooted at the same directory as the JSON layout.
	dbPath := filepath.Join(root, "radarx.db")
	sqliteStore, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer sqliteStore.Close()

	targets, err := sqliteStore.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets after migration: %v", err)
	}
	if len(targets) != 1 || targets[0].ID != target.ID || targets[0].Root != target.Root {
		t.Fatalf("expected migrated target, got %+v", targets)
	}

	snaps, err := sqliteStore.ListSnapshots(target.ID)
	if err != nil {
		t.Fatalf("ListSnapshots after migration: %v", err)
	}
	if len(snaps) != 1 || len(snaps[0].Assets) != 1 {
		t.Fatalf("expected migrated snapshot with 1 asset, got %+v", snaps)
	}
	if snaps[0].Assets[0].Key != "api.legacy.com" || len(snaps[0].Assets[0].IP) != 1 || snaps[0].Assets[0].IP[0] != "9.9.9.9" {
		t.Fatalf("migrated asset mismatch: %+v", snaps[0].Assets[0])
	}
}

// TestSQLiteMigrationSkippedWhenAlreadyPopulated ensures the one-time import
// doesn't run (or duplicate data) once the SQLite DB already has targets.
func TestSQLiteMigrationSkippedWhenAlreadyPopulated(t *testing.T) {
	root := t.TempDir()

	dbPath := filepath.Join(root, "radarx.db")
	s, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	must(t, s.SaveTarget(model.Target{ID: "fresh", Root: "fresh.com", AddedAt: time.Now().UTC()}))
	s.Close()

	// Now create a legacy JSON layout alongside it with a *different* target.
	jsonStore, err := NewJSONStore(root)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	must(t, jsonStore.SaveTarget(model.Target{ID: "legacy", Root: "legacy.com", AddedAt: time.Now().UTC()}))

	// Reopening SQLiteStore should NOT import the legacy target since it
	// already has data.
	s2, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore (reopen): %v", err)
	}
	defer s2.Close()

	targets, err := s2.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(targets) != 1 || targets[0].ID != "fresh" {
		t.Fatalf("expected migration to be skipped, got %+v", targets)
	}
}

// TestSQLiteDiffAcrossTwoScans exercises the Faza 2 "done" criterion at the
// store+diff level: two SaveSnapshot calls for the same target, diffed with
// internal/diff, correctly reports new/removed/modified assets — regardless
// of which Store implementation produced the snapshots.
func TestSQLiteDiffAcrossTwoScans(t *testing.T) {
	s := openTestStore(t)
	must(t, s.SaveTarget(model.Target{ID: "t1", Root: "t1.com", AddedAt: time.Now().UTC()}))

	first := model.Snapshot{
		TargetID: "t1", Root: "t1.com", TakenAt: time.Now().UTC(),
		Assets: []model.Asset{
			{Kind: model.KindSubdomain, Key: "www.t1.com", Host: "www.t1.com"},
			{Kind: model.KindEndpoint, Key: "https://www.t1.com", Host: "www.t1.com", Scheme: "https", StatusCode: 403, Title: "Forbidden"},
		},
	}
	must(t, s.SaveSnapshot(first))

	second := model.Snapshot{
		TargetID: "t1", Root: "t1.com", TakenAt: first.TakenAt.Add(time.Hour),
		Assets: []model.Asset{
			{Kind: model.KindSubdomain, Key: "www.t1.com", Host: "www.t1.com"},
			{Kind: model.KindSubdomain, Key: "admin.t1.com", Host: "admin.t1.com"}, // new
			{Kind: model.KindEndpoint, Key: "https://www.t1.com", Host: "www.t1.com", Scheme: "https", StatusCode: 200, Title: "Welcome"}, // modified
		},
	}
	must(t, s.SaveSnapshot(second))

	snaps, err := s.ListSnapshots("t1")
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}

	d := diff.Compare(snaps[0], snaps[1])
	if d.NewCount() != 1 {
		t.Fatalf("expected 1 new asset, got %d (%+v)", d.NewCount(), d.Changes)
	}
	var sawNew, sawModified bool
	for _, c := range d.Changes {
		if c.Type == model.ChangeNew && c.Key == "admin.t1.com" {
			sawNew = true
		}
		if c.Type == model.ChangeModified && c.Key == "https://www.t1.com" {
			sawModified = true
		}
	}
	if !sawNew {
		t.Fatalf("expected new admin.t1.com in diff, got %+v", d.Changes)
	}
	if !sawModified {
		t.Fatalf("expected modified https://www.t1.com in diff, got %+v", d.Changes)
	}

	// LatestSnapshot must agree with the last entry of ListSnapshots.
	latest, ok, err := s.LatestSnapshot("t1")
	if err != nil || !ok {
		t.Fatalf("LatestSnapshot: ok=%v err=%v", ok, err)
	}
	if !latest.TakenAt.Equal(second.TakenAt) {
		t.Fatalf("LatestSnapshot mismatch: got %v, want %v", latest.TakenAt, second.TakenAt)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func rootsOf(ts []model.Target) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Root
	}
	return out
}
