package adapters

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	slackClient *slack.Client
}

// NewSlackAdapter は新しいアダプターインスタンスを作成します。
func NewSlackAdapter(httpClient httpkit.Requester, webhookURL string) (*SlackAdapter, error) {
	if webhookURL == "" {
		// オプショナル機能として扱い、空のままインスタンスを返す
		return &SlackAdapter{}, nil
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
		slackClient: client,
	}, nil
}

// Notify は処理完了時のSlack通知を送信します。
func (s *SlackAdapter) Notify(ctx context.Context, result *domain.PublishResult) error {
	if result == nil {
		return fmt.Errorf("publish result is nil")
	}
	return s.NotifyWithRequest(ctx, result.SignedURL, result.StorageURI, domain.NotificationRequest{})
}

// NotifyWithRequest は詳細情報付きでSlack通知を送信します。
func (s *SlackAdapter) NotifyWithRequest(ctx context.Context, publicURL, storageURI string, req domain.NotificationRequest) error {
	if s.webhookURL == "" || s.slackClient == nil {
		slog.InfoContext(ctx, "Slack通知が無効化されているか、クライアントが未初期化のためスキップします。", "storage_uri", storageURI)
		return nil
	}

	title := fmt.Sprintf("%s 処理が完了しました！", "🎼")
	content := s.buildSlackContent(publicURL, storageURI, req)

	if err := s.slackClient.SendTextWithHeader(ctx, title, content); err != nil {
		return fmt.Errorf("Slackへの投稿に失敗しました: %w", err)
	}

	slog.Info("Slack に完了通知を送信しました。", "public_url", publicURL)
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
	if req.SourceURL != "" {
		sb.WriteString(fmt.Sprintf("*ソース:* %s\n", req.SourceURL))
	}
	if req.OutputCategory != "" {
		sb.WriteString(fmt.Sprintf("*カテゴリ:* %s\n", req.OutputCategory))
	}
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

// buildSlackContent 指定された公開URL、ストレージURI、通知リクエストに基づき、Slack メッセージの内容を生成します。
func (s *SlackAdapter) buildSlackContent(publicURL, storageURI string, req domain.NotificationRequest) string {
	var sb strings.Builder

	if req.SourceURL != "" {
		sb.WriteString(fmt.Sprintf("*ソース:* %s\n", req.SourceURL))
	}
	if req.OutputCategory != "" {
		sb.WriteString(fmt.Sprintf("*カテゴリ:* %s\n", req.OutputCategory))
	}
	if publicURL != "" {
		sb.WriteString(fmt.Sprintf("*再生URL:* %s\n", publicURL))
	}
	if storageURI != "" {
		sb.WriteString(fmt.Sprintf("*Storage URI:* %s\n", storageURI))
	}
	if sb.Len() == 0 {
		sb.WriteString(domain.NotAvailable)
	}

	return sb.String()
}
