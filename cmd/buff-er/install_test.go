package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallCreatesSettings(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	settingsFile := filepath.Join(tmpHome, ".claude", "settings.json")

	binPath := "/usr/local/bin/buff-er"
	entries := hookEntries(binPath)

	settings := make(map[string]interface{})
	hooks := make(map[string]interface{})
	for event, entry := range entries {
		hooks[event] = []interface{}{entry}
	}
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := writeSettingsAtomic(settingsFile, out); err != nil {
		t.Fatal(err)
	}

	// Verify file was created
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("settings file not valid JSON: %v", err)
	}

	hooksSection, ok := parsed["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("no hooks section in settings")
	}

	for _, event := range []string{"PreToolUse", "PostToolUse", "Stop"} {
		eventEntries, ok := hooksSection[event].([]interface{})
		if !ok {
			t.Errorf("no entry for %s", event)
			continue
		}
		if len(eventEntries) == 0 {
			t.Errorf("empty entries for %s", event)
		}
		found := false
		for _, e := range eventEntries {
			if isBuffErEntry(e) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("buff-er entry not found for %s", event)
		}
	}
}

func TestInstallPreservesExistingHooks(t *testing.T) {
	tmpHome := t.TempDir()
	settingsFile := filepath.Join(tmpHome, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsFile), 0o755); err != nil {
		t.Fatal(err)
	}

	// Write existing settings with a non-buff-er hook
	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "other-tool hook pre",
						},
					},
				},
			},
		},
	}
	data, err := json.Marshal(existing)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	settings := installHooksIntoFile(t, settingsFile)

	// Verify: PreToolUse should have 2 entries (original + buff-er)
	resultHooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks section missing from result")
	}
	preEntries, ok := resultHooks["PreToolUse"].([]interface{})
	if !ok {
		t.Fatal("PreToolUse section missing from result")
	}
	if len(preEntries) != 2 {
		t.Errorf("expected 2 PreToolUse entries, got %d", len(preEntries))
	}
}

// installHooksIntoFile reads a settings file, adds buff-er hooks, writes it back,
// and returns the resulting parsed settings.
func installHooksIntoFile(t *testing.T, settingsFile string) map[string]interface{} {
	t.Helper()

	rawData, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatal(err)
	}
	settings := make(map[string]interface{})
	if err := json.Unmarshal(rawData, &settings); err != nil {
		t.Fatal(err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks section missing")
	}

	addBuffErHooks(t, hooks)
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSettingsAtomic(settingsFile, out); err != nil {
		t.Fatal(err)
	}

	readBack, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(readBack, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

// addBuffErHooks merges buff-er hook entries into existing hooks, skipping duplicates.
func addBuffErHooks(t *testing.T, hooks map[string]interface{}) {
	t.Helper()

	entries := hookEntries("/usr/local/bin/buff-er")
	for event, entry := range entries {
		existing, ok := hooks[event].([]interface{})
		if !ok {
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
}

func TestUninstallRemovesBuffErHooks(t *testing.T) {
	tmpHome := t.TempDir()
	settingsFile := filepath.Join(tmpHome, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsFile), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create settings with buff-er + another hook
	binPath := "/usr/local/bin/buff-er"
	entries := hookEntries(binPath)
	hooks := map[string]interface{}{
		"PreToolUse": []interface{}{
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{
						"type":    "command",
						"command": "other-tool hook pre",
					},
				},
			},
			entries["PreToolUse"],
		},
	}
	settings := map[string]interface{}{"hooks": hooks}
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Simulate uninstall logic
	rawData, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(rawData, &parsed); err != nil {
		t.Fatal(err)
	}

	hooksSection, ok := parsed["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks section missing")
	}
	removed := 0
	for event, val := range hooksSection {
		eventEntries, ok := val.([]interface{})
		if !ok {
			continue
		}
		var filtered []interface{}
		for _, e := range eventEntries {
			if isBuffErEntry(e) {
				removed++
			} else {
				filtered = append(filtered, e)
			}
		}
		if len(filtered) == 0 {
			delete(hooksSection, event)
		} else {
			hooksSection[event] = filtered
		}
	}

	if removed == 0 {
		t.Error("expected to remove at least one hook")
	}

	// PreToolUse should still exist with 1 entry
	preEntries, ok := hooksSection["PreToolUse"].([]interface{})
	if !ok {
		t.Fatal("PreToolUse should still exist")
	}
	if len(preEntries) != 1 {
		t.Errorf("expected 1 remaining entry, got %d", len(preEntries))
	}
}

func TestIsBuffErEntry(t *testing.T) {
	tests := []struct {
		name  string
		entry interface{}
		want  bool
	}{
		{
			name: "buff-er entry",
			entry: map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{
						"type":    "command",
						"command": "/usr/local/bin/buff-er hook pre-tool-use",
					},
				},
			},
			want: true,
		},
		{
			name: "non buff-er entry",
			entry: map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{
						"type":    "command",
						"command": "other-tool hook pre",
					},
				},
			},
			want: false,
		},
		{
			name:  "wrong type",
			entry: "not a map",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBuffErEntry(tt.entry)
			if got != tt.want {
				t.Errorf("isBuffErEntry = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHookEntries(t *testing.T) {
	entries := hookEntries("/usr/local/bin/buff-er")

	for _, event := range []string{"PreToolUse", "PostToolUse", "Stop"} {
		entry, ok := entries[event]
		if !ok {
			t.Errorf("missing entry for %s", event)
			continue
		}
		m, ok := entry.(map[string]interface{})
		if !ok {
			t.Errorf("entry for %s is not a map", event)
			continue
		}
		hooks, ok := m["hooks"].([]interface{})
		if !ok || len(hooks) == 0 {
			t.Errorf("entry for %s has no hooks", event)
		}
	}
}
