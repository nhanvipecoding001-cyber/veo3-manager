package config

import (
	"os"
	"path/filepath"
)

type AppConfig struct {
	ChromePath   string `json:"chromePath"`
	UserDataDir  string `json:"userDataDir"`
	DownloadDir  string `json:"downloadDir"`
	DebugPort    string `json:"debugPort"`
	AspectRatio  string `json:"aspectRatio"`
	Model        string `json:"model"`
	OutputCount  int    `json:"outputCount"`
	DBPath       string `json:"dbPath"`
}

func DefaultConfig() *AppConfig {
	appData, _ := os.UserConfigDir()
	baseDir := filepath.Join(appData, "veo3-manager")
	os.MkdirAll(baseDir, 0755)
	os.MkdirAll(filepath.Join(baseDir, "videos"), 0755)
	os.MkdirAll(filepath.Join(baseDir, "chrome-data"), 0755)

	return &AppConfig{
		ChromePath:  "",
		UserDataDir: filepath.Join(baseDir, "chrome-data"),
		DownloadDir: filepath.Join(baseDir, "videos"),
		DebugPort:   "9222",
		AspectRatio: "16:9",
		Model:       "veo_3_1_t2v_lite",
		OutputCount: 4,
		DBPath:      filepath.Join(baseDir, "veo3.db"),
	}
}
