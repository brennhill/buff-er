package notify

import (
	"os/exec"
	"runtime"
)

// Send fires an OS notification in the background. Non-blocking, best-effort.
// Falls back silently if the notification system is unavailable.
func Send(title, message string) {
	switch runtime.GOOS {
	case "darwin":
		script := `display notification "` + escapeAppleScript(message) + `" with title "` + escapeAppleScript(title) + `"`
		_ = exec.Command("osascript", "-e", script).Start()
	case "linux":
		_ = exec.Command("notify-send", title, message).Start()
	}
}

// escapeAppleScript escapes quotes and backslashes for AppleScript strings.
func escapeAppleScript(s string) string {
	var out []byte
	for i := range len(s) {
		switch s[i] {
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}
