package timing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PendingEntry tracks a started tool use that hasn't completed yet.
type PendingEntry struct {
	StartTime       time.Time `json:"start_time"`
	CommandPattern  string    `json:"command_pattern"`
	ExerciseSuggested bool   `json:"exercise_suggested"`
}

// PendingStore manages in-flight tool use tracking via temp files.
type PendingStore struct {
	dir string
}

// NewPendingStore creates a pending store for a session.
func NewPendingStore(sessionID string) *PendingStore {
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("buff-er-%s", sessionID))
	os.MkdirAll(dir, 0755)
	return &PendingStore{dir: dir}
}

// Set records a pending tool use.
func (p *PendingStore) Set(toolUseID string, entry PendingEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(p.dir, toolUseID+".json")
	return os.WriteFile(path, data, 0644)
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
		os.Remove(path)
		return nil, err
	}

	os.Remove(path) // clean up after reading
	return &entry, nil
}
