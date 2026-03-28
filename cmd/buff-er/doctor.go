package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/brennhill/buff-er/internal/config"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check buff-er health and configuration",
	RunE:  runDoctor,
}

func runDoctor(_ *cobra.Command, _ []string) error {
	fmt.Println("buff-er doctor")
	fmt.Println("==============")
	fmt.Println()

	checkBinary()
	checkHookRegistration()
	checkConfig()
	checkDataDir()

	fmt.Println()
	return nil
}

func checkBinary() {
	path, err := exec.LookPath("buff-er")
	if err != nil {
		fmt.Println("[WARN] buff-er not found in PATH")
	} else {
		fmt.Printf("[OK]   binary: %s\n", path)
	}
}

func checkHookRegistration() {
	settingsFile, err := settingsPath()
	if err != nil {
		fmt.Printf("[WARN] %v\n", err)
		return
	}
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		fmt.Printf("[WARN] settings not found: %s\n", settingsFile)
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		fmt.Printf("[FAIL] settings parse error: %v\n", err)
		return
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		fmt.Println("[WARN] no hooks section in settings")
		return
	}

	for _, event := range []string{"PreToolUse", "PostToolUse", "Stop"} {
		entries, ok := hooks[event].([]interface{})
		if !ok {
			fmt.Printf("[MISS] hook not registered: %s\n", event)
			continue
		}
		found := false
		for _, e := range entries {
			if isBuffErEntry(e) {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("[OK]   hook registered: %s\n", event)
		} else {
			fmt.Printf("[MISS] hook not registered: %s\n", event)
		}
	}
}

func checkConfig() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("[WARN] config error: %v (using defaults)\n", err)
		return
	}

	configPath := config.ConfigPath()
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		fmt.Println("[OK]   config: using defaults (no config file)")
	} else {
		fmt.Printf("[OK]   config: %s\n", configPath)
	}
	fmt.Printf("       enabled: %v\n", cfg.Enabled)
	fmt.Printf("       min trigger: %.0f min\n", cfg.MinTriggerMinutes)
	fmt.Printf("       break cooldown: %d min\n", cfg.BreakCooldownMinutes)
	if len(cfg.Exercises) > 0 {
		fmt.Printf("       custom exercises: %d\n", len(cfg.Exercises))
	}
}

func checkDataDir() {
	dataDir := config.DataDir()
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("[OK]   data: no timing data yet (learning phase)")
		} else {
			fmt.Printf("[WARN] data dir error: %v\n", err)
		}
		return
	}

	projects := 0
	for _, e := range entries {
		if e.IsDir() {
			projects++
		}
	}
	fmt.Printf("[OK]   data: %s (%d project(s))\n", dataDir, projects)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dbPath := filepath.Join(dataDir, e.Name(), "timings.db")
		if _, statErr := os.Stat(dbPath); statErr == nil {
			name := e.Name()
			if len(name) > 8 {
				name = name[:8]
			}
			fmt.Printf("       project %s: has timing data\n", name)
		}
	}
}
