package timing

import (
	"testing"
	"time"
)

func TestStoreRecordAndQuery(t *testing.T) {
	store, err := OpenStore(t.TempDir(), "test-project")
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now()

	// Record 3 timings for "cargo build"
	for i, dur := range []int64{180000, 200000, 220000} { // 3m, 3m20s, 3m40s
		err := store.Record("cargo build", now.Add(time.Duration(-i)*time.Hour), dur)
		if err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	stats, err := store.QueryStats("cargo build")
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}

	if stats.Count != 3 {
		t.Errorf("Count = %d, want 3", stats.Count)
	}

	// Average should be 200000ms
	if stats.AvgMs != 200000 {
		t.Errorf("AvgMs = %d, want 200000", stats.AvgMs)
	}

	// P75 should be 220000 (index 2 of [180000, 200000, 220000])
	if stats.P75Ms != 220000 {
		t.Errorf("P75Ms = %d, want 220000", stats.P75Ms)
	}
}

func TestStoreQueryNoData(t *testing.T) {
	store, err := OpenStore(t.TempDir(), "test-project")
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	stats, err := store.QueryStats("nonexistent")
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}

	if stats.Count != 0 {
		t.Errorf("Count = %d, want 0", stats.Count)
	}
}

func TestStorePrune(t *testing.T) {
	store, err := OpenStore(t.TempDir(), "test-project")
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Record one old and one new timing
	old := time.Now().Add(-4 * 24 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)

	if err := store.Record("cargo build", old, 180000); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := store.Record("cargo build", recent, 200000); err != nil {
		t.Fatalf("Record: %v", err)
	}

	if err := store.Prune(); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	stats, err := store.QueryStats("cargo build")
	if err != nil {
		t.Fatalf("QueryStats: %v", err)
	}

	if stats.Count != 1 {
		t.Errorf("Count after prune = %d, want 1", stats.Count)
	}
}

func TestStoreState(t *testing.T) {
	store, err := OpenStore(t.TempDir(), "test-project")
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Get non-existent key
	val, err := store.GetState("last_suggestion")
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if val != "" {
		t.Errorf("GetState empty = %q, want empty", val)
	}

	// Set and get
	if err := store.SetState("last_suggestion", "2024-01-01T00:00:00Z"); err != nil {
		t.Fatalf("SetState: %v", err)
	}

	val, err = store.GetState("last_suggestion")
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if val != "2024-01-01T00:00:00Z" {
		t.Errorf("GetState = %q, want 2024-01-01T00:00:00Z", val)
	}
}

func TestStoreWALModeAndBusyTimeout(t *testing.T) {
	store, err := OpenStore(t.TempDir(), "test-wal")
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	var journalMode string
	if err := store.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}

	var busyTimeout int
	if err := store.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busyTimeout)
	}
}

func TestPruneStateOnlyRemovesOldSessions(t *testing.T) {
	store, err := OpenStore(t.TempDir(), "test-project")
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Set a stale session (4 days old) and a fresh one (now)
	staleTime := time.Now().Add(-4 * 24 * time.Hour).Format(time.RFC3339)
	freshTime := time.Now().Format(time.RFC3339)

	if err := store.SetState("session_start_stale-session", staleTime); err != nil {
		t.Fatal(err)
	}
	if err := store.SetState("session_start_active-session", freshTime); err != nil {
		t.Fatal(err)
	}

	if err := store.PruneState(); err != nil {
		t.Fatal(err)
	}

	// Stale session should be gone
	val, err := store.GetState("session_start_stale-session")
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Errorf("stale session should have been pruned, got %q", val)
	}

	// Active session should remain
	val, err = store.GetState("session_start_active-session")
	if err != nil {
		t.Fatal(err)
	}
	if val == "" {
		t.Error("active session should NOT have been pruned")
	}
}
