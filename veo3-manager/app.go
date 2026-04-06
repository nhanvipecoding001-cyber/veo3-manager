package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"veo3-manager/internal/chrome"
	"veo3-manager/internal/config"
	"veo3-manager/internal/database"
	"veo3-manager/internal/pipeline"
	"veo3-manager/internal/queue"
)

type App struct {
	ctx        context.Context
	db         *database.DB
	cfg        *config.AppConfig
	browserMgr *chrome.BrowserManager
	queueMgr   *queue.Manager
}

func NewApp(cfg *config.AppConfig) *App {
	return &App{cfg: cfg}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	db, err := database.New(a.cfg.DBPath)
	if err != nil {
		fmt.Println("Failed to open database:", err)
		return
	}
	a.db = db

	// Load saved settings into config
	settings, err := a.db.GetSettings()
	if err == nil {
		if v, ok := settings["download_folder"]; ok && v != "" {
			a.cfg.DownloadDir = v
		}
		if v, ok := settings["chrome_path"]; ok && v != "" {
			a.cfg.ChromePath = v
		}
		if v, ok := settings["user_data_dir"]; ok && v != "" {
			a.cfg.UserDataDir = v
		}
		if v, ok := settings["debug_port"]; ok && v != "" {
			a.cfg.DebugPort = v
		}
	}

	// Initialize browser manager
	debugPort := 9222
	if p, err := strconv.Atoi(a.cfg.DebugPort); err == nil {
		debugPort = p
	}
	chromeCfg := &chrome.ChromeConfig{
		ChromePath:  a.cfg.ChromePath,
		UserDataDir: a.cfg.UserDataDir,
		DebugPort:   debugPort,
	}
	a.browserMgr = chrome.NewBrowserManager(chromeCfg)
	a.browserMgr.SetContext(ctx)

	// Initialize pipeline and queue
	p := pipeline.New(a.browserMgr, a.db, a.cfg.DownloadDir)
	a.queueMgr = queue.NewManager(p, a.db)
	a.queueMgr.SetContext(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	if a.queueMgr != nil {
		a.queueMgr.Stop()
	}
	if a.browserMgr != nil {
		a.browserMgr.Disconnect()
	}
	if a.db != nil {
		a.db.Close()
	}
}

func (a *App) domReady(ctx context.Context) {
	runtime.EventsEmit(a.ctx, "app:ready", true)
}

// === Task bindings ===

func (a *App) CreateTask(prompt, aspectRatio, model string, outputCount int) (*database.Task, error) {
	return a.db.CreateTask(prompt, aspectRatio, model, outputCount)
}

func (a *App) CreateTasksBatch(prompts []string, aspectRatio, model string, outputCount int) ([]database.Task, error) {
	return a.db.CreateTasksBatch(prompts, aspectRatio, model, outputCount)
}

func (a *App) GetTask(id string) (*database.Task, error) {
	return a.db.GetTask(id)
}

func (a *App) ListTasks(filter database.TaskFilter) ([]database.Task, error) {
	return a.db.ListTasks(filter)
}

func (a *App) DeleteTask(id string) error {
	return a.db.DeleteTask(id)
}

func (a *App) GetTaskStats() (*database.TaskStats, error) {
	return a.db.GetTaskStats()
}

// GetVideoData returns base64-encoded video data for playback in WebView
func (a *App) GetVideoData(videoPath string) (string, error) {
	// Try as-is first, then try just filename in download dir
	absPath := videoPath
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		absPath = filepath.Join(a.cfg.DownloadDir, filepath.Base(videoPath))
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("cannot read video: %w", err)
	}
	return "data:video/mp4;base64," + base64.StdEncoding.EncodeToString(data), nil
}

// === Settings bindings ===

func (a *App) GetSettings() (map[string]string, error) {
	return a.db.GetSettings()
}

func (a *App) UpdateSetting(key, value string) error {
	if err := a.db.UpdateSetting(key, value); err != nil {
		return err
	}
	// Apply setting to running app immediately
	switch key {
	case "download_folder":
		a.cfg.DownloadDir = value
		if a.queueMgr != nil && a.queueMgr.Pipeline() != nil {
			a.queueMgr.Pipeline().SetDownloadDir(value)
		}
	case "chrome_path":
		a.cfg.ChromePath = value
		if a.browserMgr != nil {
			a.browserMgr.Config().ChromePath = value
		}
	case "user_data_dir":
		a.cfg.UserDataDir = value
		if a.browserMgr != nil {
			a.browserMgr.Config().UserDataDir = value
		}
	case "debug_port":
		a.cfg.DebugPort = value
		if a.browserMgr != nil {
			if p, err := strconv.Atoi(value); err == nil {
				a.browserMgr.Config().DebugPort = p
			}
		}
	case "delay_between_tasks":
		// Delay is read from DB at each task execution, no runtime update needed
	}
	return nil
}

func (a *App) UpdateSettings(settings map[string]string) error {
	return a.db.UpdateSettings(settings)
}

func (a *App) GetConfig() *config.AppConfig {
	return a.cfg
}

// === File/Folder Dialogs ===

func (a *App) SelectFile(title string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: "Executable (*.exe)", Pattern: "*.exe"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

func (a *App) SelectDirectory(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

// === Validation ===

func (a *App) ValidateChromePath(path string) string {
	if path == "" {
		// Auto-detect
		detected := chrome.DefaultChromeConfig().ChromePath
		if detected == "" {
			return "error:Không tìm thấy Chrome trên máy. Vui lòng cài Chrome hoặc nhập đường dẫn thủ công."
		}
		return "ok:Tự động phát hiện: " + detected
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "error:Đường dẫn không tồn tại: " + path
	}
	if info.IsDir() {
		return "error:Đây là thư mục, cần trỏ đến file chrome.exe"
	}
	return "ok:Đường dẫn hợp lệ: " + path
}

func (a *App) ValidateUserDataDir(path string) string {
	if path == "" {
		defaultDir := filepath.Join(os.Getenv("APPDATA"), "veo3-manager", "chrome-data")
		return "ok:Sử dụng mặc định: " + defaultDir
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		// Directory doesn't exist yet - that's OK, it will be created
		return "ok:Thư mục sẽ được tạo tự động: " + path
	}
	if !info.IsDir() {
		return "error:Đây là file, cần trỏ đến thư mục"
	}
	// Check if writable by trying to create a temp file
	testFile := filepath.Join(path, ".veo3_test_write")
	f, err := os.Create(testFile)
	if err != nil {
		return "error:Không có quyền ghi vào thư mục này"
	}
	f.Close()
	os.Remove(testFile)
	return "ok:Thư mục hợp lệ: " + path
}

func (a *App) ValidateDownloadFolder(path string) string {
	if path == "" {
		defaultDir := filepath.Join(os.Getenv("USERPROFILE"), "Downloads", "veo3-manager")
		return "ok:Sử dụng mặc định: " + defaultDir
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "ok:Thư mục sẽ được tạo tự động: " + path
	}
	if !info.IsDir() {
		return "error:Đây là file, cần trỏ đến thư mục"
	}
	testFile := filepath.Join(path, ".veo3_test_write")
	f, err := os.Create(testFile)
	if err != nil {
		return "error:Không có quyền ghi vào thư mục này"
	}
	f.Close()
	os.Remove(testFile)
	return "ok:Thư mục hợp lệ: " + path
}

// === Window controls ===

func (a *App) WindowMinimise() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) WindowToggleMaximise() {
	runtime.WindowToggleMaximise(a.ctx)
}

func (a *App) WindowClose() {
	runtime.Quit(a.ctx)
}

// === Browser controls ===

func (a *App) LaunchBrowser() error {
	return a.browserMgr.Connect()
}

func (a *App) DisconnectBrowser() error {
	return a.browserMgr.Disconnect()
}

func (a *App) GetBrowserStatus() string {
	return string(a.browserMgr.Status())
}

func (a *App) GetBrowserInfo() *chrome.BrowserInfo {
	return a.browserMgr.GetInfo()
}

// === Queue controls ===

func (a *App) StartQueue() error {
	return a.queueMgr.Start()
}

func (a *App) PauseQueue() {
	a.queueMgr.Pause()
}

func (a *App) ResumeQueue() {
	a.queueMgr.Resume()
}

func (a *App) StopQueue() {
	a.queueMgr.Stop()
}

func (a *App) GetQueueState() string {
	return string(a.queueMgr.State())
}
