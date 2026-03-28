package hook

import (
	"encoding/json"
	"io"
	"os"
)

// Input represents the common fields in all hook event payloads.
type Input struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`
	ToolName       string `json:"tool_name,omitempty"`
	ToolUseID      string `json:"tool_use_id,omitempty"`
}

// PreToolUseInput represents the PreToolUse hook payload.
type PreToolUseInput struct {
	Input
	ToolInput BashToolInput `json:"tool_input"`
}

// PostToolUseInput represents the PostToolUse hook payload.
type PostToolUseInput struct {
	Input
	ToolInput    BashToolInput `json:"tool_input"`
	ToolResponse BashResponse  `json:"tool_response"`
}

// StopInput represents the Stop hook payload.
type StopInput struct {
	Input
	StopHookActive       bool   `json:"stop_hook_active"`
	LastAssistantMessage string `json:"last_assistant_message"`
}

// BashToolInput represents the tool_input for Bash commands.
type BashToolInput struct {
	Command string `json:"command"`
}

// BashResponse represents the tool_response for Bash commands.
type BashResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// ReadInput reads and parses the hook input from stdin.
func ReadInput() ([]byte, error) {
	return io.ReadAll(os.Stdin)
}

// ParsePreToolUse parses a PreToolUse payload from raw JSON.
func ParsePreToolUse(data []byte) (*PreToolUseInput, error) {
	var input PreToolUseInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ParsePostToolUse parses a PostToolUse payload from raw JSON.
func ParsePostToolUse(data []byte) (*PostToolUseInput, error) {
	var input PostToolUseInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}
	return &input, nil
}

// ParseStop parses a Stop payload from raw JSON.
func ParseStop(data []byte) (*StopInput, error) {
	var input StopInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, err
	}
	return &input, nil
}
