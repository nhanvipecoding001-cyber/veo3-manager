# Phase 3: Video Generation Pipeline

## Context

- **Parent plan:** [plan.md](./plan.md)
- **Dependencies:** [Phase 2](./phase-02-chrome-automation.md) (Chrome automation, token extraction)
- **Research:** [researcher-02-report.md](./research/researcher-02-report.md) — Slate.js, redirect capture, API patterns

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-04-03 |
| Description | API client for submit/poll, Slate.js prompt entry via CDP, settings configuration (aspect ratio, model, output count), video download via redirect capture |
| Priority | P0 — Critical |
| Implementation Status | Not Started |
| Review Status | Pending |

## Key Insights

- **Fact #5:** Slate.js editor requires CDP `Input.insertText`. Keyboard events don't work. Submit button filtered by y > 680px.
- **Fact #6:** Settings use two different UI patterns — dropdown with `aria-haspopup="menu"` for model, `role="tab"` with `data-state` for aspect ratio/output count.
- **Fact #7:** No direct download. Open redirect URL in new Chrome tab, capture signed GCS URL from redirect chain, download via HTTP.
- **Fact #8:** Success status is `MEDIA_GENERATION_STATUS_SUCCESSFUL`, not `COMPLETED`.
- **Fact #9:** API base `aisandbox-pa.googleapis.com/v1`. Submit returns media IDs, poll every 10s, 5 min timeout. Only model `veo_3_1_t2v_fast_ultra` works. Each prompt creates 1-4 videos.
- **Fact #11:** reCAPTCHA optional, API works without it.

## Requirements

1. HTTP client for aisandbox API (submit generation, poll status)
2. Slate.js prompt entry automation via CDP Input.insertText
3. UI settings configuration (aspect ratio, model, output count) via element selectors
4. Video download via Chrome redirect capture
5. End-to-end single prompt execution (enter → submit → poll → download)

## Architecture

```
internal/pipeline/
├── api.go         # HTTP client: SubmitGeneration, PollStatus
├── prompt.go      # Slate.js automation: ClearEditor, InsertPrompt, ClickCreate
├── settings.go    # Configure aspect ratio, model, output count via selectors
└── download.go    # Redirect capture + HTTP download
```

### Single Prompt Flow
```
1. ConfigureSettings(page, aspectRatio, model, outputCount)   [settings.go]
2. InsertPrompt(page, promptText)                              [prompt.go]
3. ClickCreate(page)                                           [prompt.go]
4. mediaIDs := SubmitGeneration(token, prompt, config)         [api.go]
   OR capture media IDs from network response after clicking Create
5. Loop: PollStatus(token, mediaID) every 10s, max 5 min      [api.go]
6. For each completed video:
   a. Open redirect URL in new tab                             [download.go]
   b. Capture final GCS signed URL from redirect
   c. Download via HTTP with cookies
   d. Save to download folder
7. Update task in DB with video_paths
```

### API Endpoints (Fact #9)
```
Base: https://aisandbox-pa.googleapis.com/v1

Submit: POST /... (exact path TBD — capture from network tab)
  Headers: Authorization: Bearer {token}
  Body: { prompt, model: "veo_3_1_t2v_fast_ultra", aspectRatio, outputCount }
  Response: { mediaIds: [...], operationId: "..." }

Poll: GET /.../{operationId}
  Headers: Authorization: Bearer {token}
  Response: { status: "MEDIA_GENERATION_STATUS_SUCCESSFUL", results: [...] }
```

**Note:** Exact API paths must be captured from Chrome DevTools Network tab during manual generation. The submit may happen via browser automation (clicking Create) rather than direct API call — in which case, intercept the network request/response to get media IDs.

### Two Approaches for Submit

**Approach A — Browser Automation (Recommended):**
1. Enter prompt via CDP Input.insertText
2. Configure settings via UI selectors
3. Click Create button
4. Intercept outgoing API request to capture media IDs from response
5. Use captured IDs for polling

**Approach B — Direct API:**
1. Extract token from __NEXT_DATA__
2. Call API directly with token
3. Risk: may miss required headers/cookies that browser sends automatically

Recommend **Approach A** — more reliable, inherits all browser context.

## Related Code Files

| File | Purpose |
|------|---------|
| `internal/pipeline/api.go` | `PollStatus()` — poll with token, parse response |
| `internal/pipeline/prompt.go` | `InsertPrompt()`, `ClearEditor()`, `ClickCreate()` |
| `internal/pipeline/settings.go` | `ConfigureSettings()` — set aspect ratio, model, count |
| `internal/pipeline/download.go` | `CaptureAndDownload()` — redirect capture + save |
| `internal/chrome/browser.go` | Used for page creation, network interception |

## Implementation Steps

### 1. Create `internal/pipeline/prompt.go`

**ClearEditor(page):**
```go
// Find Slate editor contenteditable div
editor := page.MustElement("[data-slate-editor='true']")
editor.MustClick()
// Select all text
page.KeyActions().Press(input.ControlLeft).Type(input.KeyA).MustDo()
// Delete selected
page.KeyActions().Press(input.Backspace).MustDo()
```

**InsertPrompt(page, text):**
```go
// After clearing, use CDP Input.insertText (Fact #5)
editor := page.MustElement("[data-slate-editor='true']")
editor.MustClick()
// CDP command
proto.InputInsertText{Text: text}.Call(page)
```

**ClickCreate(page):**
```go
// Find all buttons with "Create" text, filter by y > 680px (Fact #5)
buttons := page.MustElements("button")
for _, btn := range buttons {
    if btn.MustText() == "Create" || strings.Contains(btn.MustText(), "Create") {
        box := btn.MustShape()
        if box.Box().Y > 680 {
            btn.MustClick()
            return
        }
    }
}
```

### 2. Create `internal/pipeline/settings.go`

**ConfigureAspectRatio(page, ratio):**
```go
// Aspect ratio uses role="tab" with data-state (Fact #6)
// Find tab matching desired ratio, click if data-state != "active"
tabs := page.MustElements("[role='tab']")
for _, tab := range tabs {
    text := tab.MustText()
    if strings.Contains(text, ratio) { // e.g. "16:9"
        state, _ := tab.Attribute("data-state")
        if state != nil && *state != "active" {
            tab.MustClick()
        }
        break
    }
}
```

**ConfigureOutputCount(page, count):**
```go
// Output count also uses role="tab" with data-state (Fact #6)
tabs := page.MustElements("[role='tab']")
for _, tab := range tabs {
    if tab.MustText() == strconv.Itoa(count) {
        state, _ := tab.Attribute("data-state")
        if state != nil && *state != "active" {
            tab.MustClick()
        }
        break
    }
}
```

**ConfigureModel(page, model):**
```go
// Model uses dropdown: button with aria-haspopup="menu" containing "crop_" (Fact #6)
dropdownBtn := page.MustElement("button[aria-haspopup='menu']")
// Check if it contains "crop_" text to confirm correct dropdown
dropdownBtn.MustClick()
// Wait for menu to appear
page.MustWaitStable()
// Select model via role="menuitem"
items := page.MustElements("[role='menuitem']")
for _, item := range items {
    if strings.Contains(item.MustText(), model) {
        item.MustClick()
        break
    }
}
```

### 3. Create `internal/pipeline/api.go`

**InterceptSubmitResponse(page):**
```go
// Set up network listener before clicking Create
// Capture the POST to aisandbox-pa.googleapis.com
router := page.HijackRequests()
var mediaIDs []string
router.MustAdd("*aisandbox-pa.googleapis.com*", func(ctx *rod.Hijack) {
    ctx.MustLoadResponse()
    if ctx.Request.Method() == "POST" {
        // Parse response body for media IDs
        var resp GenerationResponse
        json.Unmarshal([]byte(ctx.Response.Body()), &resp)
        mediaIDs = resp.MediaIDs
    }
})
go router.Run()
```

**PollStatus(token, operationID):**
```go
func (p *Pipeline) PollStatus(token, opID string) (*PollResult, error) {
    ticker := time.NewTicker(10 * time.Second)
    timeout := time.After(5 * time.Minute)
    for {
        select {
        case <-ticker.C:
            status, err := p.checkStatus(token, opID)
            if err != nil { return nil, err }
            if status.Status == "MEDIA_GENERATION_STATUS_SUCCESSFUL" {
                return status, nil  // Fact #8
            }
            if strings.Contains(status.Status, "FAILED") {
                return nil, fmt.Errorf("generation failed: %s", status.Status)
            }
            // Emit progress event
            runtime.EventsEmit(p.ctx, "task:progress", ProgressEvent{...})
        case <-timeout:
            return nil, fmt.Errorf("polling timeout after 5 minutes")
        case <-p.ctx.Done():
            return nil, p.ctx.Err()
        }
    }
}
```

### 4. Create `internal/pipeline/download.go`

**CaptureAndDownload(browser, redirectURL, destFolder):**
```go
func (p *Pipeline) CaptureAndDownload(redirectURL, destPath string) error {
    // Create new tab for redirect capture (Fact #7)
    page, err := p.browserMgr.NewStealthPage()
    if err != nil { return err }
    defer page.MustClose()

    // Enable network monitoring
    var finalURL string
    wait := make(chan struct{})

    go page.EachEvent(func(e *proto.NetworkResponseReceived) {
        // Track redirect chain — look for GCS signed URL
        url := e.Response.URL
        if strings.Contains(url, "storage.googleapis.com") ||
           strings.Contains(url, "googleusercontent.com") {
            finalURL = url
            close(wait)
        }
    })()

    // Navigate to redirect URL
    page.MustNavigate(redirectURL)

    // Wait for redirect to complete or timeout
    select {
    case <-wait:
    case <-time.After(30 * time.Second):
        // Fallback: grab current page URL
        finalURL = page.MustInfo().URL
    }

    // Download via HTTP (cookies from browser not needed for signed GCS URLs)
    resp, err := http.Get(finalURL)
    if err != nil { return err }
    defer resp.Body.Close()

    out, err := os.Create(destPath)
    if err != nil { return err }
    defer out.Close()

    _, err = io.Copy(out, resp.Body)
    return err
}
```

### 5. Create `Pipeline` Orchestrator

```go
// internal/pipeline/pipeline.go
type Pipeline struct {
    browserMgr *chrome.BrowserManager
    db         *database.DB
    ctx        context.Context
}

func (p *Pipeline) ExecuteTask(task *database.Task) error {
    page, _ := p.browserMgr.NewStealthPage()
    defer page.MustClose()

    page.MustNavigate("https://labs.google/fx/tools/video-fx")
    page.MustWaitStable()

    // 1. Configure settings
    ConfigureSettings(page, task.AspectRatio, task.Model, task.OutputCount)

    // 2. Enter prompt
    ClearEditor(page)
    InsertPrompt(page, task.Prompt)

    // 3. Click create + intercept response
    mediaIDs := InterceptAndCreate(page)

    // 4. Update task with media IDs
    p.db.UpdateTask(task.ID, map[string]interface{}{"media_ids": mediaIDs, "status": "polling"})

    // 5. Poll until complete
    token, _ := p.browserMgr.ExtractToken(page)
    result, _ := p.PollStatus(token, mediaIDs)

    // 6. Download each video
    var paths []string
    for i, videoResult := range result.Videos {
        destPath := filepath.Join(downloadFolder, fmt.Sprintf("%s_%d.mp4", task.ID, i))
        p.CaptureAndDownload(videoResult.URL, destPath)
        paths = append(paths, destPath)
    }

    // 7. Update task
    p.db.UpdateTask(task.ID, map[string]interface{}{
        "video_paths": paths, "status": "completed", "completed_at": time.Now(),
    })
    return nil
}
```

## Todo

- [ ] Implement Slate.js editor automation (clear, insert via CDP Input.insertText)
- [ ] Implement Create button click with y > 680px filter
- [ ] Implement settings configuration (aspect ratio via tabs, model via dropdown)
- [ ] Set up network request interception to capture media IDs
- [ ] Implement PollStatus with 10s interval and 5 min timeout
- [ ] Implement redirect capture for video download
- [ ] Implement HTTP download of signed GCS URLs
- [ ] Create Pipeline orchestrator combining all steps
- [ ] Test end-to-end: single prompt → video downloaded
- [ ] Handle token expiry / 401 with re-extraction

## Success Criteria

1. Prompt entered into Slate.js editor via CDP Input.insertText
2. Settings (aspect ratio, model, output count) configured via UI automation
3. Create button clicked, media IDs captured from API response
4. Polling detects `MEDIA_GENERATION_STATUS_SUCCESSFUL` correctly
5. Video downloaded via redirect capture to local filesystem
6. Full end-to-end single prompt execution succeeds

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Google Labs UI selectors change | High | Centralize selectors as constants, easy to update |
| Redirect URL format changes | High | Flexible URL pattern matching, fallback to page URL |
| API response schema changes | Medium | Loose JSON parsing, log raw responses for debugging |
| Signed GCS URL expires before download | Medium | Start download immediately after capture |
| Rate limiting on rapid sequential tasks | Medium | Configurable delay between tasks |

## Security Considerations

- Bearer tokens transmitted over HTTPS only
- Downloaded videos stored in user-specified folder
- Network interception only captures aisandbox API responses, not all traffic

## Next Steps

After Phase 3 completion, proceed to [Phase 4: Queue Management System](./phase-04-queue-management.md).
