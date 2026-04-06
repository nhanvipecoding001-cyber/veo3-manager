# Phase 4: Queue Management System

## Context

- **Parent plan:** [plan.md](./plan.md)
- **Dependencies:** [Phase 3](./phase-03-video-pipeline.md) (Pipeline.ExecuteTask)
- **Research:** [researcher-02-report.md](./research/researcher-02-report.md) — Go queue patterns

## Overview

| Field | Value |
|-------|-------|
| Date | 2026-04-03 |
| Description | Job queue with single goroutine worker, pause/resume/stop, config hot-reload, error handling/retry, progress via Wails events |
| Priority | P0 — Critical |
| Implementation Status | Not Started |
| Review Status | Pending |

## Key Insights

- Single worker goroutine — sequential processing (one prompt at a time per Google Labs constraints)
- Config changes take effect on next task without restarting queue (hot-reload via channel message)
- Use `context.WithCancel` for graceful stop
- Pause/resume via dedicated channels — worker blocks on pause signal
- Wails `runtime.EventsEmit` for real-time progress updates to frontend

## Requirements

1. Single-worker job queue consuming tasks from DB (pending status)
2. Pause/resume/stop controls
3. Config hot-reload — next task picks up latest settings
4. Error handling with optional retry (configurable max retries)
5. Progress reporting via Wails events (task status changes, polling progress)
6. Queue state persistence — resume after app restart

## Architecture

```
internal/queue/
├── worker.go    # QueueWorker — main processing loop
└── queue.go     # QueueManager — public API, state management
```

### State Machine
```
Idle → Running → Paused → Running → Idle
  ↓       ↓         ↓        ↓
  └───────┴─────────┴────────┴──→ Stopped
```

### Channel Architecture
```go
type QueueManager struct {
    pipeline   *pipeline.Pipeline
    db         *database.DB
    ctx        context.Context
    cancelFunc context.CancelFunc

    state      QueueState    // idle | running | paused | stopped
    mu         sync.Mutex

    pauseCh    chan struct{} // signal pause
    resumeCh   chan struct{} // signal resume
    configCh   chan QueueConfig // hot-reload config
    
    wailsCtx   context.Context // for EventsEmit
}
```

### Event Types Emitted
```
queue:state    → { state: "running" | "paused" | "idle" | "stopped" }
task:started   → { taskId, prompt }
task:progress  → { taskId, phase: "submitting"|"polling"|"downloading", detail }
task:completed → { taskId, videoPaths }
task:failed    → { taskId, error }
queue:stats    → { pending, processing, completed, failed }
```

## Related Code Files

| File | Purpose |
|------|---------|
| `internal/queue/queue.go` | `QueueManager` — Start, Pause, Resume, Stop, AddTask |
| `internal/queue/worker.go` | Worker loop, task processing, event emission |
| `app.go` | Bindings: `StartQueue()`, `PauseQueue()`, `ResumeQueue()`, `StopQueue()`, `AddToQueue()` |
| `internal/database/tasks.go` | `GetNextPendingTask()`, `UpdateTaskStatus()` |

## Implementation Steps

### 1. Create `internal/queue/queue.go`

```go
type QueueState string
const (
    StateIdle    QueueState = "idle"
    StateRunning QueueState = "running"
    StatePaused  QueueState = "paused"
    StateStopped QueueState = "stopped"
)

type QueueConfig struct {
    AspectRatio string
    Model       string
    OutputCount int
    DelayBetweenTasks time.Duration
}

type QueueManager struct {
    pipeline   *pipeline.Pipeline
    db         *database.DB
    state      QueueState
    mu         sync.RWMutex
    ctx        context.Context
    cancel     context.CancelFunc
    pauseCh    chan struct{}
    resumeCh   chan struct{}
    wailsCtx   context.Context
}

func NewQueueManager(p *pipeline.Pipeline, db *database.DB, wailsCtx context.Context) *QueueManager

func (qm *QueueManager) Start() error    // Start processing loop
func (qm *QueueManager) Pause() error    // Pause after current task completes
func (qm *QueueManager) Resume() error   // Resume paused queue
func (qm *QueueManager) Stop() error     // Stop after current task, cancel context
func (qm *QueueManager) GetState() QueueState
func (qm *QueueManager) AddTask(prompt string) (*database.Task, error)
func (qm *QueueManager) AddTasks(prompts []string) ([]*database.Task, error) // Bulk add
func (qm *QueueManager) RemoveTask(taskID string) error // Only if pending
func (qm *QueueManager) RequeueTask(taskID string) error // Reset failed → pending
```

### 2. Create `internal/queue/worker.go`

```go
func (qm *QueueManager) runWorker() {
    defer func() {
        qm.setState(StateIdle)
        qm.emitState()
    }()

    for {
        // Check for pause
        select {
        case <-qm.pauseCh:
            qm.setState(StatePaused)
            qm.emitState()
            // Block until resume or stop
            select {
            case <-qm.resumeCh:
                qm.setState(StateRunning)
                qm.emitState()
            case <-qm.ctx.Done():
                return
            }
        case <-qm.ctx.Done():
            return
        default:
        }

        // Get next pending task from DB
        // Config read fresh each iteration (hot-reload: Fact about config changes)
        task, err := qm.db.GetNextPendingTask()
        if err != nil {
            qm.emitError(err)
            continue
        }
        if task == nil {
            // No more tasks — queue is empty
            qm.setState(StateIdle)
            qm.emitState()
            return
        }

        // Process task
        qm.emitTaskStarted(task)
        qm.db.UpdateTaskStatus(task.ID, "processing")

        err = qm.pipeline.ExecuteTask(task)
        if err != nil {
            qm.db.UpdateTask(task.ID, map[string]interface{}{
                "status": "failed", "error_message": err.Error(),
            })
            qm.emitTaskFailed(task, err)
        } else {
            qm.emitTaskCompleted(task)
        }

        // Optional delay between tasks
        select {
        case <-time.After(2 * time.Second):
        case <-qm.ctx.Done():
            return
        }
    }
}
```

### 3. Config Hot-Reload
- Settings stored in DB (settings table)
- `Pipeline.ExecuteTask` reads current settings from DB at start of each task
- No channel needed — just read DB each iteration
- UI changes write to DB immediately → next task picks up new values

### 4. Wails Bindings in `app.go`

```go
func (a *App) StartQueue() error {
    return a.queueManager.Start()
}

func (a *App) PauseQueue() error {
    return a.queueManager.Pause()
}

func (a *App) ResumeQueue() error {
    return a.queueManager.Resume()
}

func (a *App) StopQueue() error {
    return a.queueManager.Stop()
}

func (a *App) GetQueueState() string {
    return string(a.queueManager.GetState())
}

func (a *App) AddToQueue(prompt string) (*database.Task, error) {
    return a.queueManager.AddTask(prompt)
}

func (a *App) AddBulkToQueue(prompts []string) ([]*database.Task, error) {
    return a.queueManager.AddTasks(prompts)
}

func (a *App) RemoveFromQueue(taskID string) error {
    return a.queueManager.RemoveTask(taskID)
}

func (a *App) RequeueTask(taskID string) error {
    return a.queueManager.RequeueTask(taskID)
}
```

### 5. DB Methods for Queue

```go
// GetNextPendingTask returns oldest pending task (FIFO)
func (db *DB) GetNextPendingTask() (*Task, error) {
    // SELECT * FROM tasks WHERE status = 'pending' ORDER BY created_at ASC LIMIT 1
}

// CancelPendingTasks marks all pending tasks as cancelled
func (db *DB) CancelPendingTasks() error {
    // UPDATE tasks SET status = 'cancelled' WHERE status = 'pending'
}
```

### 6. App Restart Recovery
- On startup, check for tasks with status `processing` or `polling` — set to `failed` (interrupted)
- Queue doesn't auto-start — user must click Start
- Pending tasks preserved across restarts

## Todo

- [ ] Create QueueManager with Start/Pause/Resume/Stop
- [ ] Implement single-worker goroutine loop
- [ ] Implement pause/resume via channels
- [ ] Implement graceful stop via context cancellation
- [ ] Read config from DB each iteration (hot-reload)
- [ ] Emit Wails events for state changes and task progress
- [ ] Create Wails bindings for all queue operations
- [ ] Add bulk task creation (multiple prompts)
- [ ] Add requeue functionality for failed tasks
- [ ] Handle app restart recovery (mark interrupted tasks as failed)
- [ ] Add configurable delay between tasks
- [ ] Test: add 3 tasks, start queue, pause mid-processing, resume, complete

## Success Criteria

1. Queue processes tasks sequentially (FIFO order)
2. Pause stops after current task completes, resume continues
3. Stop cancels processing and returns to idle
4. Config changes in UI reflected in next task without restart
5. Frontend receives real-time events for all state/progress changes
6. App restart preserves pending tasks, marks interrupted as failed

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Goroutine leak on unclean shutdown | Medium | Context cancellation + WaitGroup |
| Race condition on state transitions | Medium | Mutex protection on state field |
| Task stuck in "processing" forever | High | Timeout per task (10 min max), mark as failed |
| Panic in worker crashes app | High | Recover in worker goroutine, log error, continue |

## Security Considerations

- Queue operations only accessible via Wails bindings (local app, no network)
- No user data leaves the machine except via Chrome to Google Labs

## Next Steps

After Phase 4 completion, proceed to [Phase 5: Frontend — Layout & Navigation](./phase-05-frontend-layout.md).
