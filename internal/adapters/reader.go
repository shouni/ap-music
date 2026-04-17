package adapters

import (
	"context"
	"fmt"

	"ap-music/internal/domain"
)

// ReaderAdapter は入力情報を収集します。
type ReaderAdapter struct{}

// Collect は URL/Text/Image の入力を結合します。
func (ReaderAdapter) Collect(_ context.Context, task domain.Task) (string, error) {
	if task.RequestURL == "" && task.InputText == "" && task.ImageURL == "" {
		return "", fmt.Errorf("no input provided")
	}
	return fmt.Sprintf("url=%s\ntext=%s\nimage=%s", task.RequestURL, task.InputText, task.ImageURL), nil
}
