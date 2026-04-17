package adapters

import (
	"context"

	"ap-music/internal/domain"
)

// SlackAdapter は Slack 通知アダプタの雛形です。
type SlackAdapter struct {
	WebhookURL string
}

// Notify は通知を行います。
func (a SlackAdapter) Notify(_ context.Context, _ domain.PublishResult) error {
	_ = a.WebhookURL
	return nil
}
