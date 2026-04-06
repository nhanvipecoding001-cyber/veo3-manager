# Phase 3: Fix Download Flow

## Context
- [API Analysis](research/researcher-03-api-analysis.md)
- [Phase 2: Pipeline](phase-02-fix-pipeline.md)

## Overview
Simplify `download.go`. Current approach opens a new Chrome tab per download and listens for `NetworkResponseReceived`. Simpler: use browser `fetch()` in page context to follow the redirect, extract final GCS URL from `response.url`, then download via plain HTTP.

## Key Insights
- Redirect URL: `https://labs.google/fx/api/trpc/media.getMediaUrlRedirect?name={mediaId}`
- Redirect requires browser cookies (labs.google session). Cannot use raw HTTP for redirect step.
- Final GCS URL is signed -- no auth needed for actual download
- Files are ~7-8MB video/mp4
- Current `CaptureAndDownload` works but is fragile (event listener race conditions)

## Requirements
1. Replace tab-based redirect capture with `page.Evaluate(fetch())` approach
2. Extract signed GCS URL from fetch response
3. Download via HTTP (existing `downloadFile` function is fine)
4. Add retry logic for transient failures
5. Add progress reporting during download

## Architecture

### New Download Flow
```
1. page.Evaluate(`fetch(redirectURL, {redirect:'follow'}).then(r => r.url)`)
   → returns signed GCS URL string
2. downloadFile(gcsURL, destPath)  // existing HTTP download
```

This is 3 lines instead of 40. No new tab, no event listeners, no race conditions.

### Fallback
If `fetch()` fails (CORS, cookie issues), fall back to current tab-based approach.

## Related Code Files
- `veo3-manager/internal/pipeline/download.go` (SIMPLIFY)

## Implementation Steps

### Step 1: Add FetchDownloadURL function
```go
func FetchDownloadURL(page *rod.Page, mediaID string) (string, error) {
    redirectURL := fmt.Sprintf(
        "https://labs.google/fx/api/trpc/media.getMediaUrlRedirect?name=%s", mediaID)
    
    result, err := page.Evaluate(rod.Eval(fmt.Sprintf(`
        fetch("%s", {redirect: "follow", credentials: "include"})
            .then(r => r.url)
    `, redirectURL)).ByPromise())
    if err != nil {
        return "", fmt.Errorf("fetch redirect failed: %w", err)
    }
    return result.Value.Str(), nil
}
```

### Step 2: Add DownloadVideo wrapper
```go
func DownloadVideo(page *rod.Page, mediaID, destPath string) error {
    gcsURL, err := FetchDownloadURL(page, mediaID)
    if err != nil {
        return err
    }
    return downloadFile(gcsURL, destPath)
}
```

### Step 3: Add retry wrapper
Wrap DownloadVideo with 3 retries, 2s backoff. GCS signed URLs expire (~1hr) so retry quickly.

### Step 4: Keep downloadFile as-is
The existing `downloadFile` function (HTTP GET to file) is correct. No changes needed.

### Step 5: Remove CaptureAndDownload
Delete the old tab-based approach. Keep as commented fallback initially, remove after testing.

### Step 6: Remove InterceptDownloadURL from pipeline.go
This method on Pipeline struct duplicates download.go logic. Delete it.

## Todo
- [ ] Implement FetchDownloadURL using page.Evaluate + fetch()
- [ ] Implement DownloadVideo wrapper
- [ ] Add retry logic (3 attempts, 2s backoff)
- [ ] Remove old CaptureAndDownload function
- [ ] Remove InterceptDownloadURL from pipeline.go
- [ ] Test with real mediaID

## Success Criteria
- FetchDownloadURL returns valid GCS signed URL
- Downloaded file is valid .mp4 (~7-8MB)
- Retry handles transient network errors
- No orphan Chrome tabs created during download

## Risk Assessment
- **CORS blocking fetch()**: The fetch runs in page context (labs.google origin) requesting labs.google URL. Same-origin, should work. If not, fall back to tab approach.
- **Cookie expiry**: Browser session cookies may expire during long batches. Token refresh (Phase 2) should also refresh cookies via page reload.
- **Large files**: 7-8MB is small. No need for chunked download or progress bars on individual files.

## Security Considerations
- GCS signed URLs contain access signatures. Do not log full URLs. Truncate in logs.
- Downloaded files go to user-configured directory. Validate path to prevent directory traversal.

## Next Steps
Phase 4 fixes UI automation selectors (button text, VIDEO tab).
