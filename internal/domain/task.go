package domain

import "time"

type AIModels struct {
	// LyricsModel は歌詞生成に使用するモデル
	LyricsModel string `json:"lyrics_model,omitempty"`
	// ComposeModel は音声生成に使用するモデル
	ComposeModel string `json:"compose_model,omitempty"`
}

// Task は生成ジョブです。
type Task struct {
	JobID      string            `json:"job_id"`
	RequestURL string            `json:"request_url,omitempty"`
	InputText  string            `json:"input_text,omitempty"`
	ImageURL   string            `json:"image_url,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	AIModels
}

// PublishResult は生成結果です。
type PublishResult struct {
	JobID            string `json:"job_id"`
	StorageURI       string `json:"storage_uri"`
	SignedURL        string `json:"signed_url"`
	RecipeStorageURI string `json:"recipe_storage_uri"`
	RecipeSignedURL  string `json:"recipe_signed_url"`
}
