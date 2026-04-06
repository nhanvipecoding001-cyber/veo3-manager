# Veo3 Batch Video Generator — Research Report
**Date:** 2026-04-03 | **Focus:** Implementation patterns & gotchas

---

## 1. Google Labs Veo Video Generation API

### Current Landscape
- **Veo 3 availability:** Available on Vertex AI (official) and Gemini API (structured access)
- **aisandbox-pa.googleapis.com:** Google AI sandbox backend for inference. Handles video generation operations
- **Auth patterns:** Vertex AI uses OAuth 2.0; Gemini API uses API key OR OAuth 2.0
- **Video specs:** Up to 4K, 16:9 or 9:16 aspect ratio, 4/6/8 sec duration with audio

### API Access Routes
1. **Vertex AI (official):** `us-central1-aiplatform.googleapis.com/v1/projects/*/locations/us-central1/publishers/google/models/*:predictLongRunning`
2. **Gemini API (documented):** `generativelanguage.googleapis.com/v1beta/`
3. **Labs playground:** Likely uses proprietary internal endpoint; auth via `__NEXT_DATA__` token extraction

### __NEXT_DATA__ Auth Extraction
- Next.js pages embed auth tokens in page HTML
- Look for `window.__NEXT_DATA__.props.pageProps` or similar structures
- Contains session/bearer tokens for API calls
- Gotcha: Tokens expire; need refresh mechanism

### Polling Pattern
- Long-running operations return operation IDs
- Status values include: `MEDIA_GENERATION_STATUS_SUCCESSFUL`, `MEDIA_GENERATION_STATUS_PENDING`, `MEDIA_GENERATION_STATUS_FAILED`
- Poll status endpoint periodically until terminal state
- Typical polling: 2-5 second intervals to avoid rate limiting

---

## 2. Chrome Automation Anti-Detection

### Key Evasion Techniques
- **`--disable-blink-features=AutomationControlled`:** Prevents setting `navigator.webdriver = true`. Critical baseline
- **navigator.webdriver override:** Set to `undefined` or `false` via JS injection after launch
- **CDP approach:** Use `Runtime.evaluate` to inject stealth scripts before page navigation

### Stealth Plugin Patterns (go-rod/stealth reference)
- Override `navigator.webdriver`
- Remove `HeadlessChrome` from User-Agent (common detection vector)
- Patch `navigator.permissions.query()` to avoid detection
- Override `chrome` object properties
- Disable WebGL fingerprinting vectors
- Spoof timezone/locale/screen dimensions

### Critical Gotchas
1. **HeadlessChrome in User-Agent still detectable** — need custom User-Agent override
2. **WebGL fingerprinting:** Chrome exposes UNMASKED_RENDERER/VENDOR. Inject WebGL spoofing
3. **Timing attacks:** Real users have variable delays; add random delays between actions
4. **Navigator properties:** Anti-bot systems check plugins, languages, hardwareConcurrency
5. **go-rod/stealth not a silver bullet** — works for most, not all anti-bot systems (DataDome, Distil known to have detection methods beyond WebDriver)

### Recommended Approach
- Use `--disable-blink-features=AutomationControlled` + stealth script injection
- Override User-Agent header before page load
- Test against actual Google Labs playground to validate

---

## 3. Slate.js Editor Automation

### Why Keyboard Events Fail
- Slate intercepts onKeyDown/onKeyUp at React event level
- Direct DOM keyboard event dispatch doesn't trigger Slate's event handlers
- Framework-level state mutations ignored by Slate's change tracking

### CDP Input.insertText Solution
- **Protocol method:** `Input.insertText(text)` via Chrome DevTools Protocol
- Operates at Chromium layer, bypasses React/Slate interception
- Behaves closer to real typing; Slate receives synthetic input events
- More reliable than DOM manipulation or JS `insertText()` calls

### Best Practices for Slate Automation
1. **Clear text:** Select all (Ctrl+A) → CDP Input.insertText("")
2. **Set text:** Clear first, then Input.insertText(newText)
3. **Insert without clearing:** Use Input.insertText(text) at cursor position
4. **Dispatch beforeinput event:** Create InputEvent with type="beforeinput", inputType="insertText"
5. **Avoid setState races:** Ensure editor state settled before next operation

### Gotchas
- CDP Input.insertText doesn't fire all Slate event handlers; use `Runtime.evaluate` to trigger custom handlers if needed
- Selection state matters; cursor position affects insertion point
- Slate's change tracking may not recognize CDP insertions as "user input" depending on implementation

---

## 4. Video Download via Chrome Redirect Capture

### Pattern: Signed GCS URL Redirect
1. Video generation endpoint returns temp signed URL (short TTL)
2. Direct Bearer token requests fail (Google-served media requires session context)
3. Must open URL in browser tab to trigger redirect chain
4. Tab eventually redirects to final signed Google Cloud Storage URL
5. Capture redirect via CDP Network.requestWillBeSent → final URL
6. Download from final signed URL with session cookies

### Why Direct HTTP Fails
- Google serves media through signed URLs with session validation
- Bearer token auth insufficient; requires HttpOnly cookies from authenticated session
- Direct HTTP request lacks session context; gets 403/401

### CDP Redirect Capture
- Enable `Network.enable()` domain
- Monitor `Network.responseReceived` events
- Track URL changes in redirect chain: status 301/302/303/307/308
- Final URL typically has `gsutil/gcs-signed-url` patterns or similar
- Extract final URL from response headers (Location) or next request URL

### Implementation Pattern
1. Create headless tab
2. Enable Network domain
3. Navigate to signed URL
4. Wait for redirect chain completion
5. Extract final GCS URL from last response
6. Use separate HTTP client with cookies to download

### Gotchas
- **Timeout:** Signed URLs expire; must capture before expiry (usually 15-30 min)
- **Multiple redirects:** Track intermediate redirects; don't assume single hop
- **Cookie stripping:** Some proxies/middleware strip cookies on redirects
- **Tab session persistence:** Must keep session tab alive or reuse cookies from it
- **Download size limits:** Don't rely on Content-Length; stream download

---

## 5. Queue Management in Go

### Goroutine Queue with Pause/Resume/Stop

```go
type Queue struct {
    jobChan    chan Job
    ctx        context.Context
    cancel     context.CancelFunc
    pauseChan  chan struct{}
    resumeChan chan struct{}
}

// Worker goroutine selects on:
// - jobChan: incoming jobs
// - pauseChan: pause signal
// - ctx.Done(): stop signal
// - resumeChan: resume after pause
```

### Channel Patterns
- **Stop:** Close jobChan or invoke cancel() on context
- **Pause:** Send to pauseChan, worker blocks on select
- **Resume:** Send to resumeChan, worker resumes job processing
- **Status polling:** Separate statusChan for queue length/state queries

### Config Changes Without Restart
- Don't mutate config directly in running worker
- Pattern: Send "reconfigure" message to configChan
- Worker reads new config from message
- Applies changes on next job iteration
- No need for locks if messages pass config atomically

### Implementation Principles
1. **Worker loop:** Single goroutine per queue (no goroutine-per-job unless scaled)
2. **Context for cancellation:** Use context.WithCancel for graceful shutdown
3. **WaitGroup for completion:** Track pending jobs before shutdown
4. **Buffered channels:** jobChan with buffer avoids blocking producers
5. **Channel ownership:** Sender closes channels (worker reads only)

### Gotchas
1. **Panic on closed channel:** Never close jobChan from worker
2. **Goroutine leaks:** Ensure all select branches unblock on stop
3. **Config race:** Pass config via message, not shared reference
4. **WaitGroup misuse:** Add before send, Done after job complete
5. **Unbuffered channels:** Can deadlock if sender expects immediate pickup

---

## Summary & Unresolved Questions

**Implemented:** Auth extraction, anti-detection stealth, Slate automation via CDP, signed URL capture, Go queue patterns

**Key Dependencies:** Chrome DevTools Protocol library (go-rod, puppeteer, etc), Vertex AI SDK or custom HTTP client for Veo API, session management for Google Labs auth persistence

**Unresolved Questions:**
- How to extract auth from Google Labs __NEXT_DATA__ without hardcoding selectors (page structure stability)?
- Does Google Labs expose generation API endpoint directly or only via internal sandbox?
- Exact rate limits on aisandbox-pa.googleapis.com for batch generation
- Minimum polling interval before getting 429 Too Many Requests?
- Session TTL for Google Labs auth tokens and refresh strategy?

---

## Sources

- [Veo on Vertex AI video generation API](https://docs.cloud.google.com/vertex-ai/generative-ai/docs/model-reference/veo-video-generation)
- [Generate videos with Veo 3.1 in Gemini API](https://ai.google.dev/gemini-api/docs/video)
- [Veo 3 API Guide for Developers](https://www.veo3ai.io/blog/veo-3-api-guide-developers-2026)
- [How to Avoid Bot Detection With Selenium](https://www.zenrows.com/blog/selenium-avoid-bot-detection)
- [Using --disable-blink-features=AutomationControlled](https://www.zenrows.com/blog/disable-blink-features-automationcontrolled)
- [Undetected ChromeDriver](https://github.com/ultrafunkamsterdam/undetected-chromedriver)
- [How to Modify Selenium navigator.webdriver](https://www.zenrows.com/blog/navigator-webdriver)
- [Slate.js Adding Event Handlers](https://docs.slatejs.org/walkthroughs/02-adding-event-handlers)
- [Handle keyboard events with slate js](https://egghead.io/lessons/react-handle-keyboard-events-with-slate-js)
- [Slate editor insertText discussion](https://github.com/ianstormtaylor/slate/issues/2549)
- [Chrome DevTools Protocol](https://chromedevtools.github.io/devtools-protocol/)
- [Job Queues in Go](https://www.opsdash.com/blog/job-queues-in-go.html)
- [Golang Concurrency patterns](https://medium.com/openskill/golang-concurrency-patterns-with-goroutine-channel-and-wait-group-661374915b22)
