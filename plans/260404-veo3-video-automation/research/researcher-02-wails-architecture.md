# Wails v2 Architecture Research: Video Queue Manager

**Date**: 2026-04-04 | **Focus**: Desktop app patterns for video generation queue with real-time progress

---

## 1. Project Structure & Go-Frontend Binding

**Structure**: Wails wraps Go backend + web frontend (React/Vue/Svelte with Vite) into single Windows binary using WebView2.

- Backend: `main.go` initializes app, exposes struct methods via `Bind: []interface{}{&App{}}`
- Frontend: Auto-generated `wailsjs/go/` bindings with TypeScript declarations
- Assets: Bundled into binary; frontend hotreloads in dev, immutable in production

**Key**: Public methods on bound structs auto-expose to frontend. Private methods stay server-side.

---

## 2. Frontend-Backend Communication Patterns

### A. RPC-Style Bindings (Sync/Async Calls)
```
Frontend → Go: window.go.Package.MethodName(arg1, arg2)
Returns: Promise<T> always (even for sync Go methods)
Auto-typed: Full TypeScript declarations generated
```

### B. Events (Async Push from Go to UI)
```
Go → Frontend: runtime.EventsEmit(ctx, "eventName", data)
Frontend listen: window.wails.EventsOn("eventName", (data) => {...})
Use case: Progress updates, long-running task notifications
```

**For video queue**: Use Bind for pause/resume/add-task commands. Use EventsEmit for progress (0-100%), status text, frame preview URLs.

---

## 3. Exposing Go Functions (wails.Bind Pattern)

Create an App struct with public methods:
```go
type App struct {
  ctx context.Context
  queue *VideoQueue
}

func (a *App) EnqueueTask(req VideoRequest) string { ... }
func (a *App) PauseQueue() error { ... }
func (a *App) GetStatus() QueueStatus { ... }
```

In `main.go`:
```go
app := &App{queue: NewVideoQueue()}
wails.Run(&options.App{
  Bind: []interface{}{app},
  ...
})
```

**Struct field rules**: Use `json` tags for serialization. Return types must be serializable (int, string, struct with json tags).

---

## 4. Serving Local Files (MP4 Videos)

**Method**: Implement custom `http.Handler` in `AssetsHandler` option.

Issue: Bundled assets work; dynamic files (videos) need custom handler.

```go
type AssetsHandler struct {
  videosDir string
}

func (h *AssetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  if strings.HasPrefix(r.URL.Path, "/videos/") {
    filePath := filepath.Join(h.videosDir, strings.TrimPrefix(r.URL.Path, "/videos/"))
    http.ServeFile(w, r, filePath)
    return
  }
  http.NotFound(w, r)
}

// In main.go:
wails.Run(&options.App{
  AssetsHandler: &AssetsHandler{videosDir: "./videos"},
  ...
})
```

Frontend: `<video src="/videos/myfile.mp4" />`

**Security**: Validate & sanitize file paths. Don't expose `..` traversal. Consider allowlisting directories.

---

## 5. Background Goroutines & Progress Events

**Pattern**: Spawn worker goroutine, use channels for pause/resume signals, emit events for UI updates.

```go
type VideoQueue struct {
  tasks chan Task
  pause chan bool
  ctx   context.Context
}

func (q *VideoQueue) Run(ctx context.Context) {
  for {
    select {
    case task := <-q.tasks:
      q.processTask(task, ctx)
    case paused := <-q.pause:
      if paused { <-q.resume } // block until resumed
    case <-ctx.Done():
      return
    }
  }
}

func (q *VideoQueue) processTask(task Task, ctx context.Context) {
  for progress := 0; progress <= 100; progress += 10 {
    runtime.EventsEmit(ctx, "progress", map[string]interface{}{
      "taskId": task.ID,
      "progress": progress,
    })
    time.Sleep(time.Second) // simulate work
  }
}

func (a *App) PauseQueue() error {
  a.queue.pause <- true
  return nil
}
```

Frontend listens:
```ts
window.wails.EventsOn('progress', (data) => {
  updateProgressBar(data.progress);
});
```

**Key**: Use context for cancellation. Channels for pause/resume. EventsEmit for status.

---

## 6. SQLite Database (modernc.org/sqlite)

**Why pure Go**: No CGo compilation. Cross-platform. Simple deployment.

```go
import "modernc.org/sqlite"
import "database/sql"

db, _ := sql.Open("sqlite", "file:queue.db")
// Standard SQL operations work normally
```

**Schema for video queue**:
```sql
CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  status TEXT,
  progress INT,
  created_at TIMESTAMP,
  completed_at TIMESTAMP
);
```

**Usage in App struct**:
```go
type App struct {
  db *sql.DB
  ctx context.Context
}

func (a *App) GetTaskHistory() []Task {
  rows, _ := a.db.Query("SELECT id, status, progress FROM tasks ORDER BY created_at DESC")
  // scan & return
}
```

**Persistence**: Tasks survive app restart. Store before/after queue runs.

---

## 7. Frameless Window with Custom Title Bar

**Enable frameless**:
```go
wails.Run(&options.App{
  Windows: []*windows.Options{{
    Frameless: true,
    Resizable: true,
  }},
})
```

**CSS drag region** (React example):
```tsx
<header style={{'--wails-draggable': 'drag'} as any}>
  <span>VEO3 Queue Manager</span>
  <div className="window-controls">
    <button onClick={() => window.wails.Runtime.Window.Minimise()}>_</button>
    <button onClick={() => window.wails.Runtime.Window.Maximise()}>[ ]</button>
    <button onClick={() => window.wails.Runtime.Window.Close()}>×</button>
  </div>
</header>
```

**Limitation**: Windows WebView2 doesn't support native window control overlays. Custom HTML buttons are the workaround.

---

## 8. Queue Management: Pause/Resume/Stop

**State machine** (Go backend):
```go
type QueueState string
const (
  Running QueueState = "running"
  Paused  QueueState = "paused"
  Stopped QueueState = "stopped"
)

type VideoQueue struct {
  state    QueueState
  mu       sync.RWMutex
  tasks    []Task
  current  *Task
}

func (q *VideoQueue) Pause() {
  q.mu.Lock()
  q.state = Paused
  q.mu.Unlock()
}

func (q *VideoQueue) Resume() {
  q.mu.Lock()
  q.state = Running
  q.mu.Unlock()
}
```

Frontend commands:
```
App.PauseQueue() → Backend sets state=Paused, worker goroutine checks before processing next task
App.ResumeQueue() → Sets state=Running, worker continues
App.StopQueue() → Sets state=Stopped, clears pending tasks, emits "queueStopped" event
```

---

## 9. Building & Distributing for Windows

**Development**: 
```bash
wails dev  # Hot-reload with live frontend changes
```

**Production build**:
```bash
wails build -platform windows/amd64  # Creates .exe in build/bin/
```

**Output**: Single .exe file. WebView2 runtime bundled. No external dependencies.

**Distribution**: 
- Direct .exe delivery
- MSIX package (Windows Store)
- Auto-updater via go-update (3rd party)

---

## 10. Architecture Summary for Your App

```
Frontend (React/TS)
├── Queue UI (tasks, progress bars, controls)
├── Video preview (HTML5 <video> tag)
└── Calls App.PauseQueue(), App.AddTask(), listens to "progress" events

Wails Runtime
├── EventsEmit/EventsOn bridge
└── HTTP AssetsHandler (serves /videos/ from disk)

Backend (Go)
├── App struct (bound methods: EnqueueTask, PauseQueue, etc.)
├── VideoQueue worker (goroutine, channels, progress events)
├── SQLite DB (task history, status persistence)
└── File handler (streams MP4 to frontend)
```

---

## Unresolved Questions

1. **Frame preview serving**: Best way to serve individual frame PNGs from video during processing for live preview?
2. **Large file handling**: How to handle 100MB+ videos efficiently? Stream or memory-map?
3. **Auto-update strategy**: Which library/approach for auto-updates in production?
4. **Error recovery**: Retry logic for failed tasks—persistence pattern?

---

**Sources**:
- [The Wails Project](https://wails.io/)
- [Wails Events Reference](https://wails.io/docs/reference/runtime/events/)
- [How Wails Works](https://wails.io/docs/howdoesitwork/)
- [Dynamic Assets Guide](https://wails.io/docs/guides/dynamic-assets/)
- [Frameless Applications](https://wails.io/docs/guides/frameless/)
- [Method Bindings (v3)](https://v3alpha.wails.io/features/bindings/methods/)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)
- [Go-WebView2](https://github.com/wailsapp/go-webview2)
- [DEV Community: Password Manager with Wails](https://dev.to/emarifer/a-minimalist-password-manager-desktop-app-a-foray-into-golangs-wails-framework-part-1-kao)
- [Medium: Building a Desktop App in Go using Wails](https://medium.com/@pliutau/building-a-desktop-app-in-go-using-wails-756c1f31f75)
