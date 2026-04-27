package domain

import (
	"fmt"
	"strings"
	"time"
)

type AIModels struct {
	// TextModel は歌詞生成およびレシピ構築（LLM）に使用するモデル
	TextModel string `json:"text_model,omitempty"`
	// AudioModel は音声生成に使用するモデル
	AudioModel string `json:"audio_model,omitempty"`
	// ComposeMode は使用するプロンプトテンプレートのキー (assets/prompts)
	// 例: "default", "heroic", "techno"
	ComposeMode string `json:"compose_mode,omitempty"`
	Seed        *int64 `json:"seed,omitempty"`
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

// ValidateSubmission は、ジョブ投入前に最低限必要な入力が揃っていることを検証します。
func (t Task) ValidateSubmission() error {
	if strings.TrimSpace(t.RequestURL) == "" &&
		strings.TrimSpace(t.InputText) == "" &&
		strings.TrimSpace(t.ImageURL) == "" {
		return fmt.Errorf("at least one input is required: url, text, or image")
	}

	return nil
}

// PublishResult は生成結果です。
type PublishResult struct {
	JobID            string `json:"job_id"`
	StorageURI       string `json:"storage_uri"`
	SignedURL        string `json:"signed_url"`
	RecipeStorageURI string `json:"recipe_storage_uri"`
	RecipeSignedURL  string `json:"recipe_signed_url"`
}
