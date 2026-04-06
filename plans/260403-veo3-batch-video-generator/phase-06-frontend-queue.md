# Phase 6: Frontend — Queue Page

## Context

- **Parent plan:** [plan.md](./plan.md)
- **Dependencies:** [Phase 4](./phase-04-queue-management.md) (queue bindings), [Phase 5](./phase-05-frontend-layout.md) (layout, stores, events)
- **Research:** [researcher-01-report.md](./research/researcher-01-report.md) — Zustand patterns

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-04-03 |
| Description | Queue page with settings panel, prompt input (single + bulk), task list with real-time status, queue controls, live progress |
| Priority | P1 — High |
| Implementation Status | Not Started |
| Review Status | Pending |

## Key Insights

- Settings panel changes write to DB immediately, take effect on next task (hot-reload)
- Prompt input supports single textarea and bulk import (one prompt per line)
- Task list updates via Wails events (`task:started`, `task:progress`, `task:completed`, `task:failed`)
- Queue controls: Start, Pause, Resume, Stop — visibility depends on queue state
- Each task can produce 1-4 videos (Fact #9)

## Requirements

1. Settings panel: aspect ratio (16:9 / 9:16), model selector, output count (1-4)
2. Prompt input area with Add button + bulk import
3. Task list showing all tasks with status badges
4. Queue control buttons with state-dependent visibility
5. Live progress indicators during task processing

## Architecture

```
frontend/src/
├── pages/
│   └── Queue.tsx              # Queue page composition
├── components/
│   └── queue/
│       ├── SettingsPanel.tsx   # Aspect ratio, model, output count
│       ├── PromptInput.tsx     # Textarea + Add + Bulk import
│       ├── TaskList.tsx        # Scrollable task list
│       ├── TaskItem.tsx        # Single task row with status
│       ├── QueueControls.tsx   # Start/Pause/Resume/Stop buttons
│       └── ProgressBar.tsx     # Animated progress indicator
├── stores/
│   ├── queueStore.ts          # Tasks, queue state, CRUD actions
│   └── settingsStore.ts       # Settings state, persist to backend
```

### Queue Page Layout
```
┌─────────────────────────────────────────────┐
│  Settings Panel (collapsible)               │
│  ┌──────────┐ ┌──────────┐ ┌─────────────┐ │
│  │Aspect    │ │  Model   │ │Output Count │ │
│  │16:9|9:16 │ │ Dropdown │ │  1|2|3|4    │ │
│  └──────────┘ └──────────┘ └─────────────┘ │
├─────────────────────────────────────────────┤
│  Prompt Input                               │
│  ┌─────────────────────────────┐ ┌───────┐ │
│  │  Enter prompt...            │ │  Add  │ │
│  └─────────────────────────────┘ └───────┘ │
│  [Bulk Import]                              │
├─────────────────────────────────────────────┤
│  Queue Controls                             │
│  [▶ Start] [⏸ Pause] [⏹ Stop]  Pending: 5  │
├─────────────────────────────────────────────┤
│  Task List                                  │
│  ┌─────────────────────────────────────────┐│
│  │ #1 "A cat walking..."   ● Completed  ✓ ││
│  │ #2 "Sunset over..."     ● Processing ▶ ││
│  │ #3 "City skyline..."    ○ Pending    ✕ ││
│  │ ...                                     ││
│  └─────────────────────────────────────────┘│
└─────────────────────────────────────────────┘
```

## Related Code Files

| File | Purpose |
|------|---------|
| `frontend/src/pages/Queue.tsx` | Page layout composing all queue sub-components |
| `frontend/src/components/queue/SettingsPanel.tsx` | Generation settings (aspect, model, count) |
| `frontend/src/components/queue/PromptInput.tsx` | Textarea + Add + Bulk import modal |
| `frontend/src/components/queue/TaskList.tsx` | Scrollable list container |
| `frontend/src/components/queue/TaskItem.tsx` | Task row: prompt preview, status badge, actions |
| `frontend/src/components/queue/QueueControls.tsx` | Start/Pause/Resume/Stop + pending count |
| `frontend/src/components/queue/ProgressBar.tsx` | Animated bar for active task |
| `frontend/src/stores/queueStore.ts` | Tasks state, queue state, actions |
| `frontend/src/stores/settingsStore.ts` | Settings CRUD via Wails bindings |

## Implementation Steps

### 1. Create `settingsStore.ts`
```typescript
interface SettingsState {
  aspectRatio: '16:9' | '9:16';
  model: string;
  outputCount: number;
  chromePath: string;
  userDataDir: string;
  downloadFolder: string;
  debugPort: number;
  loadSettings: () => Promise<void>;
  updateSetting: (key: string, value: string) => Promise<void>;
}
```
- `loadSettings()`: call `GetSettings()` binding, populate store
- `updateSetting()`: call `UpdateSetting(key, value)` binding, update store

### 2. Create `queueStore.ts`
```typescript
interface Task {
  id: string;
  prompt: string;
  status: 'pending' | 'processing' | 'polling' | 'downloading' | 'completed' | 'failed' | 'cancelled';
  aspectRatio: string;
  model: string;
  outputCount: number;
  videoPaths: string[];
  errorMessage?: string;
  createdAt: string;
  completedAt?: string;
}

interface QueueState {
  tasks: Task[];
  queueState: 'idle' | 'running' | 'paused' | 'stopped';
  currentTaskId: string | null;
  progressDetail: string;
  loadTasks: () => Promise<void>;
  addTask: (prompt: string) => Promise<void>;
  addBulkTasks: (prompts: string[]) => Promise<void>;
  removeTask: (id: string) => Promise<void>;
  startQueue: () => Promise<void>;
  pauseQueue: () => Promise<void>;
  resumeQueue: () => Promise<void>;
  stopQueue: () => Promise<void>;
  updateTaskFromEvent: (taskId: string, updates: Partial<Task>) => void;
  setQueueState: (state: string) => void;
}
```

### 3. Create `SettingsPanel.tsx`
- **Aspect ratio:** Two toggle buttons (16:9, 9:16), active state highlighted
- **Model:** Dropdown/select with current model displayed. Only `veo_3_1_t2v_fast_ultra` for now (Fact #9)
- **Output count:** 4 toggle buttons (1, 2, 3, 4)
- Each change calls `settingsStore.updateSetting()` immediately
- Collapsible with chevron toggle

### 4. Create `PromptInput.tsx`
- Textarea: multi-line, placeholder "Enter your video prompt..."
- Add button: calls `queueStore.addTask(prompt)`, clears textarea
- Keyboard: Ctrl+Enter to submit
- Bulk Import button opens modal:
  - Large textarea for pasting multiple prompts (one per line)
  - Import button parses lines, calls `queueStore.addBulkTasks(prompts)`
  - Shows count "X prompts detected"

### 5. Create `TaskItem.tsx`
- Layout: index | truncated prompt (max 60 chars) | status badge | action buttons
- Status badges with colors:
  - Pending: gray dot
  - Processing: blue dot + pulse animation
  - Polling: blue dot + "Waiting..."
  - Downloading: blue dot + "Downloading..."
  - Completed: green dot + checkmark
  - Failed: red dot + error icon (tooltip with error message)
  - Cancelled: gray strikethrough
- Actions:
  - Remove (X button) — only for pending tasks
  - Progress detail text for active task

### 6. Create `TaskList.tsx`
- Scrollable container with max-height
- Maps over `queueStore.tasks` (sorted: processing first, then pending, then completed)
- Empty state: "No tasks in queue. Add prompts above."
- Auto-scroll to active task

### 7. Create `QueueControls.tsx`
- Conditional button rendering based on `queueState`:
  - Idle: [Start] enabled if pending tasks exist
  - Running: [Pause] [Stop]
  - Paused: [Resume] [Stop]
  - Stopped: [Start]
- Stats summary: "Pending: X | Completed: Y | Failed: Z"
- Icons: `Play`, `Pause`, `Square`, `RotateCcw` from Lucide

### 8. Create `ProgressBar.tsx`
- Shown on active task in TaskItem
- For polling phase: indeterminate animation (since we don't know exact progress)
- For downloading: determinate if possible, else indeterminate
- Subtle blue color matching accent

### 9. Wire Up Wails Events in Queue Page
```typescript
// In Queue.tsx
useWailsEvent('queue:state', (state) => queueStore.setQueueState(state));
useWailsEvent('task:started', ({ taskId }) => {
  queueStore.updateTaskFromEvent(taskId, { status: 'processing' });
});
useWailsEvent('task:progress', ({ taskId, phase, detail }) => {
  queueStore.updateTaskFromEvent(taskId, { status: phase });
});
useWailsEvent('task:completed', ({ taskId, videoPaths }) => {
  queueStore.updateTaskFromEvent(taskId, { status: 'completed', videoPaths });
});
useWailsEvent('task:failed', ({ taskId, error }) => {
  queueStore.updateTaskFromEvent(taskId, { status: 'failed', errorMessage: error });
});
```

### 10. Initial Data Load
- On Queue page mount: `settingsStore.loadSettings()` + `queueStore.loadTasks()`
- Fetch current queue state via `GetQueueState()` binding

## Todo

- [ ] Create settingsStore with load/update via Wails bindings
- [ ] Create queueStore with tasks state and queue actions
- [ ] Implement SettingsPanel with aspect ratio, model, output count controls
- [ ] Implement PromptInput with single + bulk add
- [ ] Implement TaskItem with status badges and action buttons
- [ ] Implement TaskList with sorting and empty state
- [ ] Implement QueueControls with state-dependent buttons
- [ ] Implement ProgressBar for active tasks
- [ ] Wire Wails events to store updates
- [ ] Style all components with dark theme
- [ ] Test: add tasks, start queue, verify real-time updates

## Success Criteria

1. Settings changes persist to backend and reflect in UI
2. Single and bulk prompt addition works correctly
3. Task list updates in real-time from Wails events
4. Queue controls show correct buttons per state
5. Active task shows progress indication
6. Pending tasks can be removed, completed tasks remain in list

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Event ordering issues (events arrive out of order) | Medium | Use taskId for matching, ignore stale events |
| Large task list performance | Low | Virtual scrolling if > 100 tasks |
| Bulk import parsing edge cases | Low | Trim whitespace, filter empty lines |

## Security Considerations

- Prompt text sanitized before display (prevent XSS via React default escaping)
- No user input sent to external services from frontend directly

## Next Steps

After Phase 6 completion, proceed to [Phase 7: Frontend — Dashboard & History](./phase-07-frontend-dashboard-history.md).
