# Phase 1: Project Scaffolding & Core Infrastructure

## Context

- **Parent plan:** [plan.md](./plan.md)
- **Dependencies:** None (first phase)
- **Research:** [researcher-01-report.md](./research/researcher-01-report.md)

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-04-03 |
| Description | Initialize Wails v2 project, establish Go backend structure, SQLite database with schema, React/Tailwind/Zustand frontend scaffolding |
| Priority | P0 — Critical |
| Implementation Status | Not Started |
| Review Status | Pending |

## Key Insights

- Wails v2 auto-generates TypeScript bindings from exported Go struct methods
- `modernc.org/sqlite` is pure Go — no CGo needed, simplifies cross-compilation
- Frameless window configured via `Frameless: true` in Wails options
- `AssetsHandler` provides custom HTTP handler for serving local files (Fact #10)
- WAL mode recommended for SQLite concurrent read/write

## Requirements

1. Wails v2 project with React-TS template
2. Go package structure following standard layout
3. SQLite database with migration system
4. Frontend scaffolding with dark theme, routing, state management
5. Wails bindings for basic CRUD operations
6. Local file server handler for video playback (Fact #10)

## Architecture

```
veo3-batch-generator/
├── main.go                    # Wails entry point
├── wails.json                 # Wails config
├── go.mod / go.sum
├── app.go                     # App struct with Wails lifecycle methods
├── internal/
│   ├── chrome/                # Phase 2: Chrome automation
│   │   ├── browser.go
│   │   ├── stealth.go
│   │   └── session.go
│   ├── pipeline/              # Phase 3: Video generation
│   │   ├── api.go
│   │   ├── prompt.go
│   │   ├── settings.go
│   │   └── download.go
│   ├── queue/                 # Phase 4: Queue management
│   │   ├── worker.go
│   │   └── queue.go
│   ├── database/
│   │   ├── db.go              # DB init, migrations
│   │   ├── models.go          # Go structs
│   │   ├── tasks.go           # Task CRUD
│   │   └── settings.go        # Settings CRUD
│   ├── fileserver/
│   │   └── handler.go         # /localfile/{path} handler
│   └── config/
│       └── config.go          # App config struct
├── frontend/
│   ├── index.html
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   ├── tsconfig.json
│   ├── package.json
│   ├── src/
│   │   ├── main.tsx           # React entry
│   │   ├── App.tsx            # Router + layout
│   │   ├── components/
│   │   │   ├── layout/
│   │   │   │   ├── TitleBar.tsx
│   │   │   │   ├── Sidebar.tsx
│   │   │   │   └── AppLayout.tsx
│   │   │   ├── ui/            # Shared UI primitives
│   │   │   │   ├── Button.tsx
│   │   │   │   ├── Toast.tsx
│   │   │   │   └── ...
│   │   │   └── ...
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Queue.tsx
│   │   │   ├── History.tsx
│   │   │   └── Settings.tsx
│   │   ├── stores/
│   │   │   ├── queueStore.ts
│   │   │   ├── settingsStore.ts
│   │   │   └── appStore.ts
│   │   ├── hooks/
│   │   │   └── useWailsEvent.ts
│   │   ├── lib/
│   │   │   └── wails.ts       # Wails runtime helpers
│   │   └── types/
│   │       └── index.ts
│   └── wailsjs/               # Auto-generated bindings
└── build/
    └── windows/
        ├── icon.ico
        └── wails.exe.manifest
```

### Database Schema

```sql
-- tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id            TEXT PRIMARY KEY,        -- UUID
    prompt        TEXT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    -- status: pending | processing | polling | downloading | completed | failed | cancelled
    aspect_ratio  TEXT NOT NULL DEFAULT '16:9',
    model         TEXT NOT NULL DEFAULT 'veo_3_1_t2v_fast_ultra',
    output_count  INTEGER NOT NULL DEFAULT 4,
    media_ids     TEXT,                    -- JSON array of media operation IDs
    video_paths   TEXT,                    -- JSON array of local file paths
    error_message TEXT,
    seed          TEXT,                    -- JSON array of seeds if returned
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at  DATETIME
);

-- settings table (key-value)
CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Default settings
INSERT OR IGNORE INTO settings (key, value) VALUES
    ('chrome_path', ''),
    ('user_data_dir', ''),
    ('download_folder', ''),
    ('debug_port', '9222'),
    ('aspect_ratio', '16:9'),
    ('model', 'veo_3_1_t2v_fast_ultra'),
    ('output_count', '4');
```

## Related Code Files

| File | Purpose |
|------|---------|
| `main.go` | Wails app entry, window config, bindings registration |
| `app.go` | App struct with `startup()`, `shutdown()`, `domReady()` |
| `internal/database/db.go` | SQLite init, WAL mode, migration runner |
| `internal/database/models.go` | Task, Settings Go structs |
| `internal/database/tasks.go` | CreateTask, GetTask, ListTasks, UpdateTask |
| `internal/fileserver/handler.go` | HTTP handler serving local .mp4 files |
| `internal/config/config.go` | AppConfig struct, defaults |
| `frontend/src/main.tsx` | React entry point |
| `frontend/src/App.tsx` | Router, layout composition |
| `frontend/src/stores/appStore.ts` | App-level state (browser status, theme) |

## Implementation Steps

### 1. Initialize Wails Project
```bash
wails init -n veo3-batch-generator -t react-ts
cd veo3-batch-generator
```

### 2. Configure Go Dependencies
```bash
go get modernc.org/sqlite
go get github.com/google/uuid
```

### 3. Create `main.go`
- Set `Frameless: true` for custom title bar
- Set `Width: 1280, Height: 800, MinWidth: 960, MinHeight: 600`
- Register `App` struct with `Bind` option
- Set `AssetsHandler` to `fileserver.NewHandler(downloadFolder)` for `/localfile/` routes
- Set `OnStartup`, `OnShutdown`, `OnDomReady` lifecycle hooks
- Set `BackgroundColour` to dark theme default (#1a1a2e or similar)

### 4. Create `app.go`
- Define `App` struct holding `*database.DB`, `context.Context`
- `startup(ctx)`: init DB, run migrations, load config
- `shutdown(ctx)`: close DB, cleanup Chrome if running
- Export methods for frontend bindings: `GetTasks()`, `CreateTask()`, `GetSettings()`, `UpdateSetting()`

### 5. Create `internal/database/db.go`
- `New(dbPath string) (*DB, error)` — opens SQLite, enables WAL, runs migrations
- Embed SQL schema via `//go:embed schema.sql`
- Migration: check `PRAGMA user_version`, execute schema if version == 0, bump version

### 6. Create `internal/database/models.go`
- `Task` struct with JSON tags matching DB columns
- `VideoPaths` field: custom type wrapping `[]string` with `json.Marshal/Unmarshal` for scan/value
- `Settings` as `map[string]string`

### 7. Create `internal/database/tasks.go`
- `CreateTask(prompt, aspectRatio, model string, outputCount int) (*Task, error)`
- `GetTask(id string) (*Task, error)`
- `ListTasks(filter TaskFilter) ([]Task, error)` — support status/search filters
- `UpdateTask(id string, updates map[string]interface{}) error`
- `GetTaskStats() (*Stats, error)` — counts by status for dashboard

### 8. Create `internal/fileserver/handler.go`
- Implement `http.Handler` that serves files from download folder
- Route: requests to `/localfile/{encoded-path}` serve the file with proper MIME type
- Security: validate path is within allowed download directory (prevent directory traversal)

### 9. Frontend Scaffolding
- Install: `npm install zustand lucide-react react-router-dom`
- Configure Tailwind with `darkMode: 'class'`, set `dark` class on `<html>` by default
- Create base color palette: gray-900 bg, gray-800 cards, blue-500 accent
- Set up react-router with 4 routes: `/`, `/queue`, `/history`, `/settings`
- Create minimal page components (placeholder content)
- Create Zustand stores with initial state shapes

### 10. Verify Build
```bash
wails dev    # Development mode
wails build  # Production build
```

## Todo

- [ ] Initialize Wails v2 project with react-ts template
- [ ] Add Go dependencies (sqlite, uuid)
- [ ] Create main.go with frameless window config
- [ ] Create app.go with lifecycle hooks
- [ ] Implement database package (init, migrations, CRUD)
- [ ] Create SQLite schema (tasks + settings tables)
- [ ] Implement file server handler for /localfile/ routes
- [ ] Set up frontend with Tailwind dark theme
- [ ] Configure react-router with 4 pages
- [ ] Create Zustand stores (app, queue, settings)
- [ ] Create placeholder page components
- [ ] Verify `wails dev` runs successfully

## Success Criteria

1. `wails dev` starts without errors, shows frameless window with dark theme
2. SQLite DB created on first run with correct schema
3. CRUD operations work via Wails bindings (test via browser console)
4. React router navigates between 4 pages
5. `/localfile/` handler serves test file correctly

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Wails v2 React-TS template outdated | Low | Pin versions, update vite/react manually |
| SQLite WAL mode file locking on Windows | Medium | Ensure single writer pattern, use mutex |
| WebView2 not installed on target machine | Low | Wails bootstrapper auto-installs WebView2 |

## Security Considerations

- File server handler MUST validate paths to prevent directory traversal attacks
- Database file stored in app data directory, not user-accessible location
- No credentials stored in database; auth tokens live only in Chrome session

## Next Steps

After Phase 1 completion, proceed to [Phase 2: Chrome Automation Engine](./phase-02-chrome-automation.md).
