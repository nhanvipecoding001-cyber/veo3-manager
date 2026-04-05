package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"veo3-manager/internal/database"
	"veo3-manager/internal/pipeline"
)

type State string

const (
	StateIdle    State = "idle"
	StateRunning State = "running"
	StatePaused  State = "paused"
	StateStopping State = "stopping"
)

type Config struct {
	DelayBetweenTasks time.Duration
}

type Manager struct {
	pipeline *pipeline.Pipeline
	db       *database.DB
	wailsCtx context.Context

	state      State
	mu         sync.Mutex
	cancelFunc context.CancelFunc

	pauseCh  chan struct{}
	resumeCh chan struct{}
	doneCh   chan struct{}
}

func NewManager(p *pipeline.Pipeline, db *database.DB) *Manager {
	return &Manager{
		pipeline: p,
		db:       db,
		state:    StateIdle,
		pauseCh:  make(chan struct{}, 1),
		resumeCh: make(chan struct{}, 1),
	}
}

func (m *Manager) SetContext(ctx context.Context) {
	m.wailsCtx = ctx
	m.pipeline.SetContext(ctx)
}

func (m *Manager) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *Manager) setState(s State) {
	m.state = s
	if m.wailsCtx != nil {
		runtime.EventsEmit(m.wailsCtx, "queue:state", string(s))
	}
}

// Start begins processing the queue
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateRunning {
		return fmt.Errorf("queue already running")
	}

	// Reset any stuck "processing" tasks from previous crashed runs
	m.db.ResetStuckTasks()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	m.doneCh = make(chan struct{})
	m.setState(StateRunning)

	go m.worker(ctx)
	return nil
}

// Pause pauses the queue after current task completes
func (m *Manager) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateRunning {
		return
	}
	m.setState(StatePaused)
	select {
	case m.pauseCh <- struct{}{}:
	default:
	}
}

// Resume resumes a paused queue
func (m *Manager) Resume() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StatePaused {
		return
	}
	m.setState(StateRunning)
	select {
	case m.resumeCh <- struct{}{}:
	default:
	}
}

// Stop stops the queue gracefully
func (m *Manager) Stop() {
	m.mu.Lock()
	if m.state == StateIdle {
		m.mu.Unlock()
		return
	}
	m.setState(StateStopping)
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	// If paused, send resume to unblock worker
	select {
	case m.resumeCh <- struct{}{}:
	default:
	}
	m.mu.Unlock()

	// Wait for worker to finish
	if m.doneCh != nil {
		<-m.doneCh
	}

	m.mu.Lock()
	m.setState(StateIdle)
	m.mu.Unlock()
}

func (m *Manager) worker(ctx context.Context) {
	defer close(m.doneCh)

	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check for pause
		m.mu.Lock()
		if m.state == StatePaused {
			m.mu.Unlock()
			// Wait for resume or stop
			select {
			case <-m.resumeCh:
				continue
			case <-ctx.Done():
				return
			}
		}
		m.mu.Unlock()

		// Get next pending task from DB (hot-reload: always reads latest)
		tasks, err := m.db.GetPendingTasks()
		if err != nil {
			m.emitError("", fmt.Sprintf("Failed to get pending tasks: %v", err))
			time.Sleep(5 * time.Second)
			continue
		}

		if len(tasks) == 0 {
			// No more tasks — go idle
			m.mu.Lock()
			m.setState(StateIdle)
			m.mu.Unlock()
			return
		}

		task := tasks[0]

		// Emit task started
		if m.wailsCtx != nil {
			runtime.EventsEmit(m.wailsCtx, "task:started", map[string]string{
				"taskId": task.ID,
				"prompt": task.Prompt,
			})
		}

		// Execute task
		if err := m.pipeline.ExecuteTask(&task); err != nil {
			m.db.UpdateTaskError(task.ID, err.Error())
			if m.wailsCtx != nil {
				runtime.EventsEmit(m.wailsCtx, "task:failed", map[string]string{
					"taskId": task.ID,
					"error":  err.Error(),
				})
			}
		} else {
			if m.wailsCtx != nil {
				runtime.EventsEmit(m.wailsCtx, "task:completed", map[string]interface{}{
					"taskId": task.ID,
				})
			}
		}

		// Emit updated stats
		m.emitStats()

		// Delay between tasks — read fresh config from DB each time (hot-reload)
		delay := m.getDelay()
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) Pipeline() *pipeline.Pipeline {
	return m.pipeline
}

func (m *Manager) getDelay() time.Duration {
	val, err := m.db.GetSetting("delay_between_tasks")
	if err != nil {
		return 5 * time.Second
	}
	var seconds int
	fmt.Sscanf(val, "%d", &seconds)
	if seconds <= 0 {
		seconds = 5
	}
	return time.Duration(seconds) * time.Second
}

func (m *Manager) emitStats() {
	if m.wailsCtx == nil {
		return
	}
	stats, err := m.db.GetTaskStats()
	if err == nil {
		runtime.EventsEmit(m.wailsCtx, "queue:stats", stats)
	}
}

func (m *Manager) emitError(taskID, message string) {
	if m.wailsCtx != nil {
		runtime.EventsEmit(m.wailsCtx, "queue:error", map[string]string{
			"taskId":  taskID,
			"message": message,
		})
	}
}
