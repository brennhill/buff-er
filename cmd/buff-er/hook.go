package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brennhill/buff-er/internal/config"
	"github.com/brennhill/buff-er/internal/exercise"
	"github.com/brennhill/buff-er/internal/hook"
	"github.com/brennhill/buff-er/internal/message"
	"github.com/brennhill/buff-er/internal/notify"
	"github.com/brennhill/buff-er/internal/timing"
	"github.com/spf13/cobra"
)

// getCatalog converts config exercises to the exercise package type.
func getCatalog(cfg config.Config) []exercise.Exercise {
	custom := make([]exercise.ConfigExercise, len(cfg.Exercises))
	for i, e := range cfg.Exercises {
		custom[i] = exercise.ConfigExercise{
			Name:        e.Name,
			Description: e.Description,
			MinMinutes:  e.MinMinutes,
			MaxMinutes:  e.MaxMinutes,
			Category:    e.Category,
		}
	}
	return exercise.GetCatalog(custom)
}

const warmUpDuration = 60 * time.Second

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook handlers for Claude Code events",
}

var preToolUseCmd = &cobra.Command{
	Use:          "pre-tool-use",
	Short:        "Handle PreToolUse hook event",
	RunE:         runPreToolUse,
	SilenceUsage: true,
}

var postToolUseCmd = &cobra.Command{
	Use:          "post-tool-use",
	Short:        "Handle PostToolUse hook event",
	RunE:         runPostToolUse,
	SilenceUsage: true,
}

var stopCmd = &cobra.Command{
	Use:          "stop",
	Short:        "Handle Stop hook event",
	RunE:         runStop,
	SilenceUsage: true,
}

func init() {
	// Ensure log output goes to stderr, not stdout (stdout is for hook JSON)
	log.SetOutput(os.Stderr)

	hookCmd.AddCommand(preToolUseCmd)
	hookCmd.AddCommand(postToolUseCmd)
	hookCmd.AddCommand(stopCmd)
}

// safeRun wraps a hook handler to ensure it never returns an error.
func safeRun(fn func() (*hook.Output, error)) error {
	out, err := fn()
	if err != nil {
		log.Printf("buff-er: %v", err)
		if writeErr := hook.WriteOutput(&hook.Output{
			SystemMessage: fmt.Sprintf("buff-er: error — %v. Run 'buff-er doctor' to diagnose.", err),
		}); writeErr != nil {
			log.Printf("buff-er: failed to write error output: %v", writeErr)
		}
		return nil
	}
	if writeErr := hook.WriteOutput(out); writeErr != nil {
		log.Printf("buff-er: failed to write output: %v", writeErr)
	}
	return nil
}

// checkBreakDue checks if a break is due and returns a warm-up or full suggestion.
// Returns (output, handled). If handled is true, the caller should return the output.
func checkBreakDue(store *timing.Store) (*hook.Output, bool) {
	breakDueStr, _ := store.GetState(timing.StateKeyBreakDue)
	if breakDueStr == "" {
		return nil, false
	}

	breakDueTime, err := time.Parse(time.RFC3339, breakDueStr)
	if err != nil {
		return nil, false
	}

	elapsed := time.Since(breakDueTime)

	if elapsed < warmUpDuration {
		// Warm-up phase: soft warning
		return &hook.Output{SystemMessage: message.BreakWarning()}, true
	}

	// Past warm-up: fire the full suggestion on PreToolUse only.
	// Stop events during this phase still show warm-up.
	return nil, false
}

// fireBreakSuggestion fires the full exercise suggestion and clears break_due state.
func fireBreakSuggestion(store *timing.Store, cfg config.Config, sessionID string) *hook.Output {
	catalog := getCatalog(cfg)
	ex := exercise.Suggest(catalog, 5.0)
	if ex == nil {
		return nil
	}

	// Clear break_due
	_ = store.SetState(timing.StateKeyBreakDue, "")

	// Extend cooldown by exercise duration
	exerciseOffset := time.Duration(ex.MaxMinutes) * time.Minute
	_ = store.SetState(timing.StateKeyLastSuggestion, time.Now().Add(exerciseOffset).Format(time.RFC3339))
	sessionKey := timing.StateKeySessionPrefix + sessionID
	_ = store.SetState(sessionKey, time.Now().Add(exerciseOffset).Format(time.RFC3339))
	_ = store.PruneState()
	_ = store.SetState(timing.StateKeyPendingFollowUp, "true")

	msg := message.BreakNow(ex.Name, ex.Description)
	notify.Send("buff-er", fmt.Sprintf("Step away. Try: %s", ex.Name))
	return &hook.Output{SystemMessage: msg}
}

// handleBreakDue checks if a break is due on PreToolUse.
// Returns (output, exerciseFired). If exerciseFired, caller should return immediately.
// If output is non-nil but exerciseFired is false, it's a warm-up warning.
func handleBreakDue(store *timing.Store, cfg config.Config, sessionID string) (*hook.Output, bool) {
	breakDueStr, _ := store.GetState(timing.StateKeyBreakDue)
	if breakDueStr == "" {
		return nil, false
	}

	breakDueTime, err := time.Parse(time.RFC3339, breakDueStr)
	if err != nil {
		return nil, false
	}

	if time.Since(breakDueTime) >= warmUpDuration {
		return fireBreakSuggestion(store, cfg, sessionID), true
	}

	return &hook.Output{SystemMessage: message.BreakWarning()}, false
}

// handleCommandEstimation checks if a command is known to be slow and suggests exercise.
func handleCommandEstimation(store *timing.Store, cfg config.Config, est *timing.Estimate, sessionID string) *hook.Output {
	if !est.Confident || est.P75Minutes < cfg.MinTriggerMinutes {
		return nil
	}

	catalog := getCatalog(cfg)
	ex := exercise.Suggest(catalog, est.P75Minutes)
	if ex == nil {
		return nil
	}

	msg := message.ExerciseSuggestion(est.P75Minutes, ex.Name, ex.Description)
	exerciseOffset := time.Duration(ex.MaxMinutes) * time.Minute
	_ = store.SetState(timing.StateKeyLastSuggestion, time.Now().Add(exerciseOffset).Format(time.RFC3339))
	_ = store.SetState(timing.StateKeySessionPrefix+sessionID, time.Now().Add(exerciseOffset).Format(time.RFC3339))
	_ = store.SetState(timing.StateKeyBreakDue, "")
	notify.Send("buff-er", fmt.Sprintf("~%.0fm wait. Try: %s", est.P75Minutes, ex.Name))
	return &hook.Output{SystemMessage: msg}
}

func runPreToolUse(_ *cobra.Command, _ []string) error {
	return safeRun(func() (*hook.Output, error) {
		data, err := hook.ReadInput()
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}

		input, err := hook.ParsePreToolUse(data)
		if err != nil {
			return nil, fmt.Errorf("parse input: %w", err)
		}

		if input.ToolName != "Bash" {
			return nil, nil
		}

		cfg, cfgErr := config.Load()
		if cfgErr != nil {
			log.Printf("buff-er: config error: %v, using defaults", cfgErr)
		}
		if !cfg.Enabled {
			return nil, nil
		}

		pattern := timing.ExtractPattern(input.ToolInput.Command)
		if pattern == "" {
			return nil, nil
		}

		projectHash := timing.ProjectHash(input.CWD)
		store, err := timing.OpenStore(config.DataDir(), projectHash)
		if err != nil {
			return nil, fmt.Errorf("open store: %w", err)
		}
		defer func() { _ = store.Close() }()

		pending := timing.NewPendingStore(input.SessionID)

		// Check if break is due — if past warm-up, fire the full suggestion now
		breakOut, exerciseFired := handleBreakDue(store, cfg, input.SessionID)
		if exerciseFired {
			_ = pending.Set(input.ToolUseID, timing.PendingEntry{
				StartTime:         time.Now(),
				CommandPattern:    pattern,
				ExerciseSuggested: false,
			})
			return breakOut, nil
		}

		est, err := timing.EstimateDuration(store, pattern)
		if err != nil {
			return nil, fmt.Errorf("estimate: %w", err)
		}

		out := handleCommandEstimation(store, cfg, est, input.SessionID)
		exerciseSuggested := out != nil

		// If we're in warm-up and didn't fire a command-based suggestion, show the warning
		if out == nil && breakOut != nil {
			out = breakOut
		}

		_ = pending.Set(input.ToolUseID, timing.PendingEntry{
			StartTime:         time.Now(),
			CommandPattern:    pattern,
			ExerciseSuggested: exerciseSuggested,
		})

		return out, nil
	})
}

func runPostToolUse(_ *cobra.Command, _ []string) error {
	return safeRun(func() (*hook.Output, error) {
		data, err := hook.ReadInput()
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}

		input, err := hook.ParsePostToolUse(data)
		if err != nil {
			return nil, fmt.Errorf("parse input: %w", err)
		}

		if input.ToolName != "Bash" {
			return nil, nil
		}

		pending := timing.NewPendingStore(input.SessionID)
		entry, err := pending.Get(input.ToolUseID)
		if err != nil || entry == nil {
			return nil, nil
		}

		durationMs := time.Since(entry.StartTime).Milliseconds()

		projectHash := timing.ProjectHash(input.CWD)
		store, err := timing.OpenStore(config.DataDir(), projectHash)
		if err != nil {
			return nil, fmt.Errorf("open store: %w", err)
		}
		defer func() { _ = store.Close() }()

		if err := store.Record(entry.CommandPattern, entry.StartTime, durationMs); err != nil {
			return nil, fmt.Errorf("record timing: %w", err)
		}

		// Only prune once per hour
		lastPruneStr, _ := store.GetState(timing.StateKeyLastPrune)
		shouldPrune := true
		if lastPruneStr != "" {
			if lastPrune, parseErr := time.Parse(time.RFC3339, lastPruneStr); parseErr == nil {
				shouldPrune = time.Since(lastPrune) >= time.Hour
			}
		}
		if shouldPrune {
			_ = store.Prune()
			_ = store.SetState(timing.StateKeyLastPrune, time.Now().Format(time.RFC3339))
			timing.CleanupStale()
		}

		if entry.ExerciseSuggested {
			streak, _ := store.IncrementStreak()
			return &hook.Output{
				SystemMessage: message.FollowUp(streak),
			}, nil
		}

		return nil, nil
	})
}

func runStop(_ *cobra.Command, _ []string) error {
	return safeRun(func() (*hook.Output, error) {
		data, err := hook.ReadInput()
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}

		input, err := hook.ParseStop(data)
		if err != nil {
			return nil, fmt.Errorf("parse input: %w", err)
		}

		cfg, cfgErr := config.Load()
		if cfgErr != nil {
			log.Printf("buff-er: config error: %v, using defaults", cfgErr)
		}
		if !cfg.Enabled {
			return nil, nil
		}

		projectHash := timing.ProjectHash(input.CWD)
		store, err := timing.OpenStore(config.DataDir(), projectHash)
		if err != nil {
			return nil, fmt.Errorf("open store: %w", err)
		}
		defer func() { _ = store.Close() }()

		// Check if we owe a follow-up from a previous suggestion
		pendingFollowUp, _ := store.GetState(timing.StateKeyPendingFollowUp)
		if pendingFollowUp == "true" {
			_ = store.SetState(timing.StateKeyPendingFollowUp, "")
			streak, _ := store.IncrementStreak()
			return &hook.Output{
				SystemMessage: message.FollowUp(streak),
			}, nil
		}

		// If break_due is already set, show warm-up message
		if out, handled := checkBreakDue(store); handled {
			return out, nil
		}

		// Global cooldown: don't trigger if a suggestion fired recently
		lastStr, _ := store.GetState(timing.StateKeyLastSuggestion)
		if lastStr != "" {
			lastTime, parseErr := time.Parse(time.RFC3339, lastStr)
			if parseErr == nil {
				if time.Since(lastTime).Minutes() < float64(cfg.BreakCooldownMinutes) {
					return nil, nil
				}
			}
		}

		// Per-session start time
		sessionKey := timing.StateKeySessionPrefix + input.SessionID
		sessionStartStr, _ := store.GetState(sessionKey)
		if sessionStartStr == "" {
			_ = store.SetState(sessionKey, time.Now().Format(time.RFC3339))
			return nil, nil
		}

		sessionStart, err := time.Parse(time.RFC3339, sessionStartStr)
		if err != nil {
			return nil, nil
		}

		elapsed := time.Since(sessionStart)

		// Reset if user was away too long
		maxReasonable := time.Duration(cfg.BreakCooldownMinutes*2) * time.Minute
		if elapsed > maxReasonable {
			_ = store.SetState(sessionKey, time.Now().Format(time.RFC3339))
			return nil, nil
		}

		if elapsed.Minutes() < float64(cfg.BreakCooldownMinutes) {
			return nil, nil
		}

		// Break is due! Set the flag to start the warm-up phase.
		// The actual suggestion will fire on the next PreToolUse after 60s.
		_ = store.SetState(timing.StateKeyBreakDue, time.Now().Format(time.RFC3339))
		return &hook.Output{SystemMessage: message.BreakWarning()}, nil
	})
}
