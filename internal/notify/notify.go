package notify

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// appBundlePath returns the path to the buff-er notification app bundle.
func appBundlePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Applications", "buff-er.app")
}

// Send fires an OS notification in the background. Non-blocking, best-effort.
func Send(title, message string) {
	switch runtime.GOOS {
	case "darwin":
		sendDarwin(title, message)
	case "linux":
		_ = exec.Command("notify-send", title, message).Start()
	}
}

func sendDarwin(title, message string) {
	// Prefer the signed .app bundle for custom icon in Notification Center
	appPath := appBundlePath()
	if appPath != "" {
		if _, err := os.Stat(appPath); err == nil {
			_ = exec.Command("open", appPath, "--args", title, message).Start()
			return
		}
	}
	// Fall back to osascript (shows Script Editor icon)
	script := `display notification "` + escapeAppleScript(message) + `" with title "` + escapeAppleScript(title) + `"`
	_ = exec.Command("osascript", "-e", script).Start()
}

// escapeAppleScript escapes quotes, backslashes, and control characters for
// AppleScript string literals.
func escapeAppleScript(s string) string {
	var out []byte
	for i := range len(s) {
		switch s[i] {
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		case '\n', '\r':
			out = append(out, ' ')
		default:
			out = append(out, s[i])
		}
	}
	return string(out)
}
