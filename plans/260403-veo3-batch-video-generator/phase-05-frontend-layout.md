# Phase 5: Frontend — Layout & Navigation

## Context

- **Parent plan:** [plan.md](./plan.md)
- **Dependencies:** [Phase 1](./phase-01-scaffolding.md) (frontend scaffolding, stores)
- **Research:** [researcher-01-report.md](./research/researcher-01-report.md) — Wails frameless, Tailwind dark mode

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-04-03 |
| Description | Frameless window with custom titlebar, sidebar navigation with browser status indicator, routing between 4 pages, toast notification system |
| Priority | P1 — High |
| Implementation Status | Not Started |
| Review Status | Pending |

## Key Insights

- Wails frameless window: `Frameless: true` + CSS `.wails-draggable` for drag region
- Window controls (minimize/maximize/close) via `runtime.WindowMinimise()`, etc.
- Browser status from Wails events (`browser:status`) drives sidebar indicator
- Dark theme only (no toggle needed) — Tailwind `dark` class on `<html>`
- Toast system for success/error notifications from queue events

## Requirements

1. Custom titlebar with drag region and window controls
2. Sidebar with navigation links, active state, browser status indicator
3. React Router with 4 routes: Dashboard, Queue, History, Settings
4. Toast notification system (success, error, info)
5. Responsive layout within fixed window dimensions

## Architecture

```
frontend/src/
├── components/
│   ├── layout/
│   │   ├── AppLayout.tsx      # Main layout wrapper (titlebar + sidebar + content)
│   │   ├── TitleBar.tsx       # Custom frameless titlebar
│   │   └── Sidebar.tsx        # Navigation + browser status
│   └── ui/
│       ├── Toast.tsx          # Toast notification component
│       ├── ToastContainer.tsx # Positioned toast stack
│       └── Button.tsx         # Shared button component
├── stores/
│   ├── appStore.ts            # Browser status, toasts
│   └── ...
├── hooks/
│   └── useWailsEvent.ts       # Hook for Wails event subscription
├── App.tsx                    # Router + AppLayout
└── main.tsx                   # Entry point
```

### Color Palette (Dark Theme)
```
Background:     bg-gray-950    (#0a0a0f)
Surface:        bg-gray-900    (#111118)
Card:           bg-gray-800    (#1f1f2e)
Border:         border-gray-700 (#374151)
Text Primary:   text-gray-100  (#f3f4f6)
Text Secondary: text-gray-400  (#9ca3af)
Accent:         text-blue-500  (#3b82f6)
Success:        text-green-500 (#22c55e)
Error:          text-red-500   (#ef4444)
Warning:        text-yellow-500(#eab308)
```

## Related Code Files

| File | Purpose |
|------|---------|
| `frontend/src/App.tsx` | BrowserRouter, route definitions, AppLayout wrapper |
| `frontend/src/components/layout/AppLayout.tsx` | Grid layout: titlebar + sidebar + main content |
| `frontend/src/components/layout/TitleBar.tsx` | Drag region, app title, window controls |
| `frontend/src/components/layout/Sidebar.tsx` | Nav links with icons, browser status dot |
| `frontend/src/components/ui/Toast.tsx` | Individual toast component |
| `frontend/src/components/ui/ToastContainer.tsx` | Toast stack, auto-dismiss |
| `frontend/src/stores/appStore.ts` | `browserStatus`, `toasts[]`, `addToast()` |
| `frontend/src/hooks/useWailsEvent.ts` | `useWailsEvent(eventName, callback)` |

## Implementation Steps

### 1. Create `useWailsEvent.ts` Hook
```typescript
import { useEffect } from 'react';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';

export function useWailsEvent(event: string, callback: (...args: any[]) => void) {
  useEffect(() => {
    EventsOn(event, callback);
    return () => { EventsOff(event); };
  }, [event, callback]);
}
```

### 2. Create `appStore.ts`
```typescript
interface Toast {
  id: string;
  type: 'success' | 'error' | 'info';
  message: string;
}

interface AppState {
  browserStatus: 'disconnected' | 'connecting' | 'connected' | 'error';
  setBrowserStatus: (status: string) => void;
  toasts: Toast[];
  addToast: (type: Toast['type'], message: string) => void;
  removeToast: (id: string) => void;
}
```

### 3. Create `TitleBar.tsx`
- Full-width bar at top, height ~36px
- Left: app icon + "Veo3 Batch Generator" text
- Center: draggable region with `--wails-draggable: drag` CSS property
- Right: minimize, maximize, close buttons using `WindowMinimise()`, `WindowToggleMaximise()`, `Quit()` from Wails runtime
- Icons: `Minus`, `Square`, `X` from Lucide React

### 4. Create `Sidebar.tsx`
- Fixed width: 220px
- Nav items with Lucide icons:
  - `LayoutDashboard` → Dashboard (`/`)
  - `ListVideo` → Queue (`/queue`)
  - `History` → History (`/history`)
  - `Settings` → Settings (`/settings`)
- Active state: `bg-gray-800` background, `text-blue-500` icon/text
- Bottom section: Browser status indicator
  - Green dot + "Connected" / Red dot + "Disconnected" / Yellow dot + "Connecting"
  - Click triggers `LaunchBrowser()` binding if disconnected

### 5. Create `AppLayout.tsx`
```tsx
<div className="h-screen flex flex-col bg-gray-950 text-gray-100">
  <TitleBar />
  <div className="flex flex-1 overflow-hidden">
    <Sidebar />
    <main className="flex-1 overflow-auto p-6">
      <Outlet />
    </main>
  </div>
  <ToastContainer />
</div>
```

### 6. Create Toast System
- `Toast.tsx`: renders single toast with icon, message, close button
- `ToastContainer.tsx`: fixed position bottom-right, renders toast stack
- Auto-dismiss after 5 seconds
- Animate in/out with CSS transitions

### 7. Set Up Router in `App.tsx`
```tsx
<BrowserRouter>
  <Routes>
    <Route element={<AppLayout />}>
      <Route path="/" element={<Dashboard />} />
      <Route path="/queue" element={<Queue />} />
      <Route path="/history" element={<History />} />
      <Route path="/settings" element={<Settings />} />
    </Route>
  </Routes>
</BrowserRouter>
```

### 8. Subscribe to Wails Events
- In `AppLayout`, use `useWailsEvent('browser:status', ...)` to update appStore
- Subscribe to `task:completed` and `task:failed` to show toasts

### 9. Tailwind Configuration
```js
// tailwind.config.js
module.exports = {
  darkMode: 'class',
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // Custom overrides if needed
      }
    }
  }
}
```
- Add `class="dark"` to `<html>` element in `index.html`

## Todo

- [ ] Create useWailsEvent hook
- [ ] Create appStore with browser status and toast state
- [ ] Implement TitleBar with drag region and window controls
- [ ] Implement Sidebar with navigation and browser status indicator
- [ ] Create AppLayout composing TitleBar + Sidebar + content area
- [ ] Set up React Router with 4 routes
- [ ] Implement toast notification system
- [ ] Subscribe to browser:status Wails events
- [ ] Configure Tailwind dark theme
- [ ] Style all components with dark color palette
- [ ] Test: navigation between pages, window controls, toast display

## Success Criteria

1. Frameless window with working drag, minimize, maximize, close
2. Sidebar navigation highlights active page
3. Browser status indicator updates in real-time from Wails events
4. All 4 pages render correctly via router
5. Toast notifications appear and auto-dismiss
6. Consistent dark theme across all components

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| `--wails-draggable` CSS property not working | Medium | Fallback to `wails-draggable` data attribute |
| Route state lost on window refresh in dev | Low | Expected behavior; not an issue in production |
| Toast stack overflow on rapid events | Low | Limit max visible toasts to 5, FIFO |

## Security Considerations

- Window controls call Wails runtime only — no direct OS calls from frontend
- No sensitive data displayed in titlebar or sidebar

## Next Steps

After Phase 5 completion, proceed to [Phase 6: Frontend — Queue Page](./phase-06-frontend-queue.md).
