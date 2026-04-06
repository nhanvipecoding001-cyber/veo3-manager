# Phase 2: Fix Pipeline Orchestration

## Context
- [API Analysis](research/researcher-03-api-analysis.md)
- [Phase 1: API Layer](phase-01-fix-api-layer.md)

## Overview
Rewrite `pipeline.go` to use the hybrid approach: UI automation for page setup + token extraction, direct API calls for submit/poll. Remove `interceptAndCreate` (network hijack approach). Fix navigation URL. Add projectId to pipeline state.

## Key Insights
- Current code navigates to `video-fx` -- must be `labs.google/fx/vi/tools/flow`
- `interceptAndCreate` hijacks network and parses wrong response structure -- replace with direct API call
- Pipeline needs projectId (from `__NEXT_DATA__`) in addition to token
- Each task should reuse the same page if possible (avoid re-navigation per task)
- Token and projectId should be refreshed if stale (>30min)

## Requirements
1. Fix navigation URL to `https://labs.google/fx/vi/tools/flow`
2. Extract both token AND projectId from page
3. Replace interceptAndCreate with direct SubmitBatch API call
4. Update PollStatus call with new signature (mediaIDs + projectId)
5. Update download flow to use mediaID-based redirect URLs
6. Add page reuse between tasks (navigate once, reuse for multiple prompts)

## Architecture

### Updated ExecuteTask Flow
```
1. EnsurePage() -- create/reuse stealth page, navigate if needed
2. ExtractAuth() -- get token + projectId from __NEXT_DATA__
3. ConfigureSettings() -- open dropdown, click VIDEO, set aspect ratio + count
4. ClearEditor() + InsertPrompt() -- enter prompt text
5. ClickCreate() -- UI click triggers reCAPTCHA naturally
6. WaitForSubmission() -- intercept API response to get mediaIDs (NEW)
7. PollStatus() -- direct API poll with mediaIDs + projectId
8. DownloadVideos() -- for each mediaID, get redirect URL, download
9. Update task status in DB
```

**Decision point (Step 5-6)**: Two options for submission:
- **Option A (Recommended)**: Click "Tao" button via UI, intercept the outgoing API response to capture mediaIDs. This naturally handles reCAPTCHA.
- **Option B**: Direct API call with SubmitBatch. Faster but risks reCAPTCHA block.

We use **Option A** for submit (UI click + intercept response) but fix the response parsing to match real API structure. Then use direct API for poll+download.

### Pipeline Struct Changes
```go
type Pipeline struct {
    browserMgr  *chrome.BrowserManager
    db          *database.DB
    ctx         context.Context
    downloadDir string
    activePage  *rod.Page  // reusable page across tasks
    projectId   string     // extracted from __NEXT_DATA__
}
```

## Related Code Files
- `veo3-manager/internal/pipeline/pipeline.go` (REWRITE)
- `veo3-manager/internal/pipeline/api.go` (from Phase 1)
- `veo3-manager/internal/chrome/session.go` (projectId extraction)

## Implementation Steps

### Step 1: Fix navigation URL
Change `labs.google/fx/tools/video-fx` to `labs.google/fx/vi/tools/flow`.

### Step 2: Add projectId extraction
Update `chrome/session.go` to parse `__NEXT_DATA__` for both:
- `props.pageProps.session.access_token` (existing)
- `props.pageProps.session.projectId` or similar field (need to confirm exact path)

Alternative: projectId appears in submit response `workflows[].projectId`. Could extract from first intercepted response.

### Step 3: Rewrite interceptAndCreate
Fix the network hijack response parsing:
```go
var resp SubmitResponse  // from Phase 1
// Parse operations[].operation.name as mediaIDs
for _, op := range resp.Operations {
    mediaIDs = append(mediaIDs, op.Operation.Name)
}
```
Keep the UI click approach but fix what we extract from the response.

### Step 4: Update PollStatus integration
```go
result, err := PollStatus(ctx, token, projectId, mediaIDs, onProgress)
```
Pass all mediaIDs for batch polling. PollStatus returns when all complete.

### Step 5: Update download integration
For each completed mediaID:
```go
redirectURL := fmt.Sprintf("https://labs.google/fx/api/trpc/media.getMediaUrlRedirect?name=%s", mediaID)
```
Pass to download function (Phase 3).

### Step 6: Add page reuse
- Store `activePage` on Pipeline struct
- On first task: create page, navigate, wait for load
- On subsequent tasks: check if page still valid, reuse if so
- Clear editor + insert new prompt without re-navigating
- Re-extract token if >30min old

### Step 7: Handle multi-video response parsing
With UI click + outputCount=2, Google sends 2 separate POST requests. The interceptor must capture ALL responses (not just first). Use a counter or timeout-based collection:
```go
// Collect responses until we have `outputCount` mediaIDs or 30s timeout
for len(mediaIDs) < outputCount {
    select { case id := <-mediaCh: ... }
}
```

## Todo
- [ ] Fix navigation URL
- [ ] Extract projectId from __NEXT_DATA__ (update session.go)
- [ ] Rewrite interceptAndCreate with correct response parsing
- [ ] Collect multiple mediaIDs when outputCount > 1
- [ ] Update PollStatus call signature
- [ ] Build download URLs from mediaIDs
- [ ] Add page reuse logic
- [ ] Add token freshness check (>30min = re-extract)
- [ ] Update emitProgress messages

## Success Criteria
- Pipeline navigates to correct Flow URL
- Token and projectId extracted successfully
- All mediaIDs captured from UI-triggered submission
- Poll completes and triggers download for each video
- Page reuse works across multiple tasks in queue

## Risk Assessment
- **Page reuse staleness**: Page may become disconnected. Add health check before reuse, recreate if stale.
- **Multiple response capture**: Timing-sensitive. The 2 POST calls happen ~100ms apart. Need robust channel-based collection.
- **Token refresh during long batch**: 50+ video queue may span hours. Must re-extract token periodically.

## Security Considerations
- projectId is user-specific. Memory only, never logged or persisted.
- Active page reference must be cleaned up on shutdown.

## Next Steps
Phase 3 fixes download.go to work with the new mediaID-based flow.
