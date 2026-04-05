package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"veo3-manager/internal/chrome"
	"veo3-manager/internal/database"
)

type Pipeline struct {
	browserMgr  *chrome.BrowserManager
	db          *database.DB
	ctx         context.Context
	downloadDir string
	activePage  *rod.Page
}

func New(browserMgr *chrome.BrowserManager, db *database.DB, downloadDir string) *Pipeline {
	return &Pipeline{
		browserMgr:  browserMgr,
		db:          db,
		downloadDir: downloadDir,
	}
}

func (p *Pipeline) SetDownloadDir(dir string) {
	p.downloadDir = dir
}

func (p *Pipeline) SetContext(ctx context.Context) {
	p.ctx = ctx
}

// ExecuteTask runs the full pipeline for a single task
func (p *Pipeline) ExecuteTask(task *database.Task) error {
	defer func() { p.activePage = nil }()

	if p.browserMgr.Browser() == nil {
		return fmt.Errorf("browser not connected")
	}

	p.db.UpdateTaskStatus(task.ID, "processing")
	p.emitProgress(task.ID, "processing", "Preparing...")

	// Step 1: Get or create stealth page
	page, err := p.ensurePage()
	if err != nil {
		return fmt.Errorf("failed to get page: %w", err)
	}

	// Step 2: Extract auth
	p.emitProgress(task.ID, "processing", "Extracting auth token...")
	token, projectId, err := p.resolveAuth(page)
	if err != nil {
		return err
	}

	// Step 3: Configure settings
	p.emitProgress(task.ID, "processing", "Configuring settings...")
	if err := ConfigureSettings(page, task.AspectRatio, task.Model, task.OutputCount); err != nil {
		fmt.Println("Warning: settings configuration:", err)
	}

	// Step 4: Enter prompt
	p.emitProgress(task.ID, "processing", "Entering prompt...")
	if err := ClearEditor(page); err != nil {
		return fmt.Errorf("failed to clear editor: %w", err)
	}
	time.Sleep(300 * time.Millisecond)
	if err := InsertPrompt(page, task.Prompt); err != nil {
		return fmt.Errorf("failed to insert prompt: %w", err)
	}
	time.Sleep(500 * time.Millisecond)

	// Step 5: Click Create and capture media IDs
	p.emitProgress(task.ID, "processing", "Submitting...")
	mediaIDs, err := p.submitWithRetry(page, task)
	if err != nil {
		return err
	}

	p.db.UpdateTaskMediaIDs(task.ID, mediaIDs)
	p.emitProgress(task.ID, "polling", fmt.Sprintf("Waiting for %d video(s)...", len(mediaIDs)))

	// Step 6: Poll for completion
	result, err := p.pollWithTokenRefresh(page, token, projectId, task.ID, mediaIDs)
	if err != nil {
		return err
	}
	if !result.AllDone {
		return fmt.Errorf("polling ended but not all videos completed")
	}

	// Step 7: Download videos
	videoPaths := p.downloadVideos(page, task, mediaIDs)
	if len(videoPaths) == 0 {
		return fmt.Errorf("no videos downloaded successfully")
	}

	p.db.UpdateTaskVideoPaths(task.ID, videoPaths)
	p.emitProgress(task.ID, "completed", fmt.Sprintf("Done! %d video(s) downloaded", len(videoPaths)))
	return nil
}

// resolveAuth gets token and projectId, refreshing if needed
func (p *Pipeline) resolveAuth(page *rod.Page) (string, string, error) {
	token := p.browserMgr.GetToken()
	projectId := p.browserMgr.GetProjectId()

	if token == "" || !p.browserMgr.IsTokenValid() || projectId == "" {
		auth, err := p.browserMgr.ExtractAuth(page)
		if err != nil {
			return "", "", fmt.Errorf("failed to extract auth: %w", err)
		}
		token = auth.Token
		projectId = auth.ProjectId
	}

	if projectId == "" {
		if info, _ := page.Info(); info != nil {
			projectId = extractProjectIdFromURL(info.URL)
		}
	}

	return token, projectId, nil
}

// submitWithRetry attempts to click Create and capture media IDs with retry
func (p *Pipeline) submitWithRetry(page *rod.Page, task *database.Task) ([]string, error) {
	for attempt := 1; attempt <= 2; attempt++ {
		fmt.Printf("[Pipeline] Submit attempt %d...\n", attempt)
		ids, err := p.clickCreateAndCapture(page, task.OutputCount)
		if err == nil && len(ids) > 0 {
			return ids, nil
		}
		fmt.Printf("[Pipeline] Submit attempt %d failed: %v\n", attempt, err)
		if attempt < 2 {
			p.emitProgress(task.ID, "processing", "Retrying submit...")
			time.Sleep(2 * time.Second)
		}
	}
	return nil, fmt.Errorf("failed to submit via UI after retries")
}

// pollWithTokenRefresh polls status, retrying once with refreshed token on 401
func (p *Pipeline) pollWithTokenRefresh(page *rod.Page, token, projectId, taskID string, mediaIDs []string) (*PollResult, error) {
	onProgress := func(status string) {
		p.emitProgress(taskID, "polling", status)
	}

	result, err := PollStatus(p.ctx, token, projectId, mediaIDs, onProgress)
	if err == nil {
		return result, nil
	}
	if !strings.Contains(err.Error(), "401") {
		return nil, err
	}

	newToken, refreshErr := p.browserMgr.RefreshToken(page)
	if refreshErr != nil {
		return nil, fmt.Errorf("token refresh failed: %w (original: %v)", refreshErr, err)
	}
	return PollStatus(p.ctx, newToken, projectId, mediaIDs, onProgress)
}

// downloadVideos downloads all videos and returns paths
func (p *Pipeline) downloadVideos(page *rod.Page, task *database.Task, mediaIDs []string) []string {
	p.emitProgress(task.ID, "downloading", "Downloading videos...")
	var videoPaths []string

	for i, mediaID := range mediaIDs {
		destPath := filepath.Join(p.downloadDir, fmt.Sprintf("%s_%d.mp4", task.ID, i))
		p.emitProgress(task.ID, "downloading", fmt.Sprintf("Downloading video %d/%d...", i+1, len(mediaIDs)))

		if err := DownloadVideo(page, mediaID, destPath); err != nil {
			fmt.Printf("Warning: failed to download video %d (%s): %v\n", i, mediaID[:8], err)
			continue
		}
		videoPaths = append(videoPaths, destPath)
	}

	return videoPaths
}

// clickCreateAndCapture clicks Create button and captures mediaIDs from intercepted API response
func (p *Pipeline) clickCreateAndCapture(page *rod.Page, expectedCount int) ([]string, error) {
	var mediaIDs []string
	mediaCh := make(chan string, 10)
	done := make(chan struct{})

	router := page.HijackRequests()
	router.MustAdd("*aisandbox-pa.googleapis.com*batchAsyncGenerateVideoText*", func(ctx *rod.Hijack) {
		ctx.MustLoadResponse()
		if ctx.Request.Method() != "POST" {
			return
		}
		var resp SubmitResponse
		if err := json.Unmarshal([]byte(ctx.Response.Body()), &resp); err != nil {
			return
		}
		for _, op := range resp.Operations {
			if op.Operation.Name != "" {
				select {
				case mediaCh <- op.Operation.Name:
				default:
				}
			}
		}
	})
	go router.Run()

	go func() {
		defer close(done)
		timeout := time.After(30 * time.Second)
		for {
			select {
			case id := <-mediaCh:
				mediaIDs = append(mediaIDs, id)
				if len(mediaIDs) >= expectedCount {
					return
				}
			case <-timeout:
				return
			}
		}
	}()

	if err := ClickCreate(page); err != nil {
		router.Stop()
		return nil, err
	}

	<-done
	router.Stop()

	if len(mediaIDs) == 0 {
		return nil, fmt.Errorf("no media IDs captured from API response")
	}
	return mediaIDs, nil
}

func (p *Pipeline) emitProgress(taskID, status, message string) {
	if p.ctx == nil {
		return
	}
	runtime.EventsEmit(p.ctx, "task:progress", map[string]string{
		"taskId":  taskID,
		"status":  status,
		"message": message,
	})
}
