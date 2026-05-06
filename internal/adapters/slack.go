package adapters

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/shouni/go-http-kit/httpkit"
	"github.com/shouni/go-notifier/pkg/slack"

	"ap-music/internal/domain"
)

const (
	slackErrorTitle         = "❌ 処理中にエラーが発生しました"
	slackErrorContentHeader = "*エラー内容:*\n"
)

// SlackAdapter は、Slack APIと連携し、Webhookを介してメッセージを投稿するためのアダプタを表します。
type SlackAdapter struct {
	webhookURL  string
	serviceURL  string
	slackClient *slack.Client
}

// NewSlackAdapter は新しいアダプターインスタンスを作成します。
func NewSlackAdapter(httpClient httpkit.Requester, webhookURL, serviceURL string) (*SlackAdapter, error) {
	if webhookURL == "" {
		return &SlackAdapter{serviceURL: serviceURL}, nil
	}

	if httpClient == nil {
		return nil, errors.New("HTTPクライアントがnilです")
	}

	client, err := slack.NewClient(httpClient, webhookURL)
	if err != nil {
		return nil, fmt.Errorf("Slackクライアントの初期化に失敗しました: %w", err)
	}

	return &SlackAdapter{
		webhookURL:  webhookURL,
		serviceURL:  serviceURL,
		slackClient: client,
	}, nil
}

// Notify は処理完了時の標準的なSlack通知を送信します。
func (s *SlackAdapter) Notify(ctx context.Context, result *domain.PublishResult) error {
	return s.NotifyWithRequest(ctx, result, domain.NotificationRequest{})
}

// NotifyWithRequest は詳細情報（NotificationRequest）付きでSlack通知を送信します。
func (s *SlackAdapter) NotifyWithRequest(ctx context.Context, result *domain.PublishResult, req domain.NotificationRequest) error {
	if result == nil {
		return fmt.Errorf("publish result is nil")
	}

	if s.webhookURL == "" || s.slackClient == nil {
		slog.InfoContext(ctx, "Slack通知が無効化されているか、クライアントが未初期化のためスキップします。", "storage_uri", result.StorageURI)
		return nil
	}

	title := fmt.Sprintf("%s 処理が完了しました！", "🎼")
	content := s.buildSlackContent(result, req)

	if err := s.slackClient.SendTextWithHeader(ctx, title, content); err != nil {
		return fmt.Errorf("Slackへの投稿に失敗しました: %w", err)
	}

	slog.Info("Slack に完了通知を送信しました。",
		"public_url", result.SignedURL,
		"recipe_url", result.RecipeSignedURL,
		"job_id", result.JobID,
	)
	return nil
}

// NotifyError エラー詳細と実行メタデータを含むSlackエラー通知の送信。
func (s *SlackAdapter) NotifyError(ctx context.Context, errDetail error, req domain.NotificationRequest) error {
	if s.slackClient == nil {
		slog.Info("Slackクライアントが初期化されていないため、エラー通知をスキップします。", "error", errDetail)
		return nil
	}

	title := slackErrorTitle
	var sb strings.Builder
	writeSlackRequestMetadata(&sb, req)

	if sb.Len() > 0 {
		sb.WriteString("\n")
	}
	sb.WriteString(slackErrorContentHeader)
	if errDetail != nil {
		sb.WriteString(errDetail.Error())
	} else {
		sb.WriteString(domain.NotAvailable)
	}
	content := sb.String()

	if err := s.slackClient.SendTextWithHeader(ctx, title, content); err != nil {
		return fmt.Errorf("Slackへのエラー通知に失敗しました: %w", err)
	}

	slog.Info("Slack にエラー通知を送信しました。", "error", errDetail)
	return nil
}

// buildSlackContent 指定された結果とリクエストに基づき、Slack メッセージの内容を生成します。
func (s *SlackAdapter) buildSlackContent(result *domain.PublishResult, req domain.NotificationRequest) string {
	var sb strings.Builder

	writeSlackRequestMetadata(&sb, req)

	if historyURL := s.historyDetailURL(result.JobID); historyURL != "" {
		sb.WriteString(fmt.Sprintf("*History Detail:* <%s|%s>\n", historyURL, result.JobID))
	}

	// 音楽ファイルのリンク
	if result.SignedURL != "" && result.StorageURI != "" {
		sb.WriteString(fmt.Sprintf("*WAV File:* <%s|%s>\n", result.SignedURL, result.StorageURI))
	} else if result.StorageURI != "" {
		sb.WriteString(fmt.Sprintf("*Storage URI:* %s\n", result.StorageURI))
	}

	// レシピ JSON のリンク
	if result.RecipeSignedURL != "" && result.RecipeStorageURI != "" {
		sb.WriteString(fmt.Sprintf("*Recipe JSON:* <%s|%s>\n", result.RecipeSignedURL, result.RecipeStorageURI))
	} else if result.RecipeStorageURI != "" {
		sb.WriteString(fmt.Sprintf("*Recipe Storage URI:* %s\n", result.RecipeStorageURI))
	}

	if sb.Len() == 0 {
		sb.WriteString(domain.NotAvailable)
	}

	return sb.String()
}

func writeSlackRequestMetadata(sb *strings.Builder, req domain.NotificationRequest) {
	if req.Command != "" {
		sb.WriteString(fmt.Sprintf("*Command:* `%s`\n", req.Command))
	}
	if req.Title != "" {
		sb.WriteString(fmt.Sprintf("*Title:* %s\n", req.Title))
	}
	if req.SourceURL != "" {
		sb.WriteString(fmt.Sprintf("*Source:* %s\n", req.SourceURL))
	}
	if req.Mode != "" {
		sb.WriteString(fmt.Sprintf("*Mode:* `%s`\n", req.Mode))
	}

	if req.Seed != nil {
		sb.WriteString(fmt.Sprintf("*Seed:* `%d` 🎲\n", *req.Seed))
	}
}

func (s *SlackAdapter) historyDetailURL(jobID string) string {
	if s.serviceURL == "" || jobID == "" {
		return ""
	}

	historyURL, err := url.JoinPath(s.serviceURL, "/web/history", jobID)
	if err != nil {
		return ""
	}

	return historyURL
}
