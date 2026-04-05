package chrome

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Status string

const (
	StatusDisconnected Status = "disconnected"
	StatusConnecting   Status = "connecting"
	StatusConnected    Status = "connected"
	StatusError        Status = "error"
)

type BrowserInfo struct {
	Status       string   `json:"status"`
	ChromePath   string   `json:"chromePath"`
	ProfilePath  string   `json:"profilePath"`
	DebugPort    int      `json:"debugPort"`
	WebSocketURL string   `json:"webSocketURL"`
	Version      string   `json:"version"`
	Stealth      bool     `json:"stealth"`
	StealthMods  []string `json:"stealthMods"`
}

type BrowserManager struct {
	browser    *rod.Browser
	config     *ChromeConfig
	status     Status
	token      string
	tokenTime  time.Time
	projectId  string
	wsURL      string
	chromePath string
	ctx        context.Context
	mu         sync.Mutex
	stopHealth chan struct{}
}

func NewBrowserManager(cfg *ChromeConfig) *BrowserManager {
	return &BrowserManager{
		config:     cfg,
		status:     StatusDisconnected,
		stopHealth: make(chan struct{}),
	}
}

func (bm *BrowserManager) SetContext(ctx context.Context) {
	bm.ctx = ctx
}

func (bm *BrowserManager) Config() *ChromeConfig {
	return bm.config
}

func (bm *BrowserManager) Status() Status {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.status
}

func (bm *BrowserManager) setStatus(s Status) {
	bm.status = s
	if bm.ctx != nil {
		runtime.EventsEmit(bm.ctx, "browser:status", string(s))
	}
}

func (bm *BrowserManager) Connect() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.setStatus(StatusConnecting)

	// Try connecting to existing Chrome on debug port
	wsURL, err := bm.getExistingDebugURL()
	if err == nil && wsURL != "" {
		browser := rod.New().ControlURL(wsURL)
		if err := browser.Connect(); err == nil {
			bm.browser = browser
			bm.wsURL = wsURL
			bm.chromePath = bm.config.ChromePath
			if bm.chromePath == "" {
				bm.chromePath = findChrome()
			}
			bm.setStatus(StatusConnected)
			bm.emitInfo()
			go bm.healthCheck()
			return nil
		}
	}

	return bm.launch()
}

func (bm *BrowserManager) getExistingDebugURL() (string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", bm.config.DebugPort)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.WebSocketDebuggerURL, nil
}

func (bm *BrowserManager) Disconnect() error {
	select {
	case bm.stopHealth <- struct{}{}:
	default:
	}

	bm.mu.Lock()
	browser := bm.browser
	bm.browser = nil
	bm.token = ""
	bm.wsURL = ""
	bm.chromePath = ""
	bm.setStatus(StatusDisconnected)
	bm.mu.Unlock()

	if browser != nil {
		go func() {
			done := make(chan struct{})
			go func() {
				defer close(done)
				browser.Close()
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
			}
		}()
	}

	return nil
}

func (bm *BrowserManager) Browser() *rod.Browser {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.browser
}

func (bm *BrowserManager) GetToken() string {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.token
}

func (bm *BrowserManager) SetToken(token string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.token = token
	bm.tokenTime = time.Now()
}

func (bm *BrowserManager) GetInfo() *BrowserInfo {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.buildInfo()
}

func (bm *BrowserManager) buildInfo() *BrowserInfo {
	info := &BrowserInfo{
		Status:       string(bm.status),
		ChromePath:   bm.chromePath,
		ProfilePath:  bm.config.UserDataDir,
		DebugPort:    bm.config.DebugPort,
		WebSocketURL: bm.wsURL,
		Stealth:      true,
		StealthMods: []string{
			"navigator.webdriver: hidden",
			"navigator.platform: Win32",
			"hardwareConcurrency: 8",
			"deviceMemory: 8",
			"languages: en-US, en",
			"permissions.query: overridden",
			"go-rod/stealth: active",
		},
	}

	if bm.browser != nil {
		if ver, err := bm.browser.Version(); err == nil {
			info.Version = ver.Product
		}
	}

	return info
}

func (bm *BrowserManager) emitInfo() {
	if bm.ctx != nil {
		runtime.EventsEmit(bm.ctx, "browser:info", bm.buildInfo())
	}
}

func (bm *BrowserManager) healthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bm.mu.Lock()
			b := bm.browser
			bm.mu.Unlock()

			if b == nil {
				continue
			}
			if _, err := b.Version(); err != nil {
				bm.mu.Lock()
				if bm.browser == b {
					bm.setStatus(StatusDisconnected)
					bm.browser = nil
				}
				bm.mu.Unlock()
			}
		case <-bm.stopHealth:
			return
		}
	}
}
