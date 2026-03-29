package notify

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

//go:embed resources/Info.plist
var infoPlist []byte

//go:embed resources/buff-er.icns
var appIcon []byte

//go:embed resources/buff-er-notify
var notifyBinary []byte

// InstallApp creates the buff-er.app notification bundle in ~/Applications.
// Only runs on macOS. Ships a pre-compiled universal binary (arm64 + amd64).
func InstallApp() error {
	if runtime.GOOS != "darwin" {
		return nil
	}

	appPath := appBundlePath()
	if appPath == "" {
		return fmt.Errorf("cannot determine home directory")
	}

	macosDir := filepath.Join(appPath, "Contents", "MacOS")
	resDir := filepath.Join(appPath, "Contents", "Resources")

	if err := os.MkdirAll(macosDir, 0o755); err != nil {
		return fmt.Errorf("create app bundle: %w", err)
	}
	if err := os.MkdirAll(resDir, 0o755); err != nil {
		return fmt.Errorf("create app bundle: %w", err)
	}

	if err := os.WriteFile(filepath.Join(appPath, "Contents", "Info.plist"), infoPlist, 0o644); err != nil {
		return fmt.Errorf("write Info.plist: %w", err)
	}

	if err := os.WriteFile(filepath.Join(resDir, "buff-er.icns"), appIcon, 0o644); err != nil {
		return fmt.Errorf("write icon: %w", err)
	}

	binaryPath := filepath.Join(macosDir, "buff-er-notify")
	if err := os.WriteFile(binaryPath, notifyBinary, 0o755); err != nil {
		return fmt.Errorf("write notification binary: %w", err)
	}

	// Ad-hoc sign so macOS registers it in Notification Center
	sign := exec.Command("codesign", "-s", "-", "-f", appPath)
	if out, err := sign.CombinedOutput(); err != nil {
		return fmt.Errorf("codesign: %w\n%s", err, out)
	}

	return nil
}

// UninstallApp removes the buff-er.app bundle from ~/Applications.
func UninstallApp() error {
	appPath := appBundlePath()
	if appPath == "" {
		return nil
	}
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return nil
	}
	return os.RemoveAll(appPath)
}
