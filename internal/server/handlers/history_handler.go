package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

// ServeHistory は楽曲生成履歴の一覧画面を表示するのだ。
func (h *Handler) ServeHistory(w http.ResponseWriter, r *http.Request) {
	// 1. 本来はセッションからuserIDを取得します
	// userID := h.getUserIDFromSession(r)
	userID := "me" // 一人運用なら固定でもOKです
	histories, err := h.musicRepo.ListHistory(r.Context(), userID)
	if err != nil {
		http.Error(w, "履歴の取得に失敗したのだ", http.StatusInternalServerError)
		return
	}

	h.render(w, http.StatusOK, "history.html", "History", histories)
}

// ServeDetails は特定の楽曲の詳細画面を表示するのだ。
func (h *Handler) ServeDetails(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	recipe, err := h.musicRepo.GetRecipe(r.Context(), jobID)
	if err != nil {
		http.Error(w, "レシピが見つからないのだ", http.StatusNotFound)
		return
	}
	audioURL, _ := h.generateAudioSignedURL(r.Context(), jobID)

	data := struct {
		Recipe   *domain.MusicRecipe
		AudioURL string
	}{
		Recipe:   recipe,
		AudioURL: audioURL,
	}

	h.render(w, http.StatusOK, "music_view.html", recipe.Title, data)
}

// generateAudioSignedURL は、フラットなファイル名規則（timestamp-id.wav）に基づいて署名付きURLを生成します。
func (h *Handler) generateAudioSignedURL(ctx context.Context, jobID string) (string, error) {
	fileName := fmt.Sprintf("%s.wav", jobID)
	gcsURL := h.cfg.GetGCSObjectURL(fileName)
	signedURL, err := h.remoteIO.Signer.GenerateSignedURL(
		ctx,
		gcsURL,
		http.MethodGet,
		config.SignedURLExpiration,
	)
	if err != nil {
		return "", fmt.Errorf("音声ファイルの署名に失敗したのだ: %w", err)
	}

	return signedURL, nil
}
