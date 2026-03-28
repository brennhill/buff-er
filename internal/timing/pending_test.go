package timing

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPendingSetAndGet(t *testing.T) {
	ps := NewPendingStore("test-session-1")

	entry := PendingEntry{
		StartTime:         time.Now(),
		CommandPattern:    "cargo build",
		ExerciseSuggested: true,
	}

	if err := ps.Set("tu-001", entry); err != nil {
		t.Fatal(err)
	}

	got, err := ps.Get("tu-001")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if got.CommandPattern != "cargo build" {
		t.Errorf("CommandPattern = %q, want cargo build", got.CommandPattern)
	}
	if !got.ExerciseSuggested {
		t.Error("ExerciseSuggested should be true")
	}

	// Second get should return nil (file removed)
	got, err = ps.Get("tu-001")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("second Get should return nil")
	}
}

func TestPendingGetNonexistent(t *testing.T) {
	ps := NewPendingStore("test-session-2")

	got, err := ps.Get("does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent entry")
	}
}

func TestCleanupStale(t *testing.T) {
	// Create a fake session dir with an old file and a fresh file
	sessionDir := filepath.Join(os.TempDir(), "buff-er-test-cleanup-session")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(sessionDir) }()

	oldFile := filepath.Join(sessionDir, "old-entry.json")
	freshFile := filepath.Join(sessionDir, "fresh-entry.json")

	if err := os.WriteFile(oldFile, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(freshFile, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Make the old file look old (2 hours ago)
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	CleanupStale()

	// Old file should be gone
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old file should have been cleaned up")
	}

	// Fresh file should remain
	if _, err := os.Stat(freshFile); err != nil {
		t.Error("fresh file should still exist")
	}
}
