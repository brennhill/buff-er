package timing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// PendingEntry tracks a started tool use that hasn't completed yet.
type PendingEntry struct {
	StartTime         time.Time `json:"start_time"`
	CommandPattern    string    `json:"command_pattern"`
	ExerciseSuggested bool      `json:"exercise_suggested"`
}

// PendingStore manages in-flight tool use tracking via temp files.
type PendingStore struct {
	dir string
}

// NewPendingStore creates a pending store for a session.
func NewPendingStore(sessionID string) *PendingStore {
	dir := filepath.Join(os.TempDir(), "buff-er-"+sessionID)
	_ = os.MkdirAll(dir, 0o755)
	return &PendingStore{dir: dir}
}

// Set records a pending tool use.
func (p *PendingStore) Set(toolUseID string, entry PendingEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(p.dir, toolUseID+".json")
	return os.WriteFile(path, data, 0o644)
}

// Get retrieves and removes a pending tool use entry.
func (p *PendingStore) Get(toolUseID string) (*PendingEntry, error) {
	path := filepath.Join(p.dir, toolUseID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entry PendingEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	_ = os.Remove(path) // clean up after reading
	return &entry, nil
}

// CleanupStale removes pending entries older than 1 hour from all sessions.
func CleanupStale() {
	pattern := filepath.Join(os.TempDir(), "buff-er-*")
	dirs, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-1 * time.Hour)
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		allRemoved := true
		for _, e := range entries {
			info, err := e.Info()
			if err != nil || info.ModTime().After(cutoff) {
				allRemoved = false
				continue
			}
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
		if allRemoved {
			_ = os.Remove(dir)
		}
	}
}
