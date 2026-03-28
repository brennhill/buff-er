package hook

import (
	"encoding/json"
	"fmt"
)

// Output represents the hook response written to stdout.
type Output struct {
	SystemMessage string `json:"systemMessage,omitempty"`
}

// WriteOutput writes the hook response as JSON to stdout.
// Returns nil if the output is empty (no-op).
func WriteOutput(out *Output) error {
	if out == nil || out.SystemMessage == "" {
		return nil
	}
	data, err := json.Marshal(out)
	if err != nil {
		return err
	}
	_, err = fmt.Println(string(data))
	return err
}
