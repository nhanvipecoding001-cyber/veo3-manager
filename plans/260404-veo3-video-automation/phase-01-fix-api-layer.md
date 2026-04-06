# Phase 1: Fix API Layer

## Context
- [API Analysis](research/researcher-03-api-analysis.md)
- [Codebase Scout](scout/scout-01-codebase.md)

## Overview
Complete rewrite of `internal/pipeline/api.go`. Current code uses `GET /v1/operations/{id}` which does not exist. Real API uses POST endpoints with structured JSON bodies. All request/response types must be redefined.

## Key Insights
- Submit sends ONE request per video (outputsPerPrompt=2 means 2 POST calls with same batchId, different seeds)
- Poll accepts multiple media IDs in single request (batch check)
- Content-Type is `text/plain;charset=UTF-8` not `application/json`
- projectId comes from `__NEXT_DATA__`, batchId is a fresh UUID per prompt batch
- reCAPTCHA token is sent but currently optional

## Requirements
1. Define correct request/response structs matching live capture
2. Implement `SubmitVideo()` - single video submission
3. Implement `SubmitBatch()` - N calls for N output videos, same batchId
4. Rewrite `PollStatus()` with correct POST endpoint and body
5. Add `CheckCredits()` for remaining credit check
6. Handle 401 token expiry gracefully

## Architecture

### New Types
```go
// Submit request
type SubmitRequest struct {
    MediaGenerationContext struct {
        BatchId string `json:"batchId"`
    } `json:"mediaGenerationContext"`
    ClientContext struct {
        ProjectId        string `json:"projectId"`
        Tool             string `json:"tool"`              // "PINHOLE"
        UserPaygateTier  string `json:"userPaygateTier"`   // "PAYGATE_TIER_NOT_PAID"
        SessionId        string `json:"sessionId"`         // ";<timestamp>"
    } `json:"clientContext"`
    Requests []VideoRequest `json:"requests"`
    UseV2ModelConfig bool `json:"useV2ModelConfig"` // true
}

type VideoRequest struct {
    AspectRatio  string     `json:"aspectRatio"`  // "VIDEO_ASPECT_RATIO_LANDSCAPE"
    Seed         int        `json:"seed"`
    TextInput    TextInput  `json:"textInput"`
    VideoModelKey string   `json:"videoModelKey"` // "veo_3_1_t2v_fast"
    Metadata     struct{}   `json:"metadata"`
}

// Submit response
type SubmitResponse struct {
    Operations []struct {
        Operation struct {
            Name string `json:"name"` // media_id
        } `json:"operation"`
        Status string `json:"status"`
    } `json:"operations"`
    RemainingCredits int `json:"remainingCredits"`
}

// Poll request
type PollRequest struct {
    Media []PollMediaItem `json:"media"`
}

// Poll response
type PollResponse struct {
    Media []struct {
        Name          string `json:"name"`
        MediaMetadata struct {
            MediaStatus struct {
                MediaGenerationStatus string `json:"mediaGenerationStatus"`
            } `json:"mediaStatus"`
        } `json:"mediaMetadata"`
    } `json:"media"`
}
```

### Key Functions
```
SubmitVideo(token, projectId, prompt, aspectRatio, seed string) (*SubmitResponse, error)
SubmitBatch(token, projectId, prompt, aspectRatio string, count int) ([]string, error)  // returns mediaIDs
PollStatus(ctx, token, projectId string, mediaIDs []string, onProgress) (*PollResult, error)
```

## Related Code Files
- `veo3-manager/internal/pipeline/api.go` (REWRITE)
- `veo3-manager/internal/chrome/session.go` (token + projectId extraction)

## Implementation Steps

### Step 1: Define all request/response structs
Match exactly to captured API payloads. Use `text/plain;charset=UTF-8` content type.

### Step 2: Implement SubmitVideo
- POST to `{apiBase}/video:batchAsyncGenerateVideoText`
- Set headers: Authorization, Content-Type, Referer
- Parse response, extract `operations[0].operation.name` as mediaId
- Return remaining credits

### Step 3: Implement SubmitBatch
- Generate one UUID for batchId (shared across all videos from same prompt)
- Loop `count` times, each with random seed via `math/rand`
- Call SubmitVideo for each, collect mediaIDs
- Return all mediaIDs

### Step 4: Rewrite PollStatus
- POST to `{apiBase}/video:batchCheckAsyncVideoGenerationStatus`
- Body: `{"media": [{"name": "<id>", "projectId": "<pid>"}, ...]}`
- Check each media item's `mediaMetadata.mediaStatus.mediaGenerationStatus`
- Return when ALL are SUCCESSFUL or any FAILED
- Keep 10s interval, 5min timeout

### Step 5: Add helper - aspectRatio mapping
- Map user-facing `"16:9"` to `"VIDEO_ASPECT_RATIO_LANDSCAPE"`
- Map `"9:16"` to `"VIDEO_ASPECT_RATIO_PORTRAIT"`

### Step 6: Extract projectId from __NEXT_DATA__
- Update `chrome/session.go` to also extract projectId alongside token
- Store both in BrowserManager

## Todo
- [ ] Define SubmitRequest, SubmitResponse structs
- [ ] Define PollRequest, PollResponse structs
- [ ] Implement SubmitVideo with correct endpoint + headers
- [ ] Implement SubmitBatch (loop with shared batchId, random seeds)
- [ ] Rewrite PollStatus with POST + correct body
- [ ] Add aspectRatio string mapping helper
- [ ] Update session.go to extract projectId from __NEXT_DATA__
- [ ] Add unit tests with mock HTTP responses

## Success Criteria
- SubmitVideo returns valid mediaId from real API
- PollStatus correctly detects SUCCESSFUL/FAILED/PENDING
- SubmitBatch returns N mediaIDs for N count
- 401 errors trigger token refresh path

## Risk Assessment
- **reCAPTCHA**: Currently omitted from request. If Google enforces it, we need to extract reCAPTCHA token from page context or fall back to UI click submission.
- **Rate limiting**: Rapid SubmitBatch calls may trigger throttling. Add 500ms delay between calls if needed.
- **API versioning**: `useV2ModelConfig: true` may change. Keep as constant, easy to update.

## Security Considerations
- Bearer token is sensitive. Never log full token. Store in memory only, never persist to disk.
- projectId is per-user. Same memory-only treatment.

## Next Steps
Phase 2 will integrate these new API functions into pipeline.go orchestration.
