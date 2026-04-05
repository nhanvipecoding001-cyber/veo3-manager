package pipeline

// --- Submit types ---

type SubmitRequest struct {
	MediaGenerationContext MediaGenerationContext `json:"mediaGenerationContext"`
	ClientContext          ClientContext          `json:"clientContext"`
	Requests               []VideoRequest        `json:"requests"`
	UseV2ModelConfig       bool                  `json:"useV2ModelConfig"`
}

type MediaGenerationContext struct {
	BatchId string `json:"batchId"`
}

type ClientContext struct {
	ProjectId       string `json:"projectId"`
	Tool            string `json:"tool"`
	UserPaygateTier string `json:"userPaygateTier"`
	SessionId       string `json:"sessionId"`
}

type VideoRequest struct {
	AspectRatio   string    `json:"aspectRatio"`
	Seed          int       `json:"seed"`
	TextInput     TextInput `json:"textInput"`
	VideoModelKey string    `json:"videoModelKey"`
	Metadata      struct{}  `json:"metadata"`
}

type TextInput struct {
	StructuredPrompt StructuredPrompt `json:"structuredPrompt"`
}

type StructuredPrompt struct {
	Parts []PromptPart `json:"parts"`
}

type PromptPart struct {
	Text string `json:"text"`
}

// --- Submit response ---

type SubmitResponse struct {
	Operations []OperationEntry `json:"operations"`
	RemainingCredits int        `json:"remainingCredits"`
	Workflows  []Workflow       `json:"workflows"`
	Media      []MediaEntry     `json:"media"`
}

type OperationEntry struct {
	Operation struct {
		Name string `json:"name"` // this is the media_id
	} `json:"operation"`
	Status string `json:"status"`
}

type Workflow struct {
	Name      string `json:"name"`
	ProjectId string `json:"projectId"`
}

type MediaEntry struct {
	Name          string        `json:"name"`
	ProjectId     string        `json:"projectId"`
	WorkflowId    string        `json:"workflowId"`
	MediaMetadata MediaMetadata `json:"mediaMetadata"`
}

type MediaMetadata struct {
	MediaStatus struct {
		MediaGenerationStatus string `json:"mediaGenerationStatus"`
	} `json:"mediaStatus"`
}

// --- Poll types ---

type PollRequest struct {
	Media []PollMediaItem `json:"media"`
}

type PollMediaItem struct {
	Name      string `json:"name"`
	ProjectId string `json:"projectId"`
}

type PollResponse struct {
	Media []PollMediaResult `json:"media"`
}

type PollMediaResult struct {
	Name          string        `json:"name"`
	MediaMetadata MediaMetadata `json:"mediaMetadata"`
}

// --- Result types ---

type PollResult struct {
	AllDone  bool
	MediaIDs []string
	Statuses map[string]string // mediaID -> status
}
