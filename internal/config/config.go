package config

import (
	"fmt"
	"os"
)

// Config はアプリ設定です。
type Config struct {
	Addr            string
	ServiceURL      string
	LyriaModel      string
	SlackWebhookURL string
	GCPProjectID    string
	GCPLocationID   string
	CloudTasksQueue string
	ServiceAccount  string
	TaskAudienceURL string
	GCSMusicBucket  string
}

// LoadConfig は環境変数から設定を読み込みます。
func LoadConfig() (Config, error) {
	cfg := Config{
		Addr:            getenvDefault("ADDR", ":8080"),
		ServiceURL:      os.Getenv("SERVICE_URL"),
		LyriaModel:      getenvDefault("LYRIA_MODEL", "lyria-3"),
		SlackWebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		GCPProjectID:    os.Getenv("GCP_PROJECT_ID"),
		GCPLocationID:   os.Getenv("GCP_LOCATION_ID"),
		CloudTasksQueue: os.Getenv("CLOUD_TASKS_QUEUE_ID"),
		ServiceAccount:  os.Getenv("SERVICE_ACCOUNT_EMAIL"),
		TaskAudienceURL: os.Getenv("TASK_AUDIENCE_URL"),
		GCSMusicBucket:  os.Getenv("GCS_MUSIC_BUCKET"),
	}

	required := map[string]string{
		"SERVICE_URL":           cfg.ServiceURL,
		"GCP_PROJECT_ID":        cfg.GCPProjectID,
		"GCP_LOCATION_ID":       cfg.GCPLocationID,
		"CLOUD_TASKS_QUEUE_ID":  cfg.CloudTasksQueue,
		"SERVICE_ACCOUNT_EMAIL": cfg.ServiceAccount,
		"GCS_MUSIC_BUCKET":      cfg.GCSMusicBucket,
	}
	for key, value := range required {
		if value == "" {
			return Config{}, fmt.Errorf("missing required env: %s", key)
		}
	}

	return cfg, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
