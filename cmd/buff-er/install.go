package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Register buff-er hooks in Claude Code settings",
	RunE:  runInstall,
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove buff-er hooks from Claude Code settings",
	RunE:  runUninstall,
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func binaryPath() (string, error) {
	path, err := exec.LookPath("buff-er")
	if err != nil {
		return os.Executable()
	}
	return path, nil
}

// hookEntries returns the hook configuration entries for buff-er.
func hookEntries(binPath string) map[string]interface{} {
	makeEntry := func(subcommand string, matcher string) map[string]interface{} {
		entry := map[string]interface{}{
			"hooks": []interface{}{
				map[string]interface{}{
					"type":    "command",
					"command": binPath + " hook " + subcommand,
					"timeout": 5,
				},
			},
		}
		if matcher != "" {
			entry["matcher"] = matcher
		}
		return entry
	}

	return map[string]interface{}{
		"PreToolUse":  makeEntry("pre-tool-use", "Bash"),
		"PostToolUse": makeEntry("post-tool-use", "Bash"),
		"Stop":        makeEntry("stop", ""),
	}
}

// writeSettingsAtomic writes settings JSON to a temp file then renames into place.
func writeSettingsAtomic(settingsFile string, data []byte) error {
	dir := filepath.Dir(settingsFile)
	_ = os.MkdirAll(dir, 0o755)

	tmp, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, settingsFile); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp to settings: %w", err)
	}
	return nil
}

func runInstall(_ *cobra.Command, _ []string) error {
	binPath, err := binaryPath()
	if err != nil {
		return fmt.Errorf("cannot find buff-er binary: %w", err)
	}

	settingsFile, err := settingsPath()
	if err != nil {
		return err
	}

	settings := make(map[string]interface{})
	data, err := os.ReadFile(settingsFile)
	if err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse %s: %w", settingsFile, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", settingsFile, err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
	}

	entries := hookEntries(binPath)
	for event, entry := range entries {
		existing, ok := hooks[event].([]interface{})
		if !ok {
			// If the event exists but is not an array, warn and skip to avoid data loss
			if _, exists := hooks[event]; exists {
				fmt.Printf("buff-er: warning: hooks.%s has unexpected format, skipping\n", event)
				continue
			}
			existing = []interface{}{}
		}

		alreadyInstalled := false
		for _, e := range existing {
			if isBuffErEntry(e) {
				alreadyInstalled = true
				break
			}
		}

		if !alreadyInstalled {
			existing = append(existing, entry)
		}
		hooks[event] = existing
	}

	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	if err := writeSettingsAtomic(settingsFile, out); err != nil {
		return err
	}

	fmt.Printf("buff-er: installed hooks in %s\n", settingsFile)
	fmt.Printf("buff-er: using binary at %s\n", binPath)
	fmt.Println("buff-er: ready! Exercise suggestions will appear during long-running commands.")
	return nil
}

func runUninstall(_ *cobra.Command, _ []string) error {
	settingsFile, err := settingsPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(settingsFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("buff-er: no settings file found, nothing to uninstall")
			return nil
		}
		return fmt.Errorf("read %s: %w", settingsFile, err)
	}

	settings := make(map[string]interface{})
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parse %s: %w", settingsFile, err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		fmt.Println("buff-er: no hooks found, nothing to uninstall")
		return nil
	}

	removed := 0
	for event, val := range hooks {
		entries, ok := val.([]interface{})
		if !ok {
			continue
		}

		var filtered []interface{}
		for _, e := range entries {
			if isBuffErEntry(e) {
				removed++
			} else {
				filtered = append(filtered, e)
			}
		}

		if len(filtered) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = filtered
		}
	}

	if removed == 0 {
		fmt.Println("buff-er: no buff-er hooks found, nothing to uninstall")
		return nil
	}

	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	if err := writeSettingsAtomic(settingsFile, out); err != nil {
		return err
	}

	fmt.Printf("buff-er: removed %d hook(s) from %s\n", removed, settingsFile)
	return nil
}

// isBuffErEntry checks if a hook entry contains a buff-er hook command.
func isBuffErEntry(entry interface{}) bool {
	m, ok := entry.(map[string]interface{})
	if !ok {
		return false
	}
	innerHooks, ok := m["hooks"].([]interface{})
	if !ok {
		return false
	}
	for _, h := range innerHooks {
		hm, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "buff-er hook ") {
			return true
		}
	}
	return false
}
