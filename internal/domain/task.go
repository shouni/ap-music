package domain

import "time"

// Task は生成ジョブです。
type Task struct {
	JobID      string            `json:"job_id"`
	RequestURL string            `json:"request_url,omitempty"`
	InputText  string            `json:"input_text,omitempty"`
	ImageURL   string            `json:"image_url,omitempty"`
	Model      string            `json:"model,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
}

// PublishResult は生成結果です。
type PublishResult struct {
	JobID      string `json:"job_id"`
	StorageURI string `json:"storage_uri"`
	SignedURL  string `json:"signed_url"`
}
