# Rod & Stealth: Chrome CDP Automation Research
**2026-04-04** | Focus: Google Labs Flow automation (Slate.js, token auth, file downloads)

## 1. JavaScript Evaluation in Page Context

**Rod Pattern:**
```go
// Simple eval with return value
result, err := page.Eval(`document.title`)

// Extract token from __NEXT_DATA__
token, err := page.Eval(`window.__NEXT_DATA__.props.initialState.auth.token`)

// Complex eval with args
data, err := page.Eval(`(selector) => document.querySelector(selector).textContent`, "#my-id")
```

**Key Methods:**
- `page.Eval(js)` - Execute JS, return interface{}
- `page.EvalOnNewDocument(js)` - Run on every frame creation
- `page.EvalExpression(js)` - Raw runtime evaluation
- Return values: automatically unmarshaled to Go types

**For Slate.js:** Directly eval to check editor state, trigger commands, or verify text insertion.

---

## 2. Network Request Interception (CDP Pattern)

**Rod HijackRequests (High-Level):**
```go
router := browser.HijackRequests()
router.MustAdd("**/api/**", func(ctx *rod.Hijack) {
    // Read request
    fmt.Println(ctx.Request.URL())
    
    // Modify headers (e.g., add bearer token)
    ctx.Request.Req().Header.Set("Authorization", "Bearer "+token)
    
    // Continue with modified request
    ctx.LoadResponse(http.DefaultClient, true)
})
go router.Run()
```

**Direct CDP Protocol (requestWillBeSent via proto):**
```go
// Listen to Network.requestWillBeSent events
go func() {
    for range page.EachEvent() {
        if ev := page.GetEvent().(proto.NetworkRequestWillBeSent); ev != nil {
            fmt.Println("URL:", ev.Request.URL)
            fmt.Println("Headers:", ev.Request.Headers)
        }
    }
}()
```

**Best Practice:** Use HijackRequests for request/response modification; use proto events for monitoring-only scenarios. Rod's decode-on-demand architecture optimizes heavy network event handling.

---

## 3. Page Navigation, Wait, and Click

**Core Patterns:**
```go
// Navigate with timeout
page := browser.MustPage("https://example.com")
page.Timeout(10 * time.Second).MustWaitLoad()

// Wait for element, auto-scrolls & waits until clickable
elem := page.MustElement(".my-button")
elem.MustClick()

// Wait for page reload after click
wait := page.WaitRequestIdle(3 * time.Second)
elem.MustClick()
wait()

// Wait for element to appear
page.MustWaitElement("#loaded-content", 5 * time.Second)

// Handle navigation
page.MustNavigate("https://example.com").MustWaitStable()
```

**For Google Labs Flow:**
- Use `MustWaitStable()` after navigation (waits for DOM stability)
- Use `WaitRequestIdle()` before clicking to capture all post-click network activity
- Combine `MustWaitElement()` with input field locators before typing

---

## 4. Stealth & Bot Detection Bypass

**go-rod/stealth** wraps puppeteer-extra stealth-evasions. Key bypasses:

```go
import "github.com/go-rod/stealth"

// Apply stealth evasions
stealth.MustPassStealthTest(page)
```

**Evasions applied:**
- Sets `navigator.webdriver = undefined` (overrides default true value)
- Hides automation indicators in DevTools
- Spoofs plugin list (prevents `navigator.plugins` detection)
- Removes WebDriver from user agent
- Masks headless browser attributes

**Limitations:** More sophisticated anti-bot (e.g., DataDome, PerimeterX) may require additional:
- Rotating proxy headers
- Real-world User-Agent strings
- Behavioral timing simulation
- Canvas/WebGL fingerprint randomization

**For Google Labs:** Stealth alone may suffice if no advanced bot detection; monitor for false positives.

---

## 5. Extract Cookies & Auth Tokens

**Session Persistence:**
```go
// Get cookies after login
cookies, _ := browser.Cookies()
for _, c := range cookies {
    fmt.Println(c.Name, "=", c.Value)
}

// Save to file (JSON)
data, _ := json.Marshal(cookies)
ioutil.WriteFile("cookies.json", data, 0644)

// Restore in next run
var savedCookies []*proto.NetworkCookie
json.Unmarshal(readCookies(), &savedCookies)
browser.SetCookies(savedCookies...)
```

**Extract Token from __NEXT_DATA__:**
```go
// Direct eval + unmarshal
var nextData map[string]interface{}
err := page.Eval(`window.__NEXT_DATA__`, &nextData)
// Navigate nested: nextData["props"]["initialState"]["auth"]["token"]

// Or simpler
token, _ := page.Eval(`window.__NEXT_DATA__.props.initialState.auth.token`)
```

**Store for later requests:**
- Save to secure file or env var
- Inject via `SetCookies()` or HijackRequests header modification
- Refresh token logic: intercept 401 responses, trigger re-auth, continue

---

## 6. Open Tabs, Handle Redirects, Download Files

**New Tabs:**
```go
// Open in background
newPage := browser.MustPage("")
newPage.MustNavigate("https://example.com")

// Get all pages
pages := browser.Pages()
```

**Redirect Handling:**
```go
// Wait for redirect completion
page.MustNavigate("https://example.com")
wait := page.WaitRequestIdle(5 * time.Second)
page.MustWaitLoad()
wait()

// Check final URL
finalURL := page.MustInfo().URL
```

**File Downloads:**
```go
// Wait for download before triggering it
wait := browser.WaitDownload()
page.MustElement("a.download-btn").MustClick()

// Capture file
file := wait() // Returns *proto.Page{URL}
// Download URL can be processed by client code

// Or alternative: HijackRequests to intercept file
router := browser.HijackRequests()
router.MustAdd("**/download/**", func(ctx *rod.Hijack) {
    ctx.LoadResponse(http.DefaultClient, true)
    // Save ctx.Response.Body to disk
})
```

---

## 7. Session Management & User Data Directory

**Reuse Login Sessions:**
```go
// First run: login and save
browser := rod.New().
    MustLaunch(
        launcher.New().
            UserDataDir("/path/to/user-profile").
            Headless(false), // See login happen
    ).
    MustConnect()
page := browser.MustPage("https://example.com/login")
// ... login manually or automate ...
browser.MustClose()

// Subsequent runs: reuse cookies
browser := rod.New().
    MustLaunch(
        launcher.New().
            UserDataDir("/path/to/user-profile"),
    ).
    MustConnect()
page := browser.MustPage("https://example.com")
// Already logged in
```

**Avoid Session Issues (Linux):**
- Remove `--password-store=basic` flag if session corrupts
- Use absolute paths for UserDataDir
- Ensure single browser instance per UserDataDir

**For Google Labs:** Save authenticated UserDataDir after first successful login; reuse in automation scripts.

---

## 8. Connect to Existing Chrome vs. Launch New

**Launch New Instance:**
```go
browser := rod.New().MustLaunch().MustConnect()
defer browser.MustClose()
```

**Connect to Running Instance:**
```go
// Terminal 1: Start Chrome with debugging
// chromium --remote-debugging-port=9222

// Terminal 2: Go code
url := "ws://127.0.0.1:9222/devtools/browser/..."
browser := rod.New().MustConnect(url)
// No MustClose needed (shares Chrome instance)
```

**Use Existing User Chrome:**
```go
browser := rod.New().
    MustLaunch(
        launcher.New().UserMode(),
    ).
    MustConnect()
// Acts like browser extension, reuses default profile
```

**Comparison:**
| Approach | Pros | Cons |
|----------|------|------|
| Launch new | Isolated, clean state | Slower startup, resource overhead |
| Connect existing | Fast, reuse existing tabs | Shares browser state, debugging conflicts |
| User mode | Real profile with extensions | Slower, unpredictable state |

---

## 9. Direct CDP Protocol via Proto

**When to use:** Rod lacks a feature (rare)

```go
// Example: Enable Network.requestWillBeSent logging
err := proto.NetworkEnable{}.Call(page)

// Listen to events
go func() {
    for evt := range page.EachEvent() {
        if event, ok := evt.(*proto.NetworkRequestWillBeSent); ok {
            fmt.Println("Intercepted:", event.Request.URL)
        }
    }
}()

// Disable WebDriver detection at CDP level
proto.PageEvaluateOnNewDocument{
    Source: `Object.defineProperty(navigator, 'webdriver', {get: () => false})`,
}.Call(page)
```

---

## Practical Pattern for Google Labs Flow

**Pseudocode:**
```go
// 1. Launch with stealth
b := rod.New().MustLaunch().MustConnect()
stealth.MustPassStealthTest(p)

// 2. Navigate & extract token
p := b.MustPage("https://labs.google.com/flow")
p.MustWaitLoad()
token, _ := p.Eval(`window.__NEXT_DATA__.props.initialState.auth.token`)

// 3. Hijack API calls with token
router := b.HijackRequests()
router.MustAdd("**/api/**", func(ctx *rod.Hijack) {
    ctx.Request.Req().Header.Set("Authorization", "Bearer "+token.(string))
    ctx.LoadResponse(http.DefaultClient, true)
})
go router.Run()

// 4. Interact with Slate editor
p.MustElement("[contenteditable]").MustInput("prompt text")
// Or use CDP Input.insertText for complex edits

// 5. Trigger download
wait := b.WaitDownload()
p.MustElement("a[href*=download]").MustClick()
fileURL := wait().URL

// 6. Save session
cookies, _ := b.Cookies()
saveSession(cookies)
```

---

## Key Takeaways

1. **Rod = High-level CDP wrapper**: Use for 95% of tasks; drop to proto/CDP only when needed
2. **Stealth works but not foolproof**: Additional measures (proxies, timing) for advanced detection
3. **Token extraction via Eval**: Simplest pattern; cache & reuse across requests
4. **Network interception**: Use HijackRequests for auth injection; proto events for monitoring
5. **Session persistence**: UserDataDir > manual cookie save/restore for complex auth flows
6. **File downloads**: Browser.WaitDownload() is the idiomatic pattern

---

## Unresolved Questions

- Does Google Labs Flow use specific anti-bot detection (DataDome, etc.)? Stealth may not suffice.
- Slate.js integration: Is direct JS input sufficient or does editor require CDP Input.insertText?
- Bearer token location: Confirmed __NEXT_DATA__ contains it, but location varies per Next.js config.
