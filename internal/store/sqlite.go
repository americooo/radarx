// SQLiteStore is the second Store implementation, backed by a local SQLite
// database file (via the pure-Go, CGo-free modernc.org/sqlite driver so
// cross-compilation for Windows/Linux/macOS stays simple). JSONStore remains
// the on-disk fallback; this file only adds a second implementation of the
// same Store contract.
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"github.com/americooo/radarx/internal/model"
)

// SQLiteStore persists targets and snapshots in a single SQLite database
// file. It implements the same Store interface as JSONStore.
type SQLiteStore struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS targets (
	id             TEXT PRIMARY KEY,
	root           TEXT NOT NULL,
	label          TEXT NOT NULL,
	enabled        INTEGER NOT NULL,
	interval_m     INTEGER NOT NULL,
	wordlist_json  TEXT NOT NULL,
	added_at       TEXT NOT NULL,
	last_scan_at   TEXT
);

CREATE TABLE IF NOT EXISTS scans (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	target_id TEXT NOT NULL,
	taken_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_scans_target_taken ON scans(target_id, taken_at);

CREATE TABLE IF NOT EXISTS assets (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	scan_id     INTEGER NOT NULL REFERENCES scans(id),
	kind        TEXT NOT NULL,
	key         TEXT NOT NULL,
	host        TEXT NOT NULL,
	ip_json     TEXT NOT NULL,
	port        INTEGER NOT NULL,
	scheme      TEXT NOT NULL,
	status_code INTEGER NOT NULL,
	title       TEXT NOT NULL,
	server      TEXT NOT NULL,
	cert_cn     TEXT NOT NULL,
	cert_expiry TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assets_scan ON assets(scan_id);
`

// NewSQLiteStore opens (creating if needed) a SQLite database at path,
// applies the schema, and — if the database is brand new and a legacy
// JSONStore layout exists alongside it — migrates that data in once.
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// SQLite handles one writer at a time; a single connection avoids
	// "database is locked" errors under concurrent access from this process.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	s := &SQLiteStore{db: db}

	if err := s.migrateFromJSONIfEmpty(filepath.Dir(path)); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// Close releases the underlying database handle.
func (s *SQLiteStore) Close() error { return s.db.Close() }

// migrateFromJSONIfEmpty performs a one-time import of legacy JSONStore data
// (<root>/targets/*.json, <root>/snapshots/) into this SQLite database, but
// only when the SQLite targets table is currently empty and a legacy layout
// is present. If either condition is not met, it silently does nothing.
func (s *SQLiteStore) migrateFromJSONIfEmpty(root string) error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM targets`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	targetsDir := filepath.Join(root, "targets")
	if _, err := os.Stat(targetsDir); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	jsonStore, err := NewJSONStore(root)
	if err != nil {
		return err
	}

	targets, err := jsonStore.ListTargets()
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return nil
	}

	for _, t := range targets {
		if err := s.SaveTarget(t); err != nil {
			return fmt.Errorf("migrate target %s: %w", t.ID, err)
		}
		snaps, err := jsonStore.ListSnapshots(t.ID)
		if err != nil {
			return fmt.Errorf("migrate snapshots for %s: %w", t.ID, err)
		}
		for _, snap := range snaps {
			if err := s.SaveSnapshot(snap); err != nil {
				return fmt.Errorf("migrate snapshot for %s: %w", t.ID, err)
			}
		}
	}
	return nil
}

func (s *SQLiteStore) SaveTarget(t model.Target) error {
	wordlist, err := json.Marshal(t.Wordlist)
	if err != nil {
		return err
	}
	var lastScanAt any
	if !t.LastScanAt.IsZero() {
		lastScanAt = t.LastScanAt.UTC().Format(time.RFC3339Nano)
	}
	_, err = s.db.Exec(`
		INSERT INTO targets (id, root, label, enabled, interval_m, wordlist_json, added_at, last_scan_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			root = excluded.root,
			label = excluded.label,
			enabled = excluded.enabled,
			interval_m = excluded.interval_m,
			wordlist_json = excluded.wordlist_json,
			added_at = excluded.added_at,
			last_scan_at = excluded.last_scan_at
	`, t.ID, t.Root, t.Label, boolToInt(t.Enabled), t.IntervalM, string(wordlist),
		t.AddedAt.UTC().Format(time.RFC3339Nano), lastScanAt)
	return err
}

func (s *SQLiteStore) ListTargets() ([]model.Target, error) {
	rows, err := s.db.Query(`
		SELECT id, root, label, enabled, interval_m, wordlist_json, added_at, last_scan_at
		FROM targets ORDER BY root ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Target
	for rows.Next() {
		t, err := scanTarget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) SaveSnapshot(snap model.Snapshot) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO scans (target_id, taken_at) VALUES (?, ?)`,
		snap.TargetID, snap.TakenAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	scanID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		INSERT INTO assets (scan_id, kind, key, host, ip_json, port, scheme, status_code, title, server, cert_cn, cert_expiry)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, a := range snap.Assets {
		ipJSON, err := json.Marshal(a.IP)
		if err != nil {
			return err
		}
		if _, err := stmt.Exec(scanID, string(a.Kind), a.Key, a.Host, string(ipJSON), a.Port,
			a.Scheme, a.StatusCode, a.Title, a.Server, a.CertCN, a.CertExpiry); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// LatestSnapshot returns the most recent snapshot for a target. The bool is
// false (with no error) when the target has never been scanned.
func (s *SQLiteStore) LatestSnapshot(targetID string) (model.Snapshot, bool, error) {
	var scanID int64
	var takenAtStr string
	err := s.db.QueryRow(`
		SELECT id, taken_at FROM scans WHERE target_id = ? ORDER BY taken_at DESC LIMIT 1
	`, targetID).Scan(&scanID, &takenAtStr)
	if err == sql.ErrNoRows {
		return model.Snapshot{}, false, nil
	}
	if err != nil {
		return model.Snapshot{}, false, err
	}

	snap, err := s.buildSnapshot(targetID, scanID, takenAtStr)
	if err != nil {
		return model.Snapshot{}, false, err
	}
	return snap, true, nil
}

// ListSnapshots returns all snapshots for a target, oldest first.
func (s *SQLiteStore) ListSnapshots(targetID string) ([]model.Snapshot, error) {
	rows, err := s.db.Query(`
		SELECT id, taken_at FROM scans WHERE target_id = ? ORDER BY taken_at ASC
	`, targetID)
	if err != nil {
		return nil, err
	}
	type scanRow struct {
		id      int64
		takenAt string
	}
	var scans []scanRow
	for rows.Next() {
		var r scanRow
		if err := rows.Scan(&r.id, &r.takenAt); err != nil {
			rows.Close()
			return nil, err
		}
		scans = append(scans, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	out := make([]model.Snapshot, 0, len(scans))
	for _, r := range scans {
		snap, err := s.buildSnapshot(targetID, r.id, r.takenAt)
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, nil
}

// buildSnapshot loads a target's root and a scan's assets into a Snapshot.
func (s *SQLiteStore) buildSnapshot(targetID string, scanID int64, takenAtStr string) (model.Snapshot, error) {
	takenAt, err := time.Parse(time.RFC3339Nano, takenAtStr)
	if err != nil {
		return model.Snapshot{}, err
	}

	var root string
	if err := s.db.QueryRow(`SELECT root FROM targets WHERE id = ?`, targetID).Scan(&root); err != nil && err != sql.ErrNoRows {
		return model.Snapshot{}, err
	}

	rows, err := s.db.Query(`
		SELECT kind, key, host, ip_json, port, scheme, status_code, title, server, cert_cn, cert_expiry
		FROM assets WHERE scan_id = ? ORDER BY id ASC
	`, scanID)
	if err != nil {
		return model.Snapshot{}, err
	}
	defer rows.Close()

	var assets []model.Asset
	for rows.Next() {
		var a model.Asset
		var kind, ipJSON string
		if err := rows.Scan(&kind, &a.Key, &a.Host, &ipJSON, &a.Port, &a.Scheme,
			&a.StatusCode, &a.Title, &a.Server, &a.CertCN, &a.CertExpiry); err != nil {
			return model.Snapshot{}, err
		}
		a.Kind = model.AssetKind(kind)
		if ipJSON != "" {
			if err := json.Unmarshal([]byte(ipJSON), &a.IP); err != nil {
				return model.Snapshot{}, err
			}
		}
		assets = append(assets, a)
	}
	if err := rows.Err(); err != nil {
		return model.Snapshot{}, err
	}

	return model.Snapshot{
		TargetID: targetID,
		Root:     root,
		TakenAt:  takenAt,
		Assets:   assets,
	}, nil
}

// rowScanner abstracts *sql.Rows so scanTarget can be reused for a single row.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTarget(rs rowScanner) (model.Target, error) {
	var t model.Target
	var enabled int
	var wordlistJSON string
	var addedAtStr string
	var lastScanAtStr sql.NullString

	if err := rs.Scan(&t.ID, &t.Root, &t.Label, &enabled, &t.IntervalM, &wordlistJSON, &addedAtStr, &lastScanAtStr); err != nil {
		return model.Target{}, err
	}

	t.Enabled = enabled != 0
	if wordlistJSON != "" {
		if err := json.Unmarshal([]byte(wordlistJSON), &t.Wordlist); err != nil {
			return model.Target{}, err
		}
	}
	addedAt, err := time.Parse(time.RFC3339Nano, addedAtStr)
	if err != nil {
		return model.Target{}, err
	}
	t.AddedAt = addedAt
	if lastScanAtStr.Valid && lastScanAtStr.String != "" {
		lastScanAt, err := time.Parse(time.RFC3339Nano, lastScanAtStr.String)
		if err != nil {
			return model.Target{}, err
		}
		t.LastScanAt = lastScanAt
	}
	return t, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
