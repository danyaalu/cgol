package storage

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

// Scope identifies a category of champions.
type Scope struct {
	W       int
	H       int
	NActive int
	GenCap  int
	TaskID  int // Task index from tasks.json
}

// Candidate represents a single candidate result.
type Candidate struct {
	Scope    Scope
	GenCount int
	Program  string
}

// SubmissionOutcome describes what happened to a submission.
type SubmissionOutcome int

const (
	OutcomeInsertedNewBest SubmissionOutcome = iota
	OutcomeInsertedTie
	OutcomeCachedTie
	OutcomeDuplicate
	OutcomeIgnoredLower
)

// SubmitResult captures the result of a submission.
type SubmitResult struct {
	Outcome SubmissionOutcome
	BestGen int
	Err     error
}

type submitRequest struct {
	candidate Candidate
	resp      chan SubmitResult
}

type flushRequest struct {
	resp chan error
}

type stopRequest struct {
	resp chan error
}

const noBest = -1

// ChampionStore coordinates champion writes into SQLite via a single writer goroutine.
type ChampionStore struct {
	db        *sql.DB
	cacheSize int

	best map[Scope]int

	insertStmt *sql.Stmt
	deleteStmt *sql.Stmt

	submitCh chan submitRequest
	flushCh  chan flushRequest
	stopCh   chan stopRequest
	doneCh   chan struct{}
	wg       sync.WaitGroup
}

// NewChampionStore configures the SQLite database and starts the writer.
// Uses the pure Go SQLite driver from modernc.org/sqlite.
func NewChampionStore(dbPath string, cacheSize int) (*ChampionStore, error) {
	if cacheSize < 1 {
		cacheSize = 1
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := configureSQLite(db); err != nil {
		db.Close()
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	insertStmt, err := db.Prepare(`INSERT OR IGNORE INTO champions
		(w, h, n_active, gen_cap, task_id, gen_count, program)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("prepare insert: %w", err)
	}

	deleteStmt, err := db.Prepare(`DELETE FROM champions WHERE id IN (
		SELECT id FROM champions WHERE task_id = ? ORDER BY gen_count ASC LIMIT 1
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("prepare delete: %w", err)
	}

	best, err := loadBestMap(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	store := &ChampionStore{
		db:         db,
		cacheSize:  cacheSize,
		best:       best,
		insertStmt: insertStmt,
		deleteStmt: deleteStmt,
		submitCh:   make(chan submitRequest, 128),
		flushCh:    make(chan flushRequest),
		stopCh:     make(chan stopRequest),
		doneCh:     make(chan struct{}),
	}

	store.wg.Add(1)
	go store.loop()

	return store, nil
}

// Submit enqueues a candidate for consideration.
func (s *ChampionStore) Submit(candidate Candidate) SubmitResult {
	resp := make(chan SubmitResult, 1)
	select {
	case <-s.doneCh:
		return SubmitResult{Err: errors.New("champion store closed")}
	default:
	}
	s.submitCh <- submitRequest{candidate: candidate, resp: resp}
	return <-resp
}

// Flush forces pending cached ties to be written.
func (s *ChampionStore) Flush() error {
	resp := make(chan error, 1)
	select {
	case <-s.doneCh:
		return errors.New("champion store closed")
	default:
	}
	s.flushCh <- flushRequest{resp: resp}
	return <-resp
}

// Close flushes pending writes and stops the writer goroutine.
func (s *ChampionStore) Close() error {
	resp := make(chan error, 1)
	select {
	case <-s.doneCh:
		return nil
	default:
	}
	s.stopCh <- stopRequest{resp: resp}
	err := <-resp
	s.wg.Wait()
	dbErr := s.db.Close()
	return errors.Join(err, dbErr)
}

func (s *ChampionStore) loop() {
	defer s.wg.Done()
	defer close(s.doneCh)
	for {
		select {
		case req := <-s.submitCh:
			req.resp <- s.handleSubmit(req.candidate)
		case req := <-s.flushCh:
			req.resp <- s.flushAll()
		case req := <-s.stopCh:
			req.resp <- s.flushAll()
			return
		}
	}
}

func (s *ChampionStore) handleSubmit(candidate Candidate) SubmitResult {
	if len(candidate.Program) == 0 {
		return SubmitResult{Err: errors.New("program cannot be empty")}
	}
	scope := candidate.Scope

	// Check if this program already exists for this task (duplicate detection)
	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM champions WHERE program = ? AND task_id = ? LIMIT 1`,
		candidate.Program, scope.TaskID).Scan(&exists)
	if err == nil {
		// Program already exists for this task
		return SubmitResult{Outcome: OutcomeDuplicate, BestGen: candidate.GenCount}
	}
	if err != sql.ErrNoRows {
		return SubmitResult{Err: fmt.Errorf("duplicate check failed: %w", err)}
	}

	// Count how many programs we have for this task
	var count int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM champions WHERE task_id = ?`, scope.TaskID).Scan(&count)
	if err != nil {
		return SubmitResult{Err: fmt.Errorf("count check failed: %w", err)}
	}

	best, ok := s.best[scope]
	if !ok {
		best = s.lookupBest(scope)
	}

	const maxProgramsPerTask = 5000

	if count < maxProgramsPerTask {
		// We have room, just insert
		if err := s.insertCandidate(scope, candidate.GenCount, candidate.Program); err != nil {
			return SubmitResult{Err: err}
		}
		if candidate.GenCount > best {
			s.best[scope] = candidate.GenCount
			return SubmitResult{Outcome: OutcomeInsertedNewBest, BestGen: candidate.GenCount}
		}
		return SubmitResult{Outcome: OutcomeInsertedTie, BestGen: best}
	}

	// We're at capacity, check if this candidate is better than the worst one
	var worstGen int
	err = s.db.QueryRow(`SELECT gen_count FROM champions WHERE task_id = ? ORDER BY gen_count ASC LIMIT 1`,
		scope.TaskID).Scan(&worstGen)
	if err != nil {
		return SubmitResult{Err: fmt.Errorf("worst lookup failed: %w", err)}
	}

	if candidate.GenCount <= worstGen {
		// This candidate is not better than the worst, ignore it
		return SubmitResult{Outcome: OutcomeIgnoredLower, BestGen: best}
	}

	// This candidate is better than the worst, replace the worst one
	if err := s.replaceWorst(scope, candidate.GenCount, candidate.Program); err != nil {
		return SubmitResult{Err: err}
	}

	if candidate.GenCount > best {
		s.best[scope] = candidate.GenCount
		return SubmitResult{Outcome: OutcomeInsertedNewBest, BestGen: candidate.GenCount}
	}
	return SubmitResult{Outcome: OutcomeInsertedTie, BestGen: best}
}

func (s *ChampionStore) insertCandidate(scope Scope, genCount int, program string) error {
	_, err := s.insertStmt.Exec(scope.W, scope.H, scope.NActive, scope.GenCap, scope.TaskID, genCount, program)
	if err != nil {
		return fmt.Errorf("insert champion: %w", err)
	}
	return nil
}

func (s *ChampionStore) replaceScope(scope Scope, genCount int, program string) error {
	// This method is deprecated - use replaceWorst instead
	return s.replaceWorst(scope, genCount, program)
}

func (s *ChampionStore) replaceWorst(scope Scope, genCount int, program string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin replace tx: %w", err)
	}

	delStmt := tx.Stmt(s.deleteStmt)
	if _, err := delStmt.Exec(scope.TaskID); err != nil {
		tx.Rollback()
		return fmt.Errorf("delete worst champion: %w", err)
	}

	insStmt := tx.Stmt(s.insertStmt)
	if _, err := insStmt.Exec(scope.W, scope.H, scope.NActive, scope.GenCap, scope.TaskID, genCount, program); err != nil {
		tx.Rollback()
		return fmt.Errorf("insert new champion: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace: %w", err)
	}
	return nil
}

func (s *ChampionStore) flushScope(scope Scope) error {
	return nil
}

func (s *ChampionStore) flushAll() error {
	return nil
}

func (s *ChampionStore) lookupBest(scope Scope) int {
	var best sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(gen_count) FROM champions WHERE task_id = ?`,
		scope.TaskID).Scan(&best)
	if err != nil {
		s.best[scope] = noBest
		return noBest
	}
	if best.Valid {
		s.best[scope] = int(best.Int64)
		return int(best.Int64)
	}
	s.best[scope] = noBest
	return noBest
}

func configureSQLite(db *sql.DB) error {
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA synchronous=NORMAL`); err != nil {
		return fmt.Errorf("set synchronous: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec(`PRAGMA temp_store=MEMORY`); err != nil {
		return fmt.Errorf("set temp_store: %w", err)
	}
	if _, err := db.Exec(`PRAGMA cache_size=-20000`); err != nil {
		return fmt.Errorf("set cache_size: %w", err)
	}
	return nil
}

func migrate(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS champions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	w INTEGER NOT NULL,
	h INTEGER NOT NULL,
	n_active INTEGER NOT NULL,
	gen_cap INTEGER NOT NULL,
	task_id INTEGER NOT NULL,
	gen_count INTEGER NOT NULL,
	program TEXT NOT NULL,
	UNIQUE(program, task_id)
);
CREATE INDEX IF NOT EXISTS idx_champions_task_gen ON champions (task_id, gen_count DESC);
CREATE INDEX IF NOT EXISTS idx_champions_scope ON champions (w, h, n_active, gen_cap, task_id);`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func loadBestMap(db *sql.DB) (map[Scope]int, error) {
	rows, err := db.Query(`SELECT w, h, n_active, gen_cap, task_id, MAX(gen_count) AS best
		FROM champions
		GROUP BY task_id`)
	if err != nil {
		return nil, fmt.Errorf("load best map: %w", err)
	}
	defer rows.Close()

	result := make(map[Scope]int)
	for rows.Next() {
		var scope Scope
		var best sql.NullInt64
		if err := rows.Scan(&scope.W, &scope.H, &scope.NActive, &scope.GenCap, &scope.TaskID, &best); err != nil {
			return nil, fmt.Errorf("scan best map: %w", err)
		}
		if best.Valid {
			result[scope] = int(best.Int64)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate best map: %w", err)
	}

	return result, nil
}

// DecodeHexProgram converts a hex nibble string into packed bytes.
// Pads with a trailing zero nibble when needed to complete the final byte.
func DecodeHexProgram(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		hexStr += "0"
	}
	return hex.DecodeString(hexStr)
}
