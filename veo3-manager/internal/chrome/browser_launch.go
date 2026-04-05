package chrome

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// launch starts a new Chrome instance with stealth flags
func (bm *BrowserManager) launch() error {
	chromePath := bm.config.ChromePath
	if chromePath == "" {
		chromePath = findChrome()
	}
	if chromePath == "" {
		return fmt.Errorf("Chrome not found. Please set Chrome Path in Settings")
	}

	fixChromePreferences(bm.config.UserDataDir)

	l := launcher.New().
		Bin(chromePath).
		Leakless(false).
		NoSandbox(false).
		UserDataDir(bm.config.UserDataDir).
		Headless(false).
		Delete("no-startup-window").
		Delete("enable-automation").
		Delete("window-size").
		Delete("disable-site-isolation-trials").
		Set("remote-debugging-port", strconv.Itoa(bm.config.DebugPort)).
		Set("no-first-run").
		Set("no-default-browser-check").
		Set("start-maximized").
		Set("hide-crash-restore-bubble")

	u, err := l.Launch()
	if err != nil {
		bm.setStatus(StatusError)
		return fmt.Errorf("failed to launch Chrome: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		bm.setStatus(StatusError)
		return fmt.Errorf("failed to connect to Chrome: %w", err)
	}

	navigateFirstPage(browser)

	bm.browser = browser
	bm.wsURL = u
	bm.chromePath = chromePath
	bm.setStatus(StatusConnected)
	bm.emitInfo()
	go bm.healthCheck()
	return nil
}

// fixChromePreferences patches Chrome prefs to prevent "Restore pages?" dialog
func fixChromePreferences(userDataDir string) {
	prefsPath := filepath.Join(userDataDir, "Default", "Preferences")
	data, err := os.ReadFile(prefsPath)
	if err != nil {
		return
	}

	var prefs map[string]interface{}
	if json.Unmarshal(data, &prefs) != nil {
		return
	}

	profile, ok := prefs["profile"].(map[string]interface{})
	if !ok {
		profile = make(map[string]interface{})
		prefs["profile"] = profile
	}
	profile["exit_type"] = "Normal"
	profile["exited_cleanly"] = true

	if newData, err := json.Marshal(prefs); err == nil {
		_ = os.WriteFile(prefsPath, newData, 0644)
	}
}

// navigateFirstPage opens the target URL in the first tab
func navigateFirstPage(browser *rod.Browser) {
	pages, _ := browser.Pages()
	var page *rod.Page
	if len(pages) > 0 {
		page = pages[0]
		page.MustNavigate("https://labs.google/fx/vi/tools/flow")
		_ = page.MustWaitLoad()
	} else {
		page = browser.MustPage("https://labs.google/fx/vi/tools/flow")
		_ = page.MustWaitLoad()
	}
	_ = proto.EmulationClearDeviceMetricsOverride{}.Call(page)
}
