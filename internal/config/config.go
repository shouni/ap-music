package config

import (
	"fmt"
	"os"
	"time"
)

const (
	DefaultPort          = "8080"
	DefaultLyriaModel    = "lyria-3"
	DefaultShutdownGrace = 15 * time.Second

	DefaultHTTPTimeout = 60 * time.Second
)

// Config はアプリ設定です。
type Config struct {
	ServiceURL          string
	Port                string
	ProjectID           string
	LocationID          string
	QueueID             string
	TaskAudienceURL     string
	ServiceAccountEmail string
	GCSBucket           string
	SlackWebhookURL     string
	LyriaModel          string
	ShutdownTimeout     time.Duration

	// OAuth & Session Settings
	GoogleClientID     string
	GoogleClientSecret string
	// SessionSecret はセッションデータのHMAC署名用シークレットキーです。
	SessionSecret string
	// SessionEncryptKey はセッションデータのAES暗号化用シークレットキーです。 16, 24, 32 バイトのいずれかである必要があります。
	SessionEncryptKey string

	// Authz Settings
	AllowedEmails  []string
	AllowedDomains []string
}

// LoadConfig は環境変数から設定を読み込みます。
func LoadConfig() (Config, error) {
	cfg := Config{
		ServiceURL:          os.Getenv("SERVICE_URL"),
		Port:                getenvDefault("PORT", DefaultPort),
		ProjectID:           os.Getenv("GCP_PROJECT_ID"),
		LocationID:          os.Getenv("GCP_LOCATION_ID"),
		QueueID:             os.Getenv("CLOUD_TASKS_QUEUE_ID"),
		TaskAudienceURL:     os.Getenv("TASK_AUDIENCE_URL"),
		ServiceAccountEmail: os.Getenv("SERVICE_ACCOUNT_EMAIL"),
		GCSBucket:           os.Getenv("GCS_MUSIC_BUCKET"),
		SlackWebhookURL:     os.Getenv("SLACK_WEBHOOK_URL"),
		LyriaModel:          getenvDefault("LYRIA_MODEL", DefaultLyriaModel),
		ShutdownTimeout:     DefaultShutdownGrace,
	}

	required := map[string]string{
		"SERVICE_URL":           cfg.ServiceURL,
		"GCP_PROJECT_ID":        cfg.ProjectID,
		"GCP_LOCATION_ID":       cfg.LocationID,
		"CLOUD_TASKS_QUEUE_ID":  cfg.QueueID,
		"SERVICE_ACCOUNT_EMAIL": cfg.ServiceAccountEmail,
		"GCS_MUSIC_BUCKET":      cfg.GCSBucket,
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
