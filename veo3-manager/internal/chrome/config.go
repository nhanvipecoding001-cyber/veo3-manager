package chrome

import (
	"os"
	"path/filepath"
)

type ChromeConfig struct {
	ChromePath  string
	UserDataDir string
	DebugPort   int
}

func DefaultChromeConfig() *ChromeConfig {
	appData, _ := os.UserConfigDir()
	return &ChromeConfig{
		ChromePath:  findChrome(),
		UserDataDir: filepath.Join(appData, "veo3-manager", "chrome-data"),
		DebugPort:   9222,
	}
}

func findChrome() string {
	candidates := []string{
		os.Getenv("PROGRAMFILES") + `\Google\Chrome\Application\chrome.exe`,
		os.Getenv("PROGRAMFILES(X86)") + `\Google\Chrome\Application\chrome.exe`,
		os.Getenv("LOCALAPPDATA") + `\Google\Chrome\Application\chrome.exe`,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
