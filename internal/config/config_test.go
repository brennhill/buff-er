package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	cfg, err := LoadFromPath(filepath.Join(t.TempDir(), "nonexistent.toml"))
	if err != nil {
		t.Fatalf("Load with missing file should not error: %v", err)
	}
	if !cfg.Enabled {
		t.Error("default config should be enabled")
	}
	if cfg.MinTriggerMinutes != 3.0 {
		t.Errorf("MinTriggerMinutes = %f, want 3.0", cfg.MinTriggerMinutes)
	}
	if cfg.BreakCooldownMinutes != 52 {
		t.Errorf("BreakCooldownMinutes = %d, want 52", cfg.BreakCooldownMinutes)
	}
}

func TestLoadValidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := []byte(`
enabled = false
min_trigger_minutes = 5.0
break_cooldown_minutes = 45
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("Load valid file: %v", err)
	}
	if cfg.Enabled {
		t.Error("expected enabled = false")
	}
	if cfg.MinTriggerMinutes != 5.0 {
		t.Errorf("MinTriggerMinutes = %f, want 5.0", cfg.MinTriggerMinutes)
	}
	if cfg.BreakCooldownMinutes != 45 {
		t.Errorf("BreakCooldownMinutes = %d, want 45", cfg.BreakCooldownMinutes)
	}
}

func TestLoadMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := []byte(`this is not valid toml {{{`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err == nil {
		t.Error("expected error for malformed TOML")
	}
	// Should still return defaults since nothing could be parsed
	if !cfg.Enabled {
		t.Error("malformed file with no valid keys should keep default enabled=true")
	}
}

func TestLoadPartialFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	// The TOML parser will parse enabled=false before hitting the type error
	content := []byte(`enabled = false
min_trigger_minutes = 7.0
break_cooldown_minutes = "not a number"
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err == nil {
		t.Error("expected error for partial TOML")
	}
	// The critical fix: enabled=false should be preserved, not reset to true
	if cfg.Enabled {
		t.Error("partial parse should preserve enabled=false from the file")
	}
}

func TestLoadFileWithCustomExercises(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := []byte(`
enabled = true
min_trigger_minutes = 3.0

[[exercises]]
name = "Jumping jacks"
description = "Do 20 jumping jacks"
min_minutes = 1
max_minutes = 2
category = "movement"
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("Load with exercises: %v", err)
	}
	if len(cfg.Exercises) != 1 {
		t.Fatalf("expected 1 exercise, got %d", len(cfg.Exercises))
	}
	if cfg.Exercises[0].Name != "Jumping jacks" {
		t.Errorf("exercise name = %q, want 'Jumping jacks'", cfg.Exercises[0].Name)
	}
}

func TestClampEnforcesBounds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := []byte(`
min_trigger_minutes = 0.1
break_cooldown_minutes = 2
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("Load with below-minimum values: %v", err)
	}
	if cfg.MinTriggerMinutes != 0.5 {
		t.Errorf("MinTriggerMinutes = %f, want 0.5 (clamped from 0.1)", cfg.MinTriggerMinutes)
	}
	if cfg.BreakCooldownMinutes != 5 {
		t.Errorf("BreakCooldownMinutes = %d, want 5 (clamped from 2)", cfg.BreakCooldownMinutes)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Error("default should be enabled")
	}
	if cfg.MinTriggerMinutes != 3.0 {
		t.Error("default MinTriggerMinutes should be 3.0")
	}
	if cfg.BreakCooldownMinutes != 52 {
		t.Error("default BreakCooldownMinutes should be 30")
	}
	if len(cfg.Exercises) != 0 {
		t.Error("default should have no custom exercises")
	}
}
