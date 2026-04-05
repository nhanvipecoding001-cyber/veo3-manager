package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	apiBase          = "https://aisandbox-pa.googleapis.com/v1"
	pollInterval     = 10 * time.Second
	pollTimeout      = 5 * time.Minute
	statusSuccessful = "MEDIA_GENERATION_STATUS_SUCCESSFUL"
	statusFailed     = "MEDIA_GENERATION_STATUS_FAILED"
)

// MapAspectRatio converts user-facing ratio to API enum
func MapAspectRatio(ratio string) string {
	switch ratio {
	case "16:9", "crop_16_9":
		return "VIDEO_ASPECT_RATIO_LANDSCAPE"
	case "9:16", "crop_9_16":
		return "VIDEO_ASPECT_RATIO_PORTRAIT"
	case "1:1", "crop_square":
		return "VIDEO_ASPECT_RATIO_SQUARE"
	case "4:3", "crop_landscape":
		return "VIDEO_ASPECT_RATIO_4_3"
	case "3:4", "crop_portrait":
		return "VIDEO_ASPECT_RATIO_3_4"
	default:
		return "VIDEO_ASPECT_RATIO_LANDSCAPE"
	}
}

// mapModelKey maps user-facing model names to API model keys
func mapModelKey(model string) string {
	switch model {
	case "veo_3_1_t2v_lite", "Veo 3.1 - Lite", "lite":
		return "veo_3_1_t2v_lite"
	case "veo_3_1_t2v_fast", "Veo 3.1 - Fast", "fast", "":
		return "veo_3_1_t2v_fast"
	case "veo_3_1_t2v_quality", "Veo 3.1 - Quality", "quality":
		return "veo_3_1_t2v_quality"
	default:
		return "veo_3_1_t2v_fast"
	}
}

// SubmitVideo submits a single video generation request
func SubmitVideo(token, projectId, prompt, model, aspectRatio string, seed int) (*SubmitResponse, error) {
	batchId := uuid.New().String()
	sessionId := fmt.Sprintf(";%d", time.Now().UnixMilli())

	req := SubmitRequest{
		MediaGenerationContext: MediaGenerationContext{BatchId: batchId},
		ClientContext: ClientContext{
			ProjectId:       projectId,
			Tool:            "PINHOLE",
			UserPaygateTier: "PAYGATE_TIER_NOT_PAID",
			SessionId:       sessionId,
		},
		Requests: []VideoRequest{{
			AspectRatio:   MapAspectRatio(aspectRatio),
			Seed:          seed,
			TextInput:     TextInput{StructuredPrompt: StructuredPrompt{Parts: []PromptPart{{Text: prompt}}}},
			VideoModelKey: mapModelKey(model),
			Metadata:      struct{}{},
		}},
		UseV2ModelConfig: true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal submit request: %w", err)
	}

	return doSubmitRequest(token, body)
}

func doSubmitRequest(token string, body []byte) (*SubmitResponse, error) {
	httpReq, err := http.NewRequest("POST", apiBase+"/video:batchAsyncGenerateVideoText", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	httpReq.Header.Set("Referer", "https://labs.google/")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("submit request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("token expired (401)")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("submit API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result SubmitResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse submit response: %w", err)
	}
	return &result, nil
}

// SubmitBatch submits N video generation requests for the same prompt (different seeds)
func SubmitBatch(token, projectId, prompt, model, aspectRatio string, count int) ([]string, error) {
	var mediaIDs []string

	for i := 0; i < count; i++ {
		seed := rand.Intn(10000)
		result, err := SubmitVideo(token, projectId, prompt, model, aspectRatio, seed)
		if err != nil {
			return mediaIDs, fmt.Errorf("submit video %d/%d failed: %w", i+1, count, err)
		}

		for _, op := range result.Operations {
			mediaIDs = append(mediaIDs, op.Operation.Name)
		}

		if i < count-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return mediaIDs, nil
}

// PollStatus polls the API every 10s until all videos are done or 5 min timeout
func PollStatus(ctx context.Context, token, projectId string, mediaIDs []string, onProgress func(status string)) (*PollResult, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	timeout := time.After(pollTimeout)

	for {
		select {
		case <-ticker.C:
			result, err := checkStatus(token, projectId, mediaIDs)
			if err != nil {
				if strings.Contains(err.Error(), "401") {
					return nil, fmt.Errorf("token expired (401)")
				}
				if onProgress != nil {
					onProgress(fmt.Sprintf("Poll error: %v, retrying...", err))
				}
				continue
			}

			doneCount := 0
			for _, id := range mediaIDs {
				status := result.Statuses[id]
				if status == statusSuccessful {
					doneCount++
				}
				if strings.Contains(status, "FAILED") {
					return nil, fmt.Errorf("video %s generation failed: %s", id[:8], status)
				}
			}

			if onProgress != nil {
				onProgress(fmt.Sprintf("Generating: %d/%d complete", doneCount, len(mediaIDs)))
			}

			if doneCount == len(mediaIDs) {
				result.AllDone = true
				return result, nil
			}

		case <-timeout:
			return nil, fmt.Errorf("polling timeout after 5 minutes")

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func checkStatus(token, projectId string, mediaIDs []string) (*PollResult, error) {
	items := make([]PollMediaItem, len(mediaIDs))
	for i, id := range mediaIDs {
		items[i] = PollMediaItem{Name: id, ProjectId: projectId}
	}

	body, err := json.Marshal(PollRequest{Media: items})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", apiBase+"/video:batchCheckAsyncVideoGenerationStatus", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	httpReq.Header.Set("Referer", "https://labs.google/")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("token expired (401)")
	}

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("poll API error %d: %s", resp.StatusCode, string(respBody))
	}

	var pollResp PollResponse
	if err := json.Unmarshal(respBody, &pollResp); err != nil {
		return nil, fmt.Errorf("parse poll response: %w", err)
	}

	result := &PollResult{
		Statuses: make(map[string]string),
	}
	for _, m := range pollResp.Media {
		result.MediaIDs = append(result.MediaIDs, m.Name)
		result.Statuses[m.Name] = m.MediaMetadata.MediaStatus.MediaGenerationStatus
	}

	return result, nil
}
