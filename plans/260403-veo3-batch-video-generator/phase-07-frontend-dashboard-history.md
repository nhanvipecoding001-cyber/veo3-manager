# Phase 7: Frontend — Dashboard & History

## Context

- **Parent plan:** [plan.md](./plan.md)
- **Dependencies:** [Phase 5](./phase-05-frontend-layout.md) (layout), [Phase 6](./phase-06-frontend-queue.md) (stores, event patterns)
- **Research:** [researcher-01-report.md](./research/researcher-01-report.md) — Zustand, Lucide

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-04-03 |
| Description | Dashboard with stats cards, History page with searchable/filterable table, video preview with multi-video carousel, requeue functionality |
| Priority | P1 — High |
| Implementation Status | Not Started |
| Review Status | Pending |

## Key Insights

- Dashboard stats fetched from DB via `GetTaskStats()` binding
- History page is primary interface for reviewing generated videos
- Multi-video carousel needed since each task produces 1-4 videos (Fact #9)
- Video playback via `/localfile/{path}` HTTP handler (Fact #10)
- Requeue = create new task with same prompt and current settings

## Requirements

1. Dashboard: stat cards (total tasks, videos generated, success rate, pending count)
2. History: searchable/filterable table of completed/failed tasks
3. Video preview modal with multi-video carousel
4. Requeue action on any completed/failed task
5. Stats auto-refresh periodically or on events

## Architecture

```
frontend/src/
├── pages/
│   ├── Dashboard.tsx
│   └── History.tsx
├── components/
│   ├── dashboard/
│   │   ├── StatCard.tsx
│   │   └── StatsGrid.tsx
│   └── history/
│       ├── HistoryTable.tsx
│       ├── HistoryRow.tsx
│       ├── HistoryFilters.tsx
│       ├── VideoPreviewModal.tsx
│       └── VideoCarousel.tsx
```

### Dashboard Layout
```
┌─────────────────────────────────────────┐
│  Dashboard                              │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌────────┐│
│  │Total │ │Videos│ │Success│ │Pending ││
│  │Tasks │ │Gen'd │ │ Rate  │ │  Count ││
│  │  42  │ │ 156  │ │ 94.2% │ │   5    ││
│  └──────┘ └──────┘ └──────┘ └────────┘│
│                                         │
│  Recent Activity (last 10 completed)    │
│  ┌─────────────────────────────────────┐│
│  │ "A cat..." │ Completed │ 2 min ago ││
│  │ "Sunset.." │ Completed │ 5 min ago ││
│  │ "City..."  │ Failed    │ 8 min ago ││
│  └─────────────────────────────────────┘│
└─────────────────────────────────────────┘
```

### History Layout
```
┌─────────────────────────────────────────────┐
│  History                                    │
│  ┌──────────────┐ ┌──────┐ ┌─────────────┐ │
│  │ Search...    │ │Status│ │  Date Range  │ │
│  └──────────────┘ └──────┘ └─────────────┘ │
│  ┌─────────────────────────────────────────┐│
│  │ Prompt   │Status│Videos│Date   │Actions ││
│  │──────────┼──────┼──────┼───────┼────────││
│  │ "A cat.."│ ✓    │  4   │ 4/3   │ ▶ ↻   ││
│  │ "Sun..." │ ✓    │  2   │ 4/3   │ ▶ ↻   ││
│  │ "City.." │ ✕    │  0   │ 4/3   │   ↻   ││
│  └─────────────────────────────────────────┘│
│  Page 1 of 5  [< Prev] [Next >]            │
└─────────────────────────────────────────────┘
```

## Related Code Files

| File | Purpose |
|------|---------|
| `frontend/src/pages/Dashboard.tsx` | Dashboard page with stats grid + recent activity |
| `frontend/src/pages/History.tsx` | History page with filters + table + preview |
| `frontend/src/components/dashboard/StatCard.tsx` | Single stat card (icon, label, value) |
| `frontend/src/components/dashboard/StatsGrid.tsx` | 4-column grid of stat cards |
| `frontend/src/components/history/HistoryTable.tsx` | Table container with pagination |
| `frontend/src/components/history/HistoryRow.tsx` | Single table row with actions |
| `frontend/src/components/history/HistoryFilters.tsx` | Search input + status/date filters |
| `frontend/src/components/history/VideoPreviewModal.tsx` | Modal overlay with carousel |
| `frontend/src/components/history/VideoCarousel.tsx` | Multi-video navigation (prev/next) |
| `app.go` | Bindings: `GetTaskStats()`, `GetTaskHistory()` |

## Implementation Steps

### 1. Backend Bindings in `app.go`

```go
type TaskStats struct {
    TotalTasks     int     `json:"totalTasks"`
    TotalVideos    int     `json:"totalVideos"`
    SuccessRate    float64 `json:"successRate"`
    PendingCount   int     `json:"pendingCount"`
    CompletedCount int     `json:"completedCount"`
    FailedCount    int     `json:"failedCount"`
}

func (a *App) GetTaskStats() (*TaskStats, error)
func (a *App) GetTaskHistory(filter HistoryFilter) ([]Task, int, error)
// filter: { search, status, page, pageSize }
// returns: tasks, totalCount, error

func (a *App) RequeueTask(taskID string) (*Task, error)
```

### 2. DB Query for Stats
```sql
SELECT
    COUNT(*) as total,
    SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
    SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
    SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending
FROM tasks;

-- Total videos: count non-null video_paths entries
SELECT SUM(json_array_length(video_paths)) as total_videos
FROM tasks WHERE video_paths IS NOT NULL;
```

### 3. Create `StatCard.tsx`
```tsx
interface StatCardProps {
  icon: LucideIcon;
  label: string;
  value: string | number;
  color: string; // text-blue-500, text-green-500, etc.
}
```
- Card with `bg-gray-800 rounded-lg p-4`
- Icon top-right, value large centered, label below

### 4. Create `StatsGrid.tsx`
- 4-column grid using `grid grid-cols-4 gap-4`
- Cards: Total Tasks (`ListVideo`), Videos Generated (`Film`), Success Rate (`TrendingUp`), Pending (`Clock`)
- Fetch stats on mount + refresh every 30s or on task events

### 5. Create `Dashboard.tsx`
- StatsGrid at top
- Recent Activity section below: last 10 completed/failed tasks
- Each activity row: truncated prompt, status icon, relative time ("2 min ago")
- Click on activity row navigates to History with that task highlighted

### 6. Create `HistoryFilters.tsx`
- Search input: debounced (300ms), searches prompt text
- Status dropdown: All, Completed, Failed, Cancelled
- Calls parent callback with filter values on change

### 7. Create `HistoryTable.tsx`
- Columns: Prompt (truncated), Status, Videos (#), Date, Actions
- Sortable by date (default: newest first)
- Pagination: 20 items per page, prev/next buttons
- Loading skeleton while fetching

### 8. Create `HistoryRow.tsx`
- Prompt: truncated to ~80 chars, full text on hover tooltip
- Status: colored badge (green Completed, red Failed)
- Videos: count, clickable if > 0 to open preview
- Date: formatted relative or absolute
- Actions:
  - Preview button (eye icon) — opens VideoPreviewModal
  - Requeue button (rotate icon) — calls `RequeueTask(id)`
  - Disabled if no videos for preview

### 9. Create `VideoPreviewModal.tsx`
- Full-screen overlay with backdrop blur
- Close on Escape key or backdrop click
- Contains VideoCarousel
- Shows task prompt text below video

### 10. Create `VideoCarousel.tsx`
- Video player: `<video>` element with controls
- Source: `/localfile/{encodedPath}` (Fact #10)
- Multi-video navigation:
  - Prev/Next arrow buttons (ChevronLeft, ChevronRight from Lucide)
  - Dot indicators below (1/4, 2/4, etc.)
  - Keyboard: left/right arrow keys
- Shows "Video X of Y" counter
```tsx
<video
  src={`/localfile/${encodeURIComponent(videoPaths[currentIndex])}`}
  controls
  autoPlay
  className="w-full max-h-[70vh] rounded-lg"
/>
```

### 11. Requeue Implementation
- Click requeue → call `RequeueTask(taskID)` binding
- Backend creates new task with same prompt + current settings from DB
- Show toast "Task requeued"
- If queue is running, new task gets processed automatically

## Todo

- [ ] Create GetTaskStats and GetTaskHistory backend bindings
- [ ] Implement SQL queries for stats aggregation
- [ ] Create StatCard and StatsGrid components
- [ ] Create Dashboard page with stats + recent activity
- [ ] Create HistoryFilters with search and status filter
- [ ] Create HistoryTable with pagination
- [ ] Create HistoryRow with preview and requeue actions
- [ ] Create VideoPreviewModal with overlay
- [ ] Create VideoCarousel with multi-video navigation
- [ ] Implement requeue functionality
- [ ] Test: video playback via /localfile/ handler
- [ ] Test: search and filter History table

## Success Criteria

1. Dashboard shows accurate stats from database
2. History table loads with pagination, search filters work
3. Video preview modal opens and plays videos from local filesystem
4. Carousel navigates between multiple videos per task
5. Requeue creates new pending task with same prompt
6. Stats refresh on task completion events

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Video file deleted but path in DB | Medium | Show placeholder if file missing, graceful error |
| Large history table slow to render | Low | Server-side pagination (already planned) |
| Video codec not supported by WebView2 | Low | MP4/H.264 universally supported |

## Security Considerations

- Video file paths encoded to prevent path injection in URL
- File server validates paths within download directory (Phase 1 handler)
- Search input sanitized before DB query (parameterized queries)

## Next Steps

After Phase 7 completion, proceed to [Phase 8: Frontend — Settings & Polish](./phase-08-frontend-settings-polish.md).
