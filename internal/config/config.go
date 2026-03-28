package config

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/brennhill/buff-er/internal/exercise"
	toml "github.com/pelletier/go-toml/v2"
)

// Config holds user configuration.
type Config struct {
	Enabled             bool                `toml:"enabled"`
	MinTriggerMinutes   float64             `toml:"min_trigger_minutes"`
	BreakCooldownMinutes int                `toml:"break_cooldown_minutes"`
	Exercises           []exercise.Exercise `toml:"exercises"`
}

// DefaultConfig returns configuration defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:              true,
		MinTriggerMinutes:    3.0,
		BreakCooldownMinutes: 30,
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
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // no config file is fine
		}
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	return cfg, nil
}

// GetExerciseCatalog returns the exercise catalog, merging user config with defaults.
func GetExerciseCatalog(cfg Config) []exercise.Exercise {
	if len(cfg.Exercises) > 0 {
		return cfg.Exercises
	}
	return exercise.DefaultCatalog()
}
