# Windows Desktop App Research Report
**Date:** 2026-04-03 | **Focus:** Wails v2 + Go + React + Rod + SQLite Stack

---

## 1. Wails v2 Architecture

**Current Status:** v2.8.1+ stable (early 2025), native Webview2 on Windows (no DLL bundling req'd)

**Project Structure:**
```
project/
  ├─ frontend/          (React/TypeScript bundle)
  ├─ build/             (Compiled binaries)
  ├─ main.go            (Wails entry, Go runtime)
  ├─ wails.json         (Config: output name, build flags, asset paths)
  └─ go.mod/go.sum
```

**Go-JS Bindings:**
- Wails auto-generates TypeScript types from exported Go struct methods
- Go methods become async JS functions; params/returns must be JSON-serializable
- Bindings reflect at startup; no runtime registration needed
- Type safety via generated `.ts` interfaces matching Go struct tags

**Frameless Windows:**
- Set `Frameless: true` in runtime options to remove native chrome
- Custom drag region handled via CSS (`.wails-draggable` class)
- Resize/minimize/close via `runtime.Window` methods from Go

**Serving Local Files:**
- `AssetsHandler` option (http.Handler) intercepts non-GET or missing asset requests
- Implement custom handler to serve dynamically-generated files (video outputs, metadata)
- Requests → bundled FS first → fallback to handler → 404
- Alternative: Custom protocol schemes (e.g., `app://` prefix) for explicit routing

**Windows Gotchas:**
- Webview2 requires stable Windows 10+ (1809+)
- No CGo requirement; native Go webview2loader.dll (~130KB smaller)
- Resource embedding: use `//go:embed` directive for frontend assets

---

## 2. go-rod/rod + Stealth

**Library:** Rod v0.114+ (2024-2025); pure CDP driver for Chrome/Chromium

**Chrome Launch & Debug:**
```go
// Custom user-data-dir + stealth
u := launcher.New().
  UserDataDir("/tmp/chrome-profile").
  Headless(false).
  XVFB().  // Linux only
  Launch()

browser := rod.New().ControlURL(u).Connect()
```

**Existing Chrome Connection (debug port):**
```go
u := launcher.NewUserMode().
  RemoteDebuggingPort(9222).
  Launch()
// Then: rod.New().ControlURL("ws://localhost:9222").Connect()
```

**Stealth Mode:**
- No built-in stealth; use `go-rod/stealth` package (community-maintained)
- Inject scripts to mask automation signatures (navigator.webdriver, chrome obj refs)
- Apply per-page: `page.MustSetExtra("stealth", true)` + script injection
- Note: Bot detection evolves; stealth is arms race, not guarantee

**CDP Commands (Low-level):**
```go
// Insert text via CDP Input domain
page.MustEval(`() => { /* JS code */ }`)

// Intercept navigation/redirects
page.MustOnDialog(func(d *rod.Dialog) { d.Accept() })

// Network events for URL capture
go page.OnRequest(func(req *rod.Request) {
  println(req.URL())
})
```

**URL Capture Pattern:**
- Use `page.OnRequest` listener to log all requests (catches redirects)
- Or: `page.MustNavigateToURL()` + `page.MustInfo().URL` for final URL
- Redirects trapped via Network.requestWillBeSent events

---

## 3. SQLite in Pure Go (modernc.org/sqlite)

**Driver:** `modernc.org/sqlite` v1.31.0+ (2024)

**Key Advantage:** No CGo; pure Go transliteration of SQLite C code via ccgo. Cross-compile anywhere.

**Setup:**
```go
import "modernc.org/sqlite"
db, _ := sql.Open("sqlite", "file:app.db")
defer db.Close()
```

**Migrations Pattern:**
- No built-in migration runner; use `golang-migrate/migrate` or `pressly/goose`
- Or: Manual schema version table + `IF NOT EXISTS` checks
- Load SQL files from `//go:embed`; execute on first run

**JSON Columns:**
- SQLite JSON1 extension enabled in modernc.org/sqlite
- Store complex data: `json_extract(json_col, '$.field')` in queries
- Go: Scan into `json.RawMessage` or custom types with Marshal/Unmarshal

**Performance Notes:**
- CGo-free eliminates context switch overhead (vs sqlite3)
- ~10-15% slower than C SQLite for heavy ops (acceptable for UI app)
- WAL mode recommended for concurrent read/write scenarios

---

## 4. React + Vite + Tailwind + Zustand + Lucide

**Frontend Stack (Wails v2 compatible):**

**Vite Config:**
- `vite.config.ts`: React plugin, TS support, minify
- Auto-reload on hot changes during dev

**Tailwind Dark Mode:**
- Config: `darkMode: 'class'` in `tailwind.config.js`
- Toggle: Add/remove `dark` class on `<html>` element
- Utilities: `dark:bg-gray-900 dark:text-white` syntax

**State Management (Zustand):**
```typescript
// Store structure for batch queue
const useQueueStore = create<QueueState>((set) => ({
  queue: [],
  addJob: (job) => set((s) => ({ queue: [...s.queue, job] })),
  removeJob: (id) => set((s) => ({ queue: s.queue.filter(j => j.id !== id) })),
  setTheme: (theme) => set({ theme }),
}));
```
- No boilerplate; simple object return from create()
- Async support via middleware
- Persist to localStorage: `persist` middleware

**Lucide React:**
- Import icons: `import { Sun, Moon, Play } from 'lucide-react'`
- Size/color control: `<Sun size={20} className="..." />`
- Tree-shaking: Only used icons bundled

**Best Practices:**
- Keep stores focused (one per domain: queue, settings, etc.)
- Use TypeScript interfaces for type safety
- Separate business logic (Zustand) from UI (React components)
- Context for theme provider wrapper
- Custom hooks for computed state (derived from store)

**Bundle Size Consideration:**
- Vite tree-shaking removes unused code
- Dynamic imports for code-split routes if multi-tab UI
- Lucide over SVG files (smaller, consistent)

---

## 5. Integration Patterns for Veo3 Batch App

**Data Flow:**
1. Frontend (React): User submits batch jobs → Zustand queue store
2. Zustand async middleware: Call Wails binding to Go
3. Go backend: Dispatch to Rod for browser interaction + SQLite logging
4. Rod: Navigate, intercept URLs, store results in DB
5. Frontend: Poll status via Wails binding; update UI from Zustand

**Custom HTTP Handler Use Case:**
- Serve generated video metadata/thumbnails via `AssetsHandler`
- Path: `/videos/{id}/thumbnail.jpg` → fetch from temp folder

**Architecture Recommendations:**
- Keep Rod instances in goroutine pool (reuse browsers)
- Batch job tracking: ID → status (queued, running, done, failed) in SQLite
- WebSocket alternative: Wails runtime events (lower overhead than polling)

---

## Versions Summary

| Component | Version | Notes |
|-----------|---------|-------|
| Wails | v2.8.1+ | Latest v2 stable; v3 alpha exists |
| Go | 1.21+ | Standard; CGo not required |
| Rod | v0.114+ | Latest CDP stable |
| modernc.org/sqlite | v1.31.0+ | SQLite 3.51.2, windows/386 support |
| React | 18.x | Via Vite template |
| Vite | 5.x | Default for Wails v2 React-TS |
| Tailwind | v4.x | Latest (v3 also works) |
| Zustand | 4.x | Minimal overhead |

---

## Open Questions / Gotchas

- **Rod stealth reliability:** Detection evasion requires periodic updates; no permanent solution
- **SQLite concurrent writes:** WAL mode helps but not guaranteed safe for heavy multi-threaded writes
- **Wails asset caching:** Frontend caching strategy for video outputs (cache busting needed)
- **Rod memory leaks:** Long-running browser instances need periodic restart cycles
- **Windows Webview2 updates:** Runtime auto-updates; edge cases on locked machines

---

## References

- [Wails Documentation](https://wails.io/)
- [go-rod/rod GitHub](https://github.com/go-rod/rod)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)
- [Tailwind CSS Dark Mode](https://tailwindcss.com/docs/dark-mode)
- [Zustand GitHub](https://github.com/pmndrs/zustand)
