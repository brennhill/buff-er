package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/brennhill/buff-er/internal/config"
	"github.com/brennhill/buff-er/internal/exercise"
	"github.com/brennhill/buff-er/internal/hook"
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
				out = &hook.Output{
					SystemMessage: fmt.Sprintf(
						"buff-er: This usually takes ~%.0fm. Perfect time for: %s — %s",
						est.P75Minutes, ex.Name, ex.Description,
					),
				}
				_ = store.SetState(timing.StateKeyLastSuggestion, time.Now().Format(time.RFC3339))
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
			return &hook.Output{
				SystemMessage: "buff-er: Done! Did you get that exercise in? (you know the answer)",
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

		lastStr, _ := store.GetState(timing.StateKeyLastSuggestion)
		if lastStr != "" {
			lastTime, parseErr := time.Parse(time.RFC3339, lastStr)
			if parseErr == nil {
				elapsed := time.Since(lastTime)
				if elapsed.Minutes() < float64(cfg.BreakCooldownMinutes) {
					return nil, nil
				}
			}
		}

		sessionStartStr, _ := store.GetState(timing.StateKeySessionPrefix + input.SessionID)
		if sessionStartStr == "" {
			_ = store.SetState(timing.StateKeySessionPrefix+input.SessionID, time.Now().Format(time.RFC3339))
			return nil, nil
		}

		sessionStart, err := time.Parse(time.RFC3339, sessionStartStr)
		if err != nil {
			return nil, nil
		}

		elapsed := time.Since(sessionStart)
		if elapsed.Minutes() < float64(cfg.BreakCooldownMinutes) {
			return nil, nil
		}

		catalog := getCatalog(cfg)
		ex := exercise.Suggest(catalog, 5.0)
		if ex == nil {
			return nil, nil
		}

		_ = store.SetState(timing.StateKeyLastSuggestion, time.Now().Format(time.RFC3339))
		_ = store.PruneState()

		elapsedStr := strconv.Itoa(int(elapsed.Minutes()))

		return &hook.Output{
			SystemMessage: fmt.Sprintf(
				"buff-er: You've been at it for %sm without a break. Time to move! Try: %s — %s",
				elapsedStr, ex.Name, ex.Description,
			),
		}, nil
	})
}
