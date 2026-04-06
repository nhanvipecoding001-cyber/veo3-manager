# Phase 5: Queue & Frontend Polish

## Context
- [Phase 2: Pipeline](phase-02-fix-pipeline.md)
- [Codebase Scout](scout/scout-01-codebase.md)

## Overview
Minor fixes to queue.go for multi-video task handling, credit tracking, and error recovery. Frontend updates for correct status display and credit remaining indicator.

## Key Insights
- Queue worker loop is solid. Main fix: pass correct data to updated pipeline.
- Credit tracking: submit response returns `remainingCredits`. Should display in UI and pause queue when credits < 20.
- Each video costs 20 credits. Queue should pre-check before submitting.
- Task model stores `outputCount` but pipeline must translate this to N API calls.

## Requirements
1. Add credit tracking to queue/pipeline
2. Auto-pause queue when credits insufficient
3. Update task status events to include mediaIDs and video count
4. Add error categorization (retryable vs fatal)
5. Frontend: show remaining credits, per-task progress with video count
6. Frontend: fix any broken Wails bindings from API changes

## Architecture

### Credit Tracking
```go
// Pipeline returns remaining credits after submit
type SubmitResult struct {
    MediaIDs         []string
    RemainingCredits int
}
```
Queue checks `remainingCredits < 20 * nextTask.OutputCount` before processing. If insufficient, emit `queue:credits_low` event and pause.

### Error Categories
- **Retryable**: network timeout, 5xx errors, token expiry (auto-refresh)
- **Fatal**: 403 forbidden, content policy violation, account suspended
- Queue retries retryable errors up to 3 times with exponential backoff. Fatal errors mark task as failed immediately.

### Frontend Events (existing, verify correct)
- `task:progress` -- {taskId, status, message}
- `task:started` -- {taskId, prompt}
- `task:completed` -- {taskId}
- `task:failed` -- {taskId, error}
- `queue:state` -- "idle"|"running"|"paused"|"stopping"
- `queue:stats` -- TaskStats object
- NEW: `queue:credits` -- {remaining: int}

## Related Code Files
- `veo3-manager/internal/queue/queue.go` (MINOR FIXES)
- `veo3-manager/app.go` (ADD credit binding)
- `veo3-manager/frontend/src/pages/Queue.tsx` (UPDATE)
- `veo3-manager/frontend/src/pages/Dashboard.tsx` (UPDATE)
- `veo3-manager/frontend/src/stores/queueStore.ts` (ADD credits)

## Implementation Steps

### Step 1: Add credit tracking to pipeline
After SubmitBatch, emit `queue:credits` event with remaining count. Store in queue manager state.

### Step 2: Add pre-submit credit check
Before `pipeline.ExecuteTask`, check if stored credits >= required (20 * outputCount). If not, pause queue and emit warning.

### Step 3: Add retry logic to queue worker
```go
maxRetries := 3
for attempt := 0; attempt <= maxRetries; attempt++ {
    err := m.pipeline.ExecuteTask(&task)
    if err == nil { break }
    if !isRetryable(err) {
        m.db.UpdateTaskError(task.ID, err.Error())
        break
    }
    time.Sleep(time.Duration(1<<attempt) * time.Second)
}
```

### Step 4: Update frontend Queue.tsx
- Show remaining credits in header
- Show per-task video count (e.g., "2/2 videos generating")
- Add credit warning banner when low

### Step 5: Update frontend Dashboard.tsx
- Add credits remaining card
- Show total videos generated (not just tasks)

### Step 6: Update Wails bindings in app.go
- Add `GetCredits() int` binding if needed
- Verify all existing bindings still match updated pipeline signatures

### Step 7: Update TypeScript types
Ensure `frontend/src/types/index.ts` matches any new Go struct changes from Phases 1-4.

## Todo
- [ ] Add credit tracking after submit
- [ ] Add pre-submit credit check with auto-pause
- [ ] Add retry logic with error categorization
- [ ] Update Queue.tsx with credits display
- [ ] Update Dashboard.tsx with credits card
- [ ] Update TypeScript types for new event payloads
- [ ] Verify all Wails bindings compile
- [ ] End-to-end test: queue 3 prompts, verify all complete

## Success Criteria
- Queue pauses automatically when credits insufficient
- Retryable errors are retried (up to 3x)
- Fatal errors immediately mark task as failed
- Frontend shows accurate credit count
- Full queue run completes without manual intervention

## Risk Assessment
- **Credit count accuracy**: `remainingCredits` from API may be stale if user generates videos outside the app. Refresh on queue start.
- **Frontend event flood**: Rapid status updates may cause UI jank. Debounce events in frontend (100ms).

## Security Considerations
- Credit count is not sensitive but should not be exposed outside the app.
- Error messages from API may contain user data. Sanitize before displaying in UI.

## Next Steps
After all 5 phases complete, do full integration testing:
1. Launch browser, connect
2. Queue 2 prompts with outputCount=2
3. Verify 4 videos downloaded
4. Verify credits updated
5. Test pause/resume mid-queue
