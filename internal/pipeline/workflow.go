package pipeline

import (
	"context"

	"ap-music/internal/domain"
)

// Workflow は実行フローの共通インターフェースです。
type Workflow interface {
	Execute(ctx context.Context, task domain.Task) (domain.PublishResult, error)
}
