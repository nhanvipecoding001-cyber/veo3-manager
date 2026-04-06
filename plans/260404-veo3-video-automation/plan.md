# Veo3 Manager - Implementation Plan

## Problem
The Go+Wails desktop app automates batch video generation from Google Labs Flow. The API layer (api.go) uses wrong endpoints/structures, pipeline has incorrect URL/selectors, and download flow is overcomplicated. All five pipeline files need fixes based on live API capture data.

## Architecture Decision: Hybrid Approach
- **UI automation**: Navigate to Flow page, extract Bearer token + projectId from `__NEXT_DATA__`, configure settings (VIDEO tab, aspect ratio, output count)
- **Direct API calls**: Submit videos via `POST /v1/video:batchAsyncGenerateVideoText`, poll via `POST /v1/video:batchCheckAsyncVideoGenerationStatus`
- **Download**: Browser `fetch()` to get signed GCS URL from redirect, then plain HTTP download

This avoids reCAPTCHA issues (UI click triggers it naturally) while keeping poll/download fast and reliable.

## Phases

| # | Phase | Files Changed | Effort |
|---|-------|--------------|--------|
| 1 | [Fix API Layer](phase-01-fix-api-layer.md) | `api.go` | High |
| 2 | [Fix Pipeline](phase-02-fix-pipeline.md) | `pipeline.go` | High |
| 3 | [Fix Download](phase-03-fix-download.md) | `download.go` | Medium |
| 4 | [Fix UI Automation](phase-04-fix-ui-automation.md) | `prompt.go`, `settings.go` | Medium |
| 5 | [Queue & Frontend Polish](phase-05-queue-frontend-polish.md) | `queue.go`, frontend | Low |

## Execution Order
Phases 1-4 are sequential (each depends on prior). Phase 5 can start after Phase 2.

## Key Files
- `veo3-manager/internal/pipeline/api.go` - API client (rewrite)
- `veo3-manager/internal/pipeline/pipeline.go` - Orchestrator (rewrite)
- `veo3-manager/internal/pipeline/download.go` - Download (simplify)
- `veo3-manager/internal/pipeline/prompt.go` - Prompt input (fix selectors)
- `veo3-manager/internal/pipeline/settings.go` - Settings UI (add VIDEO tab)
- `veo3-manager/internal/queue/queue.go` - Queue manager (minor fixes)
- `veo3-manager/app.go` - Wails bindings (minor additions)

## Success Criteria
1. Submit video via API returns valid media IDs
2. Poll detects SUCCESSFUL status correctly
3. Videos download as valid .mp4 files
4. Queue processes multiple prompts end-to-end
5. No reCAPTCHA blocks during normal operation

## Risk Assessment
- **reCAPTCHA enforcement**: Currently optional, may become required. Mitigation: keep UI click as fallback submit path.
- **Token expiry mid-batch**: Bearer token may expire. Mitigation: refresh token before each task via `__NEXT_DATA__` re-read.
- **Google UI changes**: Selectors may break. Mitigation: use semantic selectors (role, data-attributes) not CSS classes.
