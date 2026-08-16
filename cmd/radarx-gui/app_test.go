package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/americooo/radarx/internal/model"
	"github.com/americooo/radarx/internal/store"
)

// testApp builds an App with a fresh SQLite store under t.TempDir(). These
// methods don't touch a.ctx, so they can be exercised directly without a
// running Wails runtime — mirrors the store test pattern in
// internal/store/sqlite_test.go.
func testApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()

	dbPath := filepath.Join(dir, "radarx.db")
	st, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	return &App{st: st}
}

func TestListTargetsEmpty(t *testing.T) {
	a := testApp(t)
	got, err := a.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no targets, got %v", got)
	}
}

func TestListTargetsNoStore(t *testing.T) {
	a := &App{}
	if _, err := a.ListTargets(); err == nil {
		t.Fatal("expected error when store is not available")
	}
}

func TestAddTargetWithoutScopeFails(t *testing.T) {
	a := testApp(t)

	err := a.AddTarget("example.com", "test", 60)
	if err == nil {
		t.Fatal("expected AddTarget to fail without any scope configured")
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("expected 'not authorized' error, got: %v", err)
	}

	targets, err := a.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(targets) != 0 {
		t.Fatalf("expected AddTarget to save nothing, got %v", targets)
	}
}

func TestAddTargetOutOfScopeFails(t *testing.T) {
	a := testApp(t)
	if err := a.st.AddScopeRoot("other.com"); err != nil {
		t.Fatal(err)
	}

	err := a.AddTarget("example.com", "test", 60)
	if err == nil {
		t.Fatal("expected AddTarget to fail for a domain outside scope")
	}
	if !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("expected 'not authorized' error, got: %v", err)
	}
}

func TestAddTargetInScopeSucceeds(t *testing.T) {
	a := testApp(t)
	if err := a.st.AddScopeRoot("example.com"); err != nil {
		t.Fatal(err)
	}

	if err := a.AddTarget("api.example.com", "Acme", 30); err != nil {
		t.Fatalf("AddTarget: %v", err)
	}

	targets, err := a.ListTargets()
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	want := model.SlugID("api.example.com")
	if targets[0].ID != want || targets[0].Root != "api.example.com" || targets[0].Label != "Acme" || targets[0].IntervalM != 30 {
		t.Fatalf("unexpected target: %+v", targets[0])
	}
}

func TestAuthorizeTargetCreatesScopeEntry(t *testing.T) {
	a := testApp(t)
	if err := a.AuthorizeTarget("newco.com"); err != nil {
		t.Fatalf("AuthorizeTarget: %v", err)
	}

	roots, err := a.st.ListScopeRoots()
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 || roots[0] != "newco.com" {
		t.Fatalf("expected scope to contain newco.com, got: %v", roots)
	}
}

func TestAuthorizeTargetAppendsToExistingScope(t *testing.T) {
	a := testApp(t)
	if err := a.st.AddScopeRoot("existing.com"); err != nil {
		t.Fatal(err)
	}

	if err := a.AuthorizeTarget("newco.com"); err != nil {
		t.Fatalf("AuthorizeTarget: %v", err)
	}

	roots, err := a.st.ListScopeRoots()
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 2 {
		t.Fatalf("expected both domains in scope, got: %v", roots)
	}
}

func TestAuthorizeTargetIsIdempotentForAlreadyAuthorizedDomain(t *testing.T) {
	a := testApp(t)
	if err := a.st.AddScopeRoot("example.com"); err != nil {
		t.Fatal(err)
	}

	if err := a.AuthorizeTarget("example.com"); err != nil {
		t.Fatalf("AuthorizeTarget: %v", err)
	}

	roots, err := a.st.ListScopeRoots()
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected no duplicate scope entry, got: %v", roots)
	}
}

func TestGetScopeRootsMissingFile(t *testing.T) {
	a := testApp(t)
	roots, err := a.GetScopeRoots()
	if err != nil {
		t.Fatalf("GetScopeRoots: %v", err)
	}
	if len(roots) != 0 {
		t.Fatalf("expected empty roots, got %v", roots)
	}
}

func TestGetScopeRootsNoStore(t *testing.T) {
	a := &App{}
	roots, err := a.GetScopeRoots()
	if err != nil {
		t.Fatalf("GetScopeRoots: %v", err)
	}
	if len(roots) != 0 {
		t.Fatalf("expected empty roots, got %v", roots)
	}
}

func TestGetScopeRootsPopulated(t *testing.T) {
	a := testApp(t)
	if err := a.st.AddScopeRoot("a.com"); err != nil {
		t.Fatal(err)
	}
	if err := a.st.AddScopeRoot("b.com"); err != nil {
		t.Fatal(err)
	}

	roots, err := a.GetScopeRoots()
	if err != nil {
		t.Fatalf("GetScopeRoots: %v", err)
	}
	if len(roots) != 2 || roots[0] != "a.com" || roots[1] != "b.com" {
		t.Fatalf("unexpected roots: %v", roots)
	}
}

func TestGetDiffNotEnoughHistory(t *testing.T) {
	a := testApp(t)

	if err := a.st.SaveTarget(model.Target{ID: "t1", Root: "t1.com", AddedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	if _, err := a.GetDiff("t1"); err == nil {
		t.Fatal("expected error for target with no scan history")
	}

	if err := a.st.SaveSnapshot(model.Snapshot{TargetID: "t1", Root: "t1.com", TakenAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GetDiff("t1"); err == nil {
		t.Fatal("expected error for target with only one snapshot")
	}
}

func TestGetDiffTwoSnapshots(t *testing.T) {
	a := testApp(t)

	if err := a.st.SaveTarget(model.Target{ID: "t1", Root: "t1.com", AddedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	base := time.Now().UTC()
	first := model.Snapshot{
		TargetID: "t1", Root: "t1.com", TakenAt: base,
		Assets: []model.Asset{{Kind: model.KindSubdomain, Key: "www.t1.com", Host: "www.t1.com"}},
	}
	if err := a.st.SaveSnapshot(first); err != nil {
		t.Fatal(err)
	}

	second := model.Snapshot{
		TargetID: "t1", Root: "t1.com", TakenAt: base.Add(time.Hour),
		Assets: []model.Asset{
			{Kind: model.KindSubdomain, Key: "www.t1.com", Host: "www.t1.com"},
			{Kind: model.KindSubdomain, Key: "admin.t1.com", Host: "admin.t1.com"},
		},
	}
	if err := a.st.SaveSnapshot(second); err != nil {
		t.Fatal(err)
	}

	d, err := a.GetDiff("t1")
	if err != nil {
		t.Fatalf("GetDiff: %v", err)
	}
	if d.NewCount() != 1 {
		t.Fatalf("expected 1 new asset, got %d (%+v)", d.NewCount(), d.Changes)
	}
}

// TestGetDiffNoChangesIsNeverNil guards against a real bug: a nil Go slice
// marshals to JSON `null`, and DiffView.tsx does `result.changes.length`
// unconditionally — a null there crashes the whole Diff tab. Two identical
// snapshots (the common "nothing changed" case) must still yield [].
func TestGetDiffNoChangesIsNeverNil(t *testing.T) {
	a := testApp(t)

	if err := a.st.SaveTarget(model.Target{ID: "t1", Root: "t1.com", AddedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	base := time.Now().UTC()
	snap := model.Snapshot{
		TargetID: "t1", Root: "t1.com", TakenAt: base,
		Assets: []model.Asset{{Kind: model.KindSubdomain, Key: "www.t1.com", Host: "www.t1.com"}},
	}
	if err := a.st.SaveSnapshot(snap); err != nil {
		t.Fatal(err)
	}
	snap.TakenAt = base.Add(time.Hour)
	if err := a.st.SaveSnapshot(snap); err != nil {
		t.Fatal(err)
	}

	d, err := a.GetDiff("t1")
	if err != nil {
		t.Fatalf("GetDiff: %v", err)
	}
	if d.Changes == nil {
		t.Fatal("Changes must never be nil — it marshals to JSON null and crashes DiffView.tsx")
	}
	if len(d.Changes) != 0 {
		t.Fatalf("expected no changes between identical snapshots, got %+v", d.Changes)
	}
}

// TestGetLatestSnapshotEmptyAssetsIsNeverNil guards the same nil-slice trap
// for ResultsView.tsx's snapshot.assets.length.
func TestGetLatestSnapshotEmptyAssetsIsNeverNil(t *testing.T) {
	a := testApp(t)

	if err := a.st.SaveTarget(model.Target{ID: "t1", Root: "t1.com", AddedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if err := a.st.SaveSnapshot(model.Snapshot{TargetID: "t1", Root: "t1.com", TakenAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	snap, err := a.GetLatestSnapshot("t1")
	if err != nil {
		t.Fatalf("GetLatestSnapshot: %v", err)
	}
	if snap.Assets == nil {
		t.Fatal("Assets must never be nil — it marshals to JSON null and crashes ResultsView.tsx")
	}
}

func TestExportReportNoSnapshot(t *testing.T) {
	a := testApp(t)
	if err := a.st.SaveTarget(model.Target{ID: "t1", Root: "t1.com", AddedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	if _, err := a.ExportReport("t1"); err == nil {
		t.Fatal("expected error when no scan has run yet")
	} else if !strings.Contains(err.Error(), "no scan yet") {
		t.Fatalf("expected 'no scan yet' error, got: %v", err)
	}
}

func TestExportReportNoStore(t *testing.T) {
	a := &App{}
	if _, err := a.ExportReport("t1"); err == nil {
		t.Fatal("expected error when store is not available")
	}
}

func TestExportReportWritesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	a := testApp(t)
	if err := a.st.SaveTarget(model.Target{ID: "t1", Root: "t1.com", Label: "Acme", AddedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	snap := model.Snapshot{
		TargetID: "t1", Root: "t1.com", TakenAt: time.Now().UTC(),
		Assets: []model.Asset{{Kind: model.KindSubdomain, Key: "www.t1.com", Host: "www.t1.com"}},
	}
	if err := a.st.SaveSnapshot(snap); err != nil {
		t.Fatal(err)
	}

	path, err := a.ExportReport("t1")
	if err != nil {
		t.Fatalf("ExportReport: %v", err)
	}

	wantDir := filepath.Join(dir, ".radarx", "reports")
	if filepath.Dir(path) != wantDir {
		t.Fatalf("expected report in %s, got %s", wantDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected report file to exist: %v", err)
	}
	if !strings.Contains(string(data), "www.t1.com") {
		t.Fatalf("expected report to mention www.t1.com, got: %s", data)
	}
}

func TestMonitoringStateTransitions(t *testing.T) {
	a := testApp(t)

	if a.IsMonitoring() {
		t.Fatal("expected monitoring to be off initially")
	}
	if err := a.StopMonitoring(); err == nil {
		t.Fatal("expected error stopping monitoring that isn't running")
	}

	if err := a.StartMonitoring(); err != nil {
		t.Fatalf("StartMonitoring: %v", err)
	}
	if !a.IsMonitoring() {
		t.Fatal("expected monitoring to be on after StartMonitoring")
	}
	if err := a.StartMonitoring(); err == nil {
		t.Fatal("expected error starting monitoring twice")
	}

	if err := a.StopMonitoring(); err != nil {
		t.Fatalf("StopMonitoring: %v", err)
	}
	if a.IsMonitoring() {
		t.Fatal("expected monitoring to be off after StopMonitoring")
	}
}

func TestStartMonitoringNoStore(t *testing.T) {
	a := &App{}
	if err := a.StartMonitoring(); err == nil {
		t.Fatal("expected error when store is not available")
	}
}

func TestSaveTelegramTokenEmpty(t *testing.T) {
	a := testApp(t)
	if err := a.SaveTelegramToken("  "); err == nil {
		t.Fatal("expected error for blank token")
	}
}

func TestSaveTelegramTokenNoStore(t *testing.T) {
	a := &App{}
	if err := a.SaveTelegramToken("tok123"); err == nil {
		t.Fatal("expected error when store is not available")
	}
}

func TestSaveTelegramTokenPersists(t *testing.T) {
	a := testApp(t)
	if err := a.SaveTelegramToken("tok123"); err != nil {
		t.Fatalf("SaveTelegramToken: %v", err)
	}

	token, ok, err := a.st.GetSetting("telegram_token")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || token != "tok123" {
		t.Fatalf("expected token to be saved, got %q ok=%v", token, ok)
	}
}

func TestSaveTelegramChatIDEmpty(t *testing.T) {
	a := testApp(t)
	if err := a.SaveTelegramChatID("  "); err == nil {
		t.Fatal("expected error for blank chat id")
	}
}

func TestSaveTelegramChatIDPersistsAlongsideToken(t *testing.T) {
	a := testApp(t)
	if err := a.SaveTelegramToken("tok123"); err != nil {
		t.Fatalf("SaveTelegramToken: %v", err)
	}
	if err := a.SaveTelegramChatID("999"); err != nil {
		t.Fatalf("SaveTelegramChatID: %v", err)
	}

	token, _, err := a.st.GetSetting("telegram_token")
	if err != nil {
		t.Fatal(err)
	}
	chatID, _, err := a.st.GetSetting("telegram_chat_id")
	if err != nil {
		t.Fatal(err)
	}
	if token != "tok123" || chatID != "999" {
		t.Fatalf("expected both token and chat id saved, got token=%q chatID=%q", token, chatID)
	}
}

func TestDetectTelegramChatIDWithoutSavedToken(t *testing.T) {
	a := testApp(t)
	if _, err := a.DetectTelegramChatID(); err == nil {
		t.Fatal("expected error when no token has been saved yet")
	}
}

func TestSendTelegramTestWithoutCredentials(t *testing.T) {
	t.Setenv("RADARX_TG_TOKEN", "")
	t.Setenv("RADARX_TG_CHAT_ID", "")

	a := testApp(t)
	if err := a.SendTelegramTest(); err == nil {
		t.Fatal("expected error when no Telegram credentials are configured")
	}
}

func TestGetTelegramStatus(t *testing.T) {
	t.Setenv("RADARX_TG_TOKEN", "")
	t.Setenv("RADARX_TG_CHAT_ID", "")
	a := testApp(t)
	if a.GetTelegramStatus() {
		t.Fatal("expected telegram status false when nothing is configured")
	}

	t.Setenv("RADARX_TG_TOKEN", "tok")
	t.Setenv("RADARX_TG_CHAT_ID", "chat")
	if !a.GetTelegramStatus() {
		t.Fatal("expected telegram status true when both env vars are set")
	}
}
