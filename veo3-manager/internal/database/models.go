package database

import (
	"database/sql"
	"encoding/json"
	"time"
)

type StringSlice []string

func (s StringSlice) Value() (string, error) {
	if s == nil {
		return "[]", nil
	}
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *StringSlice) Scan(src interface{}) error {
	if src == nil {
		*s = StringSlice{}
		return nil
	}
	var raw string
	switch v := src.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		*s = StringSlice{}
		return nil
	}
	return json.Unmarshal([]byte(raw), s)
}

type Task struct {
	ID           string       `json:"id"`
	Prompt       string       `json:"prompt"`
	Status       string       `json:"status"`
	AspectRatio  string       `json:"aspectRatio"`
	Model        string       `json:"model"`
	OutputCount  int          `json:"outputCount"`
	MediaIDs     StringSlice  `json:"mediaIds"`
	VideoPaths   StringSlice  `json:"videoPaths"`
	ErrorMessage string       `json:"errorMessage"`
	Seed         string       `json:"seed"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	CompletedAt  sql.NullTime `json:"completedAt"`
}

type TaskFilter struct {
	Status string `json:"status"`
	Search string `json:"search"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type TaskStats struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
}
