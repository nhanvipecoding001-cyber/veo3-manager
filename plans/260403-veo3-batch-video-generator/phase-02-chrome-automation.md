# Phase 2: Chrome Automation Engine

## Context

- **Parent plan:** [plan.md](./plan.md)
- **Dependencies:** [Phase 1](./phase-01-scaffolding.md) (project structure, database)
- **Research:** [researcher-02-report.md](./research/researcher-02-report.md) — anti-detection, stealth patterns

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-04-03 |
| Description | Rod/stealth Chrome automation — launch, connect, persist sessions, extract auth tokens, stealth bypass |
| Priority | P0 — Critical |
| Implementation Status | Not Started |
| Review Status | Pending |

## Key Insights

- **Fact #3:** MUST use `go-rod/stealth` — override `navigator.webdriver`, fake plugins/languages/platform. Chrome launched with `disable-blink-features=AutomationControlled`. Without stealth = blocked.
- **Fact #4:** Launch with `--user-data-dir` for persistent login. Before launching new Chrome, try connecting to existing one via `GET /json/version` on debug port.
- **Fact #2:** Auth token from `__NEXT_DATA__` JS variable, extracted via `page.MustEval()`.
- Token refresh needed — tokens expire; re-extract on 401.

## Requirements

1. Launch Chrome with stealth flags and persistent user-data-dir
2. Connect to existing Chrome instance if already running on debug port
3. Apply go-rod/stealth to every page before navigation
4. Extract auth token from `__NEXT_DATA__` on Google Labs page
5. Expose browser status (disconnected/connecting/connected/error) to frontend via Wails events
6. Graceful Chrome lifecycle management (launch, reconnect, shutdown)

## Architecture

```
internal/chrome/
├── browser.go     # BrowserManager — launch, connect, disconnect, status
├── stealth.go     # Stealth page creation, script injection
├── session.go     # Token extraction, refresh, validation
└── config.go      # ChromeConfig struct
```

### BrowserManager State Machine
```
Disconnected → Connecting → Connected → Error
     ↑              ↑           |          |
     └──────────────┴───────────┴──────────┘
```

### Token Extraction Flow
```
1. Navigate to labs.google
2. Wait for page load (networkIdle)
3. page.MustEval(`() => JSON.stringify(window.__NEXT_DATA__)`)
4. Parse JSON → extract token from props.pageProps or similar path
5. Cache token in memory (not DB)
6. On 401 during API call → re-extract
```

## Related Code Files

| File | Purpose |
|------|---------|
| `internal/chrome/browser.go` | `BrowserManager` struct — launch/connect/status |
| `internal/chrome/stealth.go` | `NewStealthPage()` — stealth-injected page creation |
| `internal/chrome/session.go` | `ExtractToken()`, `RefreshToken()` |
| `internal/chrome/config.go` | `ChromeConfig` — path, user-data-dir, debug port |
| `app.go` | New methods: `LaunchBrowser()`, `GetBrowserStatus()`, `DisconnectBrowser()` |

## Implementation Steps

### 1. Add Go Dependencies
```bash
go get github.com/go-rod/rod
go get github.com/go-rod/stealth
```

### 2. Create `internal/chrome/config.go`
```go
type ChromeConfig struct {
    ChromePath  string // Path to chrome.exe (empty = auto-detect)
    UserDataDir string // Persistent profile directory
    DebugPort   int    // Default 9222
}
```
- Load from settings DB on startup
- Auto-detect Chrome path: check `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\chrome.exe` registry, fallback to common paths

### 3. Create `internal/chrome/browser.go`
- `BrowserManager` struct: holds `*rod.Browser`, `ChromeConfig`, status, mutex
- `Connect() error`:
  1. Try `GET http://localhost:{debugPort}/json/version` (Fact #4)
  2. If response OK → extract `webSocketDebuggerUrl` → `rod.New().ControlURL(wsURL).Connect()`
  3. If connection fails → fall through to `Launch()`
- `Launch() error`:
  1. Use `launcher.New()` with flags:
     - `.Bin(config.ChromePath)` if set
     - `.UserDataDir(config.UserDataDir)`
     - `.Headless(false)`
     - `.Set("disable-blink-features", "AutomationControlled")` (Fact #3)
     - `.Set("remote-debugging-port", strconv.Itoa(config.DebugPort))`
  2. Call `.Launch()` to get control URL
  3. `rod.New().ControlURL(u).Connect()`
- `Disconnect()`: close browser gracefully, update status
- `Status() string`: return current state
- `GetPage(url string) (*rod.Page, error)`: create new stealth page, navigate
- Emit Wails events on status change: `runtime.EventsEmit(ctx, "browser:status", status)`

### 4. Create `internal/chrome/stealth.go`
```go
func (bm *BrowserManager) NewStealthPage() (*rod.Page, error) {
    page, err := stealth.Page(bm.browser)
    if err != nil {
        return nil, err
    }
    // Additional overrides beyond go-rod/stealth defaults:
    // - Custom User-Agent (remove HeadlessChrome)
    // - navigator.languages
    // - navigator.platform = "Win32"
    // - navigator.hardwareConcurrency = 8
    page.MustEvalOnNewDocument(`() => {
        Object.defineProperty(navigator, 'platform', { get: () => 'Win32' });
        Object.defineProperty(navigator, 'hardwareConcurrency', { get: () => 8 });
    }`)
    return page, nil
}
```

### 5. Create `internal/chrome/session.go`
- `ExtractToken(page *rod.Page) (string, error)`:
  1. Navigate to `https://labs.google` if not already there
  2. `page.WaitStable(2 * time.Second)` — wait for JS hydration
  3. `result := page.MustEval("() => JSON.stringify(window.__NEXT_DATA__)")`
  4. Parse JSON, walk tree to find auth/bearer token
  5. Return token string
- `ValidateToken(token string) bool`: make lightweight API call, check for 401
- `RefreshToken() (string, error)`: navigate to labs.google, re-extract
- Cache token in `BrowserManager` with expiry tracking

### 6. Wails Bindings in `app.go`
```go
func (a *App) LaunchBrowser() error
func (a *App) DisconnectBrowser() error
func (a *App) GetBrowserStatus() string
func (a *App) GetAuthToken() (string, error)  // For debug display
```
- `LaunchBrowser()`: call `browserManager.Connect()` (tries existing first, then launches)
- Emit `browser:status` events so sidebar indicator updates in real-time

### 7. Browser Health Check
- Background goroutine: every 30s, ping browser via `browser.GetVersion()`
- On failure: update status to "disconnected", emit event
- Auto-reconnect logic: try `Connect()` on health check failure

## Todo

- [ ] Add rod and stealth Go dependencies
- [ ] Create ChromeConfig struct with auto-detect Chrome path
- [ ] Implement BrowserManager with Connect/Launch/Disconnect
- [ ] Implement stealth page creation with full anti-detection overrides
- [ ] Implement __NEXT_DATA__ token extraction (Fact #2)
- [ ] Add token caching with expiry-based refresh
- [ ] Create Wails bindings for browser lifecycle
- [ ] Emit browser:status events for frontend consumption
- [ ] Implement background health check goroutine
- [ ] Test: launch Chrome, navigate to labs.google, extract token

## Success Criteria

1. Chrome launches with stealth flags, `navigator.webdriver` returns `undefined`
2. Existing Chrome instance detected and reused on restart
3. Auth token extracted from `__NEXT_DATA__` successfully
4. Browser status updates reflected in frontend via Wails events
5. Chrome session persists across app restarts (user stays logged in)

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| go-rod/stealth detection by Google | High | Layer additional overrides, monitor for blocks, update stealth scripts |
| Token path in __NEXT_DATA__ changes | Medium | Make extraction path configurable or use regex fallback |
| Chrome crashes / zombie processes | Medium | Health check + process cleanup on app startup |
| Debug port conflict with other tools | Low | Configurable port in settings |

## Security Considerations

- Auth tokens held in memory only, never persisted to disk or DB
- Chrome user-data-dir contains Google session cookies — warn user in settings UI
- Debug port only binds to localhost (default Rod behavior)

## Next Steps

After Phase 2 completion, proceed to [Phase 3: Video Generation Pipeline](./phase-03-video-pipeline.md).
