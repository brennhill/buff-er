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

// getCatalog converts config exercises to the exercise package type and returns
// the appropriate catalog.
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

// safeRun wraps a hook handler to ensure it never returns an error
// (which would cause a non-zero exit code and break the AI workflow).
// Errors are logged to stderr and a systemMessage is shown if appropriate.
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

		est, err := timing.EstimateDuration(store, pattern)
		if err != nil {
			return nil, fmt.Errorf("estimate: %w", err)
		}

		exerciseSuggested := false
		var out *hook.Output

		if est.Confident && est.P75Minutes >= cfg.MinTriggerMinutes {
			catalog := getCatalog(cfg)
			ex := exercise.Suggest(catalog, est.P75Minutes)
			if ex != nil {
				exerciseSuggested = true
				msg := message.ExerciseSuggestion(est.P75Minutes, ex.Name, ex.Description)
				out = &hook.Output{SystemMessage: msg}
				// Set last_suggestion forward by the exercise duration so the break
				// timer extends: if the exercise is 5min, next break won't fire for
				// another cooldown period after the exercise would be done.
				exerciseOffset := time.Duration(ex.MaxMinutes) * time.Minute
				_ = store.SetState(timing.StateKeyLastSuggestion, time.Now().Add(exerciseOffset).Format(time.RFC3339))
				// Also reset the session start timer so "Xm without a break" is accurate
				_ = store.SetState(timing.StateKeySessionPrefix+input.SessionID, time.Now().Add(exerciseOffset).Format(time.RFC3339))
				notify.Send("buff-er", fmt.Sprintf("~%.0fm wait. Try: %s", est.P75Minutes, ex.Name))
			}
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

		// Only prune once per hour to avoid running DELETE on every call
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

		// Check if a previous Stop suggested an exercise and we owe a follow-up
		pendingFollowUp, _ := store.GetState(timing.StateKeyPendingFollowUp)
		if pendingFollowUp == "true" {
			_ = store.SetState(timing.StateKeyPendingFollowUp, "")
			streak, _ := store.IncrementStreak()
			return &hook.Output{
				SystemMessage: message.FollowUp(streak),
			}, nil
		}

		// Global cooldown: don't suggest if any session got a suggestion recently
		lastStr, _ := store.GetState(timing.StateKeyLastSuggestion)
		if lastStr != "" {
			lastTime, parseErr := time.Parse(time.RFC3339, lastStr)
			if parseErr == nil {
				if time.Since(lastTime).Minutes() < float64(cfg.BreakCooldownMinutes) {
					return nil, nil
				}
			}
		}

		// Track time since last activity in this session.
		// Reset the timer if there's been a long gap (> cooldown period),
		// so returning after lunch doesn't immediately trigger "180m without a break."
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

		// If elapsed time is unreasonably large (> 2x cooldown), the user probably
		// stepped away and came back. Reset the timer instead of showing "199m."
		maxReasonable := time.Duration(cfg.BreakCooldownMinutes*2) * time.Minute
		if elapsed > maxReasonable {
			_ = store.SetState(sessionKey, time.Now().Format(time.RFC3339))
			return nil, nil
		}
		if elapsed.Minutes() < float64(cfg.BreakCooldownMinutes) {
			return nil, nil
		}

		catalog := getCatalog(cfg)
		ex := exercise.Suggest(catalog, 5.0)
		if ex == nil {
			return nil, nil
		}

		// Extend cooldown by the exercise duration
		exerciseOffset := time.Duration(ex.MaxMinutes) * time.Minute
		_ = store.SetState(timing.StateKeyLastSuggestion, time.Now().Add(exerciseOffset).Format(time.RFC3339))
		// Reset session timer so elapsed count restarts after the exercise
		_ = store.SetState(sessionKey, time.Now().Add(exerciseOffset).Format(time.RFC3339))
		_ = store.PruneState()

		// Set follow-up flag so the next Stop event asks "did you do it?"
		_ = store.SetState(timing.StateKeyPendingFollowUp, "true")

		elapsedMin := int(elapsed.Minutes())
		msg := message.BreakSuggestion(elapsedMin, ex.Name, ex.Description)
		notify.Send("buff-er", fmt.Sprintf("%dm without a break. Try: %s", elapsedMin, ex.Name))

		return &hook.Output{SystemMessage: msg}, nil
	})
}
