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

	scope := Scope{W: 3, H: 3, NActive: 3, GenCap: 50, TaskID: 0}

	// First champion inserts directly with new best outcome
	if res := store.Submit(Candidate{Scope: scope, GenCount: 10, Program: "AA"}); res.Outcome != OutcomeInsertedNewBest {
		t.Fatalf("expected first insert as new best, got %+v", res)
	}

	// Better champion inserts with new best outcome
	if res := store.Submit(Candidate{Scope: scope, GenCount: 12, Program: "BB"}); res.Outcome != OutcomeInsertedNewBest {
		t.Fatalf("expected insertion as new best, got %+v", res)
	}

	// Same generation as best - should be inserted as tie
	tieOne := Candidate{Scope: scope, GenCount: 12, Program: "CC"}
	if res := store.Submit(tieOne); res.Outcome != OutcomeInsertedTie {
		t.Fatalf("expected insert tie, got %+v", res)
	}

	// Duplicate program should be rejected
	if res := store.Submit(tieOne); res.Outcome != OutcomeDuplicate {
		t.Fatalf("expected duplicate detection, got %+v", res)
	}

	// Another unique submission
	if res := store.Submit(Candidate{Scope: scope, GenCount: 12, Program: "CD"}); res.Outcome != OutcomeInsertedTie {
		t.Fatalf("expected insert, got %+v", res)
	}

	// Another unique submission
	if res := store.Submit(Candidate{Scope: scope, GenCount: 12, Program: "CE"}); res.Outcome != OutcomeInsertedTie {
		t.Fatalf("expected insert, got %+v", res)
	}

	// Lower generation should be ignored
	if res := store.Submit(Candidate{Scope: scope, GenCount: 5, Program: "FF"}); res.Outcome != OutcomeInsertedTie {
		t.Fatalf("expected insert (under capacity), got %+v", res)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	defer db.Close()

	// Should have 6 rows in the database
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM champions`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 6 {
		t.Fatalf("expected 6 rows, got %d", count)
	}
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
