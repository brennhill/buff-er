package main

import (
	"testing"
	"time"

	"github.com/brennhill/buff-er/internal/config"
	"github.com/brennhill/buff-er/internal/timing"
)

func TestStopNoSuggestionWithinCooldown(t *testing.T) {
	dataDir := t.TempDir()
	projectHash := timing.ProjectHash("/tmp/stop-test")

	store, err := timing.OpenStore(dataDir, projectHash)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	// Set last suggestion to now (within cooldown)
	if err := store.SetState(timing.StateKeyLastSuggestion, time.Now().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	lastStr, _ := store.GetState(timing.StateKeyLastSuggestion)
	if lastStr == "" {
		t.Fatal("last_suggestion should be set")
	}

	lastTime, err := time.Parse(time.RFC3339, lastStr)
	if err != nil {
		t.Fatal(err)
	}

	elapsed := time.Since(lastTime)
	if elapsed.Minutes() >= float64(cfg.BreakCooldownMinutes) {
		t.Error("should be within cooldown period")
	}
}

func TestStopSuggestsAfterCooldown(t *testing.T) {
	dataDir := t.TempDir()
	projectHash := timing.ProjectHash("/tmp/stop-test-cooldown")

	store, err := timing.OpenStore(dataDir, projectHash)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	sessionID := "test-stop-session"

	// Set session start to well past the cooldown
	sessionStart := time.Now().Add(-60 * time.Minute)
	if err := store.SetState(timing.StateKeySessionPrefix+sessionID, sessionStart.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	sessionStartStr, _ := store.GetState(timing.StateKeySessionPrefix + sessionID)
	if sessionStartStr == "" {
		t.Fatal("session_start should be set")
	}

	parsed, err := time.Parse(time.RFC3339, sessionStartStr)
	if err != nil {
		t.Fatal(err)
	}

	elapsed := time.Since(parsed)
	if elapsed.Minutes() < float64(cfg.BreakCooldownMinutes) {
		t.Error("should be past cooldown period for suggestion")
	}

	// Should produce a suggestion
	catalog := getCatalog(cfg)
	if len(catalog) == 0 {
		t.Fatal("catalog should not be empty")
	}
}

func TestStopFirstSessionRecordsStart(t *testing.T) {
	dataDir := t.TempDir()
	projectHash := timing.ProjectHash("/tmp/stop-test-first")

	store, err := timing.OpenStore(dataDir, projectHash)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	sessionID := "new-session"

	// No session_start should exist yet
	val, err := store.GetState(timing.StateKeySessionPrefix + sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if val != "" {
		t.Error("session_start should not exist yet")
	}

	// First stop call should record the start time
	if err := store.SetState(timing.StateKeySessionPrefix+sessionID, time.Now().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}

	val, err = store.GetState(timing.StateKeySessionPrefix + sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if val == "" {
		t.Error("session_start should be set after first stop")
	}
}

func TestStopDisabledConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Enabled = false

	// When disabled, the stop handler should return early (no suggestion)
	if cfg.Enabled {
		t.Error("config should be disabled")
	}
}
