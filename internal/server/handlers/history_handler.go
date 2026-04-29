package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ap-music/internal/domain"
)

// ServeHistory は楽曲生成履歴の一覧画面を表示するのだ。
func (h *Handler) ServeHistory(w http.ResponseWriter, r *http.Request) {
	// 1. 本来はセッションからuserIDを取得します
	// userID := h.getUserIDFromSession(r)
	// TODO: 認証機能実装後、セッションまたはリクエストコンテキストからuserIDを取得するように修正する
	userID := "me"
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

	audioURL := fmt.Sprintf("/web/audio/%s", jobID)

	data := struct {
		ID       string
		Recipe   *domain.MusicRecipe
		AudioURL string
	}{
		ID:       jobID,
		Recipe:   recipe,
		AudioURL: audioURL,
	}

	h.render(w, http.StatusOK, "music_view.html", recipe.Title, data)
}

// ServeAudio は、指定されたリクエストに基づいて生成または取得された音声ファイルをクライアントに返します。
// HTTP レスポンスとして音声データ（WAV等）を書き込みます。
func (h *Handler) ServeAudio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobID := chi.URLParam(r, "jobID")
	if jobID == "" {
		http.Error(w, "JobID is required", http.StatusBadRequest)
		return
	}

	fileName := fmt.Sprintf("%s.wav", jobID)
	gcsURL := h.cfg.GetGCSObjectURL(fileName)
	reader, err := h.remoteIO.Reader.Open(ctx, gcsURL)
	if err != nil {
		slog.Error("Failed to open GCS object",
			"url", gcsURL,
			"jobID", jobID,
			"error", err,
		)
		http.Error(w, "Audio file not found", http.StatusNotFound)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if _, err := io.Copy(w, reader); err != nil {
		slog.Error("Stream transfer error", "jobID", jobID, "error", err)
		// ヘッダー送信後のエラーは http.Error では返せないためログのみ
		return
	}
}
