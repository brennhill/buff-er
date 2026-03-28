package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/brennhill/buff-er/internal/config"
	"github.com/brennhill/buff-er/internal/exercise"
	"github.com/brennhill/buff-er/internal/hook"
	"github.com/brennhill/buff-er/internal/timing"
)

func TestPreToolUseLearningPhase(t *testing.T) {
	input := hook.PreToolUseInput{
		Input: hook.Input{
			SessionID:     "test-session",
			CWD:           t.TempDir(),
			HookEventName: "PreToolUse",
			ToolName:      "Bash",
			ToolUseID:     "tu-001",
		},
		ToolInput: hook.BashToolInput{Command: "cargo build"},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := hook.ParsePreToolUse(data)
	if err != nil {
		t.Fatal(err)
	}

	if parsed.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash", parsed.ToolName)
	}

	projectHash := timing.ProjectHash(parsed.CWD)
	store, err := timing.OpenStore(t.TempDir(), projectHash)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	pattern := timing.ExtractPattern(parsed.ToolInput.Command)
	est, err := timing.EstimateDuration(store, pattern)
	if err != nil {
		t.Fatal(err)
	}

	if est.Confident {
		t.Error("should not be confident with no data")
	}
}

func TestPreToolUseWithEnoughData(t *testing.T) {
	dataDir := t.TempDir()
	cwd := "/tmp/test-project-integration"
	projectHash := timing.ProjectHash(cwd)

	store, err := timing.OpenStore(dataDir, projectHash)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	if err := store.Record("cargo build", now.Add(-2*time.Hour), 300000); err != nil {
		t.Fatal(err)
	}
	if err := store.Record("cargo build", now.Add(-1*time.Hour), 320000); err != nil {
		t.Fatal(err)
	}
	if err := store.Record("cargo build", now.Add(-30*time.Minute), 280000); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store2, err := timing.OpenStore(dataDir, projectHash)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store2.Close() }()

	est, err := timing.EstimateDuration(store2, "cargo build")
	if err != nil {
		t.Fatal(err)
	}

	if !est.Confident {
		t.Error("should be confident with 3 samples")
	}

	if est.P75Minutes < 3.0 {
		t.Errorf("P75Minutes = %.1f, want >= 3.0", est.P75Minutes)
	}

	cfg := config.DefaultConfig()
	catalog := config.GetExerciseCatalog(cfg)
	ex := exercise.Suggest(catalog, est.P75Minutes)
	if ex == nil {
		t.Error("should suggest an exercise")
	}
}

func TestNonBashToolIgnored(t *testing.T) {
	input := hook.PreToolUseInput{
		Input: hook.Input{
			ToolName: "Read",
		},
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := hook.ParsePreToolUse(data)
	if err != nil {
		t.Fatal(err)
	}

	if parsed.ToolName == "Bash" {
		t.Error("Read tool should not be treated as Bash")
	}
}

func TestPostToolUseRecordsTiming(t *testing.T) {
	dataDir := t.TempDir()
	sessionID := "test-post-session"
	cwd := "/tmp/test-post-project"
	projectHash := timing.ProjectHash(cwd)

	pending := timing.NewPendingStore(sessionID)
	if err := pending.Set("tu-post-001", timing.PendingEntry{
		StartTime:         time.Now().Add(-5 * time.Minute),
		CommandPattern:    "cargo build",
		ExerciseSuggested: false,
	}); err != nil {
		t.Fatal(err)
	}

	entry, err := pending.Get("tu-post-001")
	if err != nil {
		t.Fatal(err)
	}
	if entry == nil {
		t.Fatal("pending entry should exist")
	}

	durationMs := time.Since(entry.StartTime).Milliseconds()

	store, err := timing.OpenStore(dataDir, projectHash)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	if err := store.Record(entry.CommandPattern, entry.StartTime, durationMs); err != nil {
		t.Fatal(err)
	}

	stats, err := store.QueryStats("cargo build")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Count != 1 {
		t.Errorf("Count = %d, want 1", stats.Count)
	}

	if stats.AvgMs < 290000 || stats.AvgMs > 310000 {
		t.Errorf("AvgMs = %d, expected ~300000", stats.AvgMs)
	}
}

func init() {
	_ = os.Setenv("XDG_CONFIG_HOME", os.TempDir())
}
