package hook

import (
	"encoding/json"
	"os"
)

// Output represents the hook response written to stdout.
type Output struct {
	SystemMessage string `json:"systemMessage,omitempty"`
}

// WriteOutput writes the hook response as JSON to stdout.
// Returns nil if the output is empty (no-op).
// stdout must remain clean JSON only — all logging goes to stderr.
func WriteOutput(out *Output) error {
	if out == nil || out.SystemMessage == "" {
		return nil
	}
	data, err := json.Marshal(out)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = os.Stdout.Write(data)
	return err
}
