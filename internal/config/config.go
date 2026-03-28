package config

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	toml "github.com/pelletier/go-toml/v2"
)

// Exercise mirrors the exercise type for config parsing only.
type Exercise struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	MinMinutes  int    `toml:"min_minutes"`
	MaxMinutes  int    `toml:"max_minutes"`
	Category    string `toml:"category"`
}

// Config holds user configuration.
type Config struct {
	Enabled              bool       `toml:"enabled"`
	MinTriggerMinutes    float64    `toml:"min_trigger_minutes"`
	BreakCooldownMinutes int        `toml:"break_cooldown_minutes"`
	Exercises            []Exercise `toml:"exercises"`
}

// DefaultConfig returns configuration defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		MinTriggerMinutes:    3.0,
		BreakCooldownMinutes: 52,
	}
}

// DataDir returns the XDG data directory for buff-er.
func DataDir() string {
	return filepath.Join(xdg.DataHome, "buff-er")
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	return filepath.Join(xdg.ConfigHome, "buff-er", "config.toml")
}

// Load reads the config from disk, returning defaults if the file doesn't exist
// or can't be parsed. Returns the config and any error (for logging).
func Load() (Config, error) {
	return LoadFromPath(ConfigPath())
}

// LoadFromPath reads the config from the given path, returning defaults if the
// file doesn't exist. On parse errors, returns the partially-parsed config
// (preserving any fields that were successfully read) along with the error.
func LoadFromPath(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // no config file is fine
		}
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		clamp(&cfg)
		return cfg, err
	}

	clamp(&cfg)
	return cfg, nil
}

// clamp ensures config values are within sane bounds.
func clamp(cfg *Config) {
	if cfg.MinTriggerMinutes < 0.5 {
		cfg.MinTriggerMinutes = 0.5
	}
	if cfg.BreakCooldownMinutes < 5 {
		cfg.BreakCooldownMinutes = 5
	}
}
