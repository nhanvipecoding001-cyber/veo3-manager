# Phase 8: Frontend — Settings & Polish

## Context

- **Parent plan:** [plan.md](./plan.md)
- **Dependencies:** All previous phases
- **Research:** [researcher-01-report.md](./research/researcher-01-report.md)

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-04-03 |
| Description | Settings page (Chrome config, download folder, debug info), local file server verification, final UI polish, error states, edge cases |
| Priority | P2 — Medium |
| Implementation Status | Not Started |
| Review Status | Pending |

## Key Insights

- Settings page is primary configuration point — Chrome path, user-data-dir, download folder
- Debug info section helps users troubleshoot: browser status, token validity, last error
- File server for video playback must be verified end-to-end (Fact #10)
- Polish phase covers loading states, empty states, error boundaries, animations

## Requirements

1. Settings page with Chrome configuration fields
2. Download folder selector (native folder picker)
3. Debug information display (browser status, token info, app version)
4. Local file server verification and testing
5. Error boundaries and graceful error states across all pages
6. Loading skeletons and transitions
7. Keyboard shortcuts for common actions

## Architecture

```
frontend/src/
├── pages/
│   └── Settings.tsx
├── components/
│   ├── settings/
│   │   ├── ChromeSettings.tsx    # Chrome path, user-data-dir, debug port
│   │   ├── StorageSettings.tsx   # Download folder picker
│   │   └── DebugInfo.tsx         # Runtime debug information
│   └── ui/
│       ├── ErrorBoundary.tsx     # React error boundary
│       ├── Skeleton.tsx          # Loading skeleton
│       ├── EmptyState.tsx        # Empty state placeholder
│       └── ConfirmDialog.tsx     # Confirmation modal
```

## Related Code Files

| File | Purpose |
|------|---------|
| `frontend/src/pages/Settings.tsx` | Settings page composition |
| `frontend/src/components/settings/ChromeSettings.tsx` | Chrome path, user-data-dir, port inputs |
| `frontend/src/components/settings/StorageSettings.tsx` | Download folder with native picker |
| `frontend/src/components/settings/DebugInfo.tsx` | Browser status, token, version, logs |
| `frontend/src/components/ui/ErrorBoundary.tsx` | Catch React render errors |
| `frontend/src/components/ui/Skeleton.tsx` | Animated loading placeholder |
| `frontend/src/components/ui/EmptyState.tsx` | Empty state with icon + message |
| `app.go` | Bindings: `SelectFolder()`, `GetDebugInfo()`, `OpenFolder()` |
| `internal/fileserver/handler.go` | Verified /localfile/ handler |

## Implementation Steps

### 1. Backend Bindings

```go
// Folder picker using Wails runtime dialog
func (a *App) SelectFolder() (string, error) {
    return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
        Title: "Select Download Folder",
    })
}

// Open folder in Windows Explorer
func (a *App) OpenFolder(path string) error {
    return exec.Command("explorer", path).Start()
}

// Debug info
type DebugInfo struct {
    AppVersion    string `json:"appVersion"`
    BrowserStatus string `json:"browserStatus"`
    ChromePath    string `json:"chromePath"`
    UserDataDir   string `json:"userDataDir"`
    DebugPort     int    `json:"debugPort"`
    TokenValid    bool   `json:"tokenValid"`
    DbPath        string `json:"dbPath"`
    DbSize        string `json:"dbSize"`
    TotalTasks    int    `json:"totalTasks"`
}

func (a *App) GetDebugInfo() (*DebugInfo, error)
```

### 2. Create `ChromeSettings.tsx`
- **Chrome Path:** Text input + Browse button (file picker for chrome.exe)
  - Auto-detect button: calls `DetectChromePath()` binding
  - Shows current detected path
- **User Data Dir:** Text input + Browse button (folder picker)
  - Default suggestion: `%LOCALAPPDATA%/veo3-batch-generator/chrome-profile`
  - Warning text: "This folder stores your Google login session"
- **Debug Port:** Number input, default 9222
  - Validation: 1024-65535 range
- **Save button** per section or auto-save on blur
- **Test Connection button:** Calls `LaunchBrowser()`, shows result

### 3. Create `StorageSettings.tsx`
- **Download Folder:** Text input + Browse button (native folder picker via `SelectFolder()`)
  - Default: `%USERPROFILE%/Videos/Veo3`
  - "Open Folder" button to open in Explorer
  - Shows disk space available (optional)
- **Clear Cache button:** Purge completed task video files (with confirmation dialog)

### 4. Create `DebugInfo.tsx`
- Read-only information panel
- Fields displayed in `<dl>` (definition list) format:
  - App Version
  - Browser Status (with colored indicator)
  - Token Valid (green check / red X)
  - Database Path + Size
  - Total Tasks in DB
  - Chrome PID (if connected)
- "Copy Debug Info" button: copies all info as text to clipboard
- "Refresh" button to re-fetch

### 5. Create `Settings.tsx` Page
```tsx
<div className="space-y-6">
  <h1>Settings</h1>
  <ChromeSettings />
  <StorageSettings />
  <DebugInfo />
</div>
```
- Each section in a card with header

### 6. Create `ErrorBoundary.tsx`
```tsx
class ErrorBoundary extends React.Component<Props, State> {
  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }
  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center h-full">
          <AlertTriangle className="text-red-500" size={48} />
          <h2>Something went wrong</h2>
          <p>{this.state.error.message}</p>
          <Button onClick={() => this.setState({ hasError: false })}>
            Try Again
          </Button>
        </div>
      );
    }
    return this.props.children;
  }
}
```
- Wrap each page route with ErrorBoundary

### 7. Create `Skeleton.tsx` and `EmptyState.tsx`
- Skeleton: animated pulse placeholder matching card/table row shapes
- EmptyState: centered icon + title + description + optional action button
  - Dashboard empty: "No tasks yet. Go to Queue to get started."
  - History empty: "No completed tasks. Start generating videos!"
  - Queue empty: "Add prompts to begin."

### 8. Create `ConfirmDialog.tsx`
- Modal with title, message, Cancel/Confirm buttons
- Used for: clear cache, stop queue (if tasks pending), delete task
- Accessible: focus trap, Escape to close

### 9. UI Polish Pass

**Loading States:**
- Dashboard stats: skeleton cards during fetch
- History table: skeleton rows during fetch
- Queue task list: spinner on initial load

**Transitions:**
- Page transitions: subtle fade (CSS `transition-opacity`)
- Toast enter/exit: slide from right
- Modal: fade + scale up

**Error States:**
- API call failures: inline error message with retry button
- Browser disconnected: banner at top of Queue page "Browser not connected. Click to connect."
- Video file missing: placeholder in carousel "Video file not found"

**Keyboard Shortcuts:**
- `Ctrl+N`: Focus prompt input (Queue page)
- `Ctrl+Enter`: Submit prompt
- `Escape`: Close modal/preview
- `Left/Right`: Navigate carousel

### 10. Verify File Server End-to-End
- Confirm `AssetsHandler` in `main.go` routes `/localfile/*` to `fileserver.Handler`
- Test: generate a video, verify playback in History carousel
- Test: path with spaces and special characters
- Test: file not found returns 404 gracefully

### 11. Final Integration Testing
- Full workflow: Settings → Queue → add prompts → start → Dashboard shows stats → History shows completed → video plays
- Edge cases: empty DB, no Chrome installed, invalid download folder, network error mid-generation

## Todo

- [ ] Create SelectFolder and OpenFolder backend bindings
- [ ] Create GetDebugInfo backend binding
- [ ] Implement ChromeSettings with path/dir/port inputs
- [ ] Implement StorageSettings with folder picker
- [ ] Implement DebugInfo with copy-to-clipboard
- [ ] Create Settings page composing all sections
- [ ] Create ErrorBoundary component
- [ ] Create Skeleton and EmptyState components
- [ ] Create ConfirmDialog component
- [ ] Add loading states to Dashboard and History
- [ ] Add empty states to all pages
- [ ] Add CSS transitions for pages, toasts, modals
- [ ] Implement keyboard shortcuts
- [ ] Verify /localfile/ handler with video playback
- [ ] Full integration test: end-to-end workflow
- [ ] Polish: consistent spacing, alignment, hover states

## Success Criteria

1. Settings page saves Chrome config and download folder correctly
2. Native folder picker opens and returns selected path
3. Debug info displays accurate runtime information
4. Error boundaries catch and display errors gracefully
5. Loading skeletons appear during data fetches
6. Empty states display when no data present
7. Video playback works via /localfile/ handler
8. Full end-to-end workflow completes without errors

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Native dialog blocks UI thread | Low | Wails dialogs are async by default |
| Settings changes break running queue | Medium | Validate inputs before save, warn if queue running |
| Chrome not found on target machine | Medium | Clear error message with download link |

## Security Considerations

- Folder picker restricted to user-accessible directories by OS
- Debug info shows paths — not sensitive, but don't log auth tokens
- File server path validation prevents directory traversal (implemented in Phase 1)
- No auto-update mechanism — users download new versions manually

## Next Steps

After Phase 8 completion, the application is feature-complete. Consider:
- User acceptance testing
- Creating installer (NSIS or WiX via Wails build)
- Writing user documentation
- Performance profiling for long queue runs
