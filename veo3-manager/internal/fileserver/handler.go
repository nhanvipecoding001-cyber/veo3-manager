package fileserver

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Handler struct {
	allowedDir string
}

func NewHandler(allowedDir string) *Handler {
	return &Handler{allowedDir: allowedDir}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[FileServer] Request: %s\n", r.URL.Path)
	// Handle /localfile/ requests — accepts both full path and filename only
	if !strings.HasPrefix(r.URL.Path, "/localfile/") {
		return
	}

	requestedPath := strings.TrimPrefix(r.URL.Path, "/localfile/")
	requestedPath, _ = url.PathUnescape(requestedPath)

	// Extract just the filename — serves from allowedDir
	filename := filepath.Base(requestedPath)
	absPath := filepath.Join(h.allowedDir, filename)

	// Check file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// Fallback: try as full path
		fullPath := strings.ReplaceAll(requestedPath, "/", string(os.PathSeparator))
		absFullPath, err2 := filepath.Abs(fullPath)
		if err2 != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		allowedAbs, _ := filepath.Abs(h.allowedDir)
		if !strings.HasPrefix(strings.ToLower(absFullPath), strings.ToLower(allowedAbs)) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
		if _, err3 := os.Stat(absFullPath); os.IsNotExist(err3) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		absPath = absFullPath
	}

	fmt.Printf("[FileServer] Serving: %s\n", absPath)
	w.Header().Set("Content-Type", "video/mp4")
	http.ServeFile(w, r, absPath)
}
