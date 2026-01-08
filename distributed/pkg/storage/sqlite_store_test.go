package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestChampionStoreLifecycle(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "champions.db")

	store, err := NewChampionStore(dbPath, 2)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	scope := Scope{W: 3, H: 3, NActive: 3, GenCap: 50}

	// First champion inserts directly.
	if res := store.Submit(Candidate{Scope: scope, GenCount: 10, Program: []byte{0xAA}}); res.Outcome != OutcomeInsertedNewBest {
		t.Fatalf("expected first insert as new best, got %+v", res)
	}
	assertRowCount(t, store.db, 1)

	// New best replaces previous rows.
	if res := store.Submit(Candidate{Scope: scope, GenCount: 12, Program: []byte{0xBB}}); res.Outcome != OutcomeInsertedNewBest {
		t.Fatalf("expected replacement as new best, got %+v", res)
	}
	assertRowCount(t, store.db, 1)

	// Tie caches until threshold.
	tieOne := Candidate{Scope: scope, GenCount: 12, Program: []byte{0xCC}}
	if res := store.Submit(tieOne); res.Outcome != OutcomeCachedTie {
		t.Fatalf("expected cached tie, got %+v", res)
	}
	// Duplicate in cache ignored.
	if res := store.Submit(tieOne); res.Outcome != OutcomeDuplicate {
		t.Fatalf("expected duplicate detection, got %+v", res)
	}
	assertRowCount(t, store.db, 1)

	// Second unique tie triggers flush.
	if res := store.Submit(Candidate{Scope: scope, GenCount: 12, Program: []byte{0xCD}}); res.Outcome != OutcomeInsertedTie {
		t.Fatalf("expected flushed tie insert, got %+v", res)
	}
	assertRowCount(t, store.db, 3)

	// Pending cached tie flushes on close.
	if res := store.Submit(Candidate{Scope: scope, GenCount: 12, Program: []byte{0xCE}}); res.Outcome != OutcomeCachedTie {
		t.Fatalf("expected cached tie before shutdown, got %+v", res)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer db.Close()

	assertRowCount(t, db, 4)
}

func assertRowCount(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM champions`).Scan(&got); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %d rows, got %d", expected, got)
	}
}
