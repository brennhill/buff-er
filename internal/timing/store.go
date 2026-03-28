package timing

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

const pruneAge = 3 * 24 * time.Hour // 3-day sliding window

// Store manages timing data in SQLite.
type Store struct {
	db *sql.DB
}

// ProjectHash returns a short hash of a project path for use as a directory key.
func ProjectHash(projectPath string) string {
	h := sha256.Sum256([]byte(projectPath))
	return fmt.Sprintf("%x", h[:8])
}

// OpenStore opens or creates a SQLite timing database for a project.
func OpenStore(dataDir, projectHash string) (*Store, error) {
	dir := filepath.Join(dataDir, projectHash)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dir, "timings.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Enable WAL mode for concurrent access and set busy timeout
	// so concurrent writers wait instead of failing immediately
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set pragmas: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &Store{db: db}, nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS timings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command_pattern TEXT NOT NULL,
			started_at INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_timings_pattern ON timings(command_pattern);

		CREATE TABLE IF NOT EXISTS state (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
	`)
	return err
}

// Record stores a timing measurement.
func (s *Store) Record(pattern string, startedAt time.Time, durationMs int64) error {
	_, err := s.db.Exec(
		"INSERT INTO timings (command_pattern, started_at, duration_ms) VALUES (?, ?, ?)",
		pattern, startedAt.Unix(), durationMs,
	)
	return err
}

// TimingStats holds statistics for a command pattern.
type TimingStats struct {
	Count int
	AvgMs int64
	P75Ms int64
}

// QueryStats returns timing statistics for a command pattern within the sliding window.
func (s *Store) QueryStats(pattern string) (*TimingStats, error) {
	cutoff := time.Now().Add(-pruneAge).Unix()

	rows, err := s.db.Query(
		"SELECT duration_ms FROM timings WHERE command_pattern = ? AND started_at > ? ORDER BY duration_ms ASC",
		pattern, cutoff,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var durations []int64
	for rows.Next() {
		var d int64
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		durations = append(durations, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(durations) == 0 {
		return &TimingStats{Count: 0}, nil
	}

	var sum int64
	for _, d := range durations {
		sum += d
	}

	// Nearest-rank P75: ceil(0.75 * n) - 1
	p75Idx := (len(durations)*75 + 99) / 100
	if p75Idx > 0 {
		p75Idx--
	}
	if p75Idx >= len(durations) {
		p75Idx = len(durations) - 1
	}

	return &TimingStats{
		Count: len(durations),
		AvgMs: sum / int64(len(durations)),
		P75Ms: durations[p75Idx],
	}, nil
}

// Prune removes records older than the sliding window.
func (s *Store) Prune() error {
	cutoff := time.Now().Add(-pruneAge).Unix()
	_, err := s.db.Exec("DELETE FROM timings WHERE started_at < ?", cutoff)
	return err
}

// SetState stores a key-value pair in the state table.
func (s *Store) SetState(key, value string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO state (key, value) VALUES (?, ?)",
		key, value,
	)
	return err
}

// GetState retrieves a value from the state table.
func (s *Store) GetState(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM state WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// State key prefixes — all state keys must use one of these.
const (
	StateKeyLastSuggestion  = "last_suggestion"
	StateKeyLastPrune       = "last_prune"
	StateKeySessionPrefix   = "session_start_"
	StateKeyTodayStreak     = "today_streak"
	StateKeyStreakDate      = "streak_date"
	StateKeyPendingFollowUp = "pending_followup"
)

// IncrementStreak bumps today's exercise streak counter. Resets if the date changed.
func (s *Store) IncrementStreak() (int, error) {
	today := time.Now().Format("2006-01-02")
	lastDate, _ := s.GetState(StateKeyStreakDate)

	streak := 0
	if lastDate == today {
		streakStr, _ := s.GetState(StateKeyTodayStreak)
		if n, err := strconv.Atoi(streakStr); err == nil {
			streak = n
		}
	}
	streak++

	if err := s.SetState(StateKeyStreakDate, today); err != nil {
		return streak, err
	}
	return streak, s.SetState(StateKeyTodayStreak, strconv.Itoa(streak))
}

// GetStreak returns today's exercise streak count.
func (s *Store) GetStreak() int {
	today := time.Now().Format("2006-01-02")
	lastDate, _ := s.GetState(StateKeyStreakDate)
	if lastDate != today {
		return 0
	}
	streakStr, _ := s.GetState(StateKeyTodayStreak)
	n, _ := strconv.Atoi(streakStr)
	return n
}

// PruneState removes session_start_ keys whose stored timestamp is older than the prune window,
// preserving active sessions.
func (s *Store) PruneState() error {
	cutoff := time.Now().Add(-pruneAge)

	rows, err := s.db.Query("SELECT key, value FROM state WHERE key LIKE ?", StateKeySessionPrefix+"%")
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	var keysToDelete []string
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			keysToDelete = append(keysToDelete, key)
			continue
		}
		if t.Before(cutoff) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, key := range keysToDelete {
		if _, err := s.db.Exec("DELETE FROM state WHERE key = ?", key); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
