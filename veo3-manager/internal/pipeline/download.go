package pipeline

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/go-rod/rod"
)

// DownloadVideo resolves the signed GCS URL via browser fetch and downloads the video
func DownloadVideo(page *rod.Page, mediaID, destPath string) error {
	// Step 1: Use browser fetch to follow redirect and get signed GCS URL
	// This needs browser cookies (labs.google session) for the redirect
	gcsURL, err := resolveDownloadURL(page, mediaID)
	if err != nil {
		return fmt.Errorf("resolve download URL: %w", err)
	}

	// Step 2: Download from GCS (signed URL, no auth needed)
	return downloadFile(gcsURL, destPath)
}

// resolveDownloadURL uses browser fetch() to follow the redirect and get the final signed GCS URL
func resolveDownloadURL(page *rod.Page, mediaID string) (string, error) {
	redirectURL := fmt.Sprintf("https://labs.google/fx/api/trpc/media.getMediaUrlRedirect?name=%s", mediaID)

	result, err := page.Eval(fmt.Sprintf(`() => {
		return fetch("%s", {redirect: "follow"})
			.then(r => r.url)
			.catch(e => "error:" + e.message);
	}`, redirectURL))
	if err != nil {
		return "", fmt.Errorf("fetch eval failed: %w", err)
	}

	url := result.Value.Str()
	if url == "" {
		return "", fmt.Errorf("empty URL returned from fetch")
	}
	if len(url) > 6 && url[:6] == "error:" {
		return "", fmt.Errorf("fetch error: %s", url[6:])
	}

	// Validate it's a GCS URL
	if len(url) < 40 || (url[:38] != "https://storage.googleapis.com/ai-sand" && url[:30] != "https://storage.googleapis.com") {
		return "", fmt.Errorf("unexpected download URL (not GCS): %s", url[:min(80, len(url))])
	}

	return url, nil
}

func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
