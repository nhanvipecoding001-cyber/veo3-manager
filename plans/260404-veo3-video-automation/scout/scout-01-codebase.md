# Veo3 Manager - Codebase Scout Report

## Project Structure
```
veo3-manager/
├── main.go                     — Entry point, config init, Wails app launch
├── app.go                      — Main App struct, all Wails bindings (task/queue/browser/settings)
├── wails.json                  — Wails v2 config
├── go.mod                      — Dependencies: rod, stealth, sqlite, wails/v2, uuid
├── internal/
│   ├── chrome/
│   │   ├── browser.go          — BrowserManager: launch/connect Chrome, health check, token mgmt
│   │   ├── session.go          — CDP session, token extraction from __NEXT_DATA__
│   │   ├── config.go           — ChromeConfig struct (path, user-data-dir, debug port)
│   │   └── stealth.go          — go-rod/stealth anti-detection
│   ├── config/
│   │   └── config.go           — AppConfig struct (DB, download dir, Chrome path)
│   ├── database/
│   │   ├── db.go               — SQLite init, migrations, CRUD
│   │   ├── models.go           — Task, TaskFilter, TaskStats, StringSlice
│   │   ├── tasks.go            — Task operations (Create, Get, List, Update, Delete)
│   │   └── settings.go         — Key-value settings persistence
│   ├── fileserver/
│   │   └── handler.go          — Serve local .mp4 files via HTTP for WebView
│   ├── pipeline/
│   │   ├── pipeline.go         — Orchestrator: navigate, configure, submit, poll, download
│   │   ├── api.go              — API client: PollStatus, checkStatus (old structure)
│   │   ├── download.go         — CaptureAndDownload via Chrome tab redirect + HTTP
│   │   ├── prompt.go           — Slate.js editor: ClearEditor, InsertPrompt, ClickCreate
│   │   └── settings.go         — UI automation: openDropdown, selectAspectRatio/Model/Count
│   └── queue/
│       └── queue.go            — Queue Manager: Start/Pause/Resume/Stop, worker loop
└── frontend/src/
    ├── App.tsx                  — Root component with routes
    ├── main.tsx                 — React entry
    ├── style.css                — Global styles
    ├── pages/
    │   ├── Dashboard.tsx        — Stats overview
    │   ├── Queue.tsx            — Prompt input + task list + controls
    │   ├── History.tsx          — Completed tasks table
    │   └── Settings.tsx         — Chrome/download config
    ├── components/layout/
    │   ├── AppLayout.tsx        — Sidebar + content
    │   ├── Sidebar.tsx          — Navigation + browser status
    │   └── TitleBar.tsx         — Custom frameless title bar
    ├── stores/
    │   ├── appStore.ts          — Global state (Zustand)
    │   ├── queueStore.ts        — Queue state
    │   └── settingsStore.ts     — Settings state
    ├── hooks/useWailsEvent.ts   — Wails event listener hook
    └── types/index.ts           — TS interfaces
```

## Critical Issues Found (vs captured API)

1. **api.go**: Uses `GET /v1/operations/{id}` — WRONG. Real API uses `POST /v1/video:batchCheckAsyncVideoGenerationStatus` with body `{"media":[{"name":"<id>","projectId":"<pid>"}]}`
2. **api.go**: Response structure doesn't match real API. Missing `media[]` array with nested `mediaMetadata.mediaStatus`
3. **pipeline.go**: Uses `interceptAndCreate` with `HijackRequests` to parse response — response structure wrong
4. **download.go**: Uses Chrome tab + NetworkResponseReceived to capture GCS URL. Works but simpler: browser `fetch()` to getMediaUrlRedirect → response.url gives signed GCS URL directly
5. **settings.go**: Searches for `"Create"` text in buttons — real button text is `"Tạo"` or `"arrow_forwardTạo"` (Vietnamese locale)
6. **pipeline.go**: Navigates to `video-fx` not `flow` — should be `labs.google/fx/vi/tools/flow`
7. **settings.go**: Missing VIDEO/IMAGE tab switch — needs click VIDEO tab before selecting aspect ratio

## Key Data from API Capture

- Submit: `POST /v1/video:batchAsyncGenerateVideoText`
- Poll: `POST /v1/video:batchCheckAsyncVideoGenerationStatus` 
- Download: `GET labs.google/fx/api/trpc/media.getMediaUrlRedirect?name={mediaId}` → GCS signed URL
- Auth: Bearer token from `__NEXT_DATA__.props.pageProps.session.access_token`
- Model: `veo_3_1_t2v_fast` (only working model)
- Status: `MEDIA_GENERATION_STATUS_SUCCESSFUL` / `_PENDING` / `_FAILED`
- Each video costs 20 credits
