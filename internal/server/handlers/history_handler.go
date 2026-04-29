package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"

	"ap-music/internal/domain"
)

// jobIDRegex は JobID のバリデーション用正規表現です。
// 英数字とハイフンのみを許可し、パストラバーサル等を防ぎます。
var jobIDRegex = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)

// ServeHistory は楽曲生成履歴の一覧画面を表示するのだ。
func (h *Handler) ServeHistory(w http.ResponseWriter, r *http.Request) {
	// TODO: 認証機能実装後、セッションから userID を取得するように修正する
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

	// バリデーション：セキュリティとパスの安全性を確保
	if !jobIDRegex.MatchString(jobID) {
		http.Error(w, "Invalid JobID format", http.StatusBadRequest)
		return
	}

	recipe, err := h.musicRepo.GetRecipe(r.Context(), jobID)
	if err != nil {
		http.Error(w, "レシピが見つからないのだ", http.StatusNotFound)
		return
	}

	// テンプレートの JSON タブに表示するための整形済み JSON を作成
	recipeJSON, err := json.MarshalIndent(recipe, "", "  ")
	if err != nil {
		slog.ErrorContext(r.Context(), "JSONの整形に失敗したのだ", "jobID", jobID, "error", err)
		recipeJSON = []byte("{}")
	}

	audioURL := fmt.Sprintf("/web/audio/%s", jobID)

	data := struct {
		ID         string
		Recipe     *domain.MusicRecipe
		RecipeJSON string
		AudioURL   string
	}{
		ID:         jobID,
		Recipe:     recipe,
		RecipeJSON: string(recipeJSON),
		AudioURL:   audioURL,
	}

	h.render(w, http.StatusOK, "music_view.html", recipe.Title, data)
}

// ServeAudio は、GCSの署名付きURLへリダイレクトさせることで音声を配信します。
func (h *Handler) ServeAudio(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobID := chi.URLParam(r, "jobID")

	// バリデーション：共通の正規表現を使用
	if !jobIDRegex.MatchString(jobID) {
		http.Error(w, "Invalid JobID format", http.StatusBadRequest)
		return
	}

	// ファイル名の組み立て
	fileName := fmt.Sprintf("%s.wav", jobID)
	gcsURL := h.cfg.GetGCSObjectURL(fileName)

	// 署名付きURLの生成（直接GCSから配信させてサーバー負荷を軽減）
	signedURL, err := h.remoteIO.Signer.GenerateSignedURL(
		ctx,
		gcsURL,
		http.MethodGet,
		1*time.Hour,
	)
	if err != nil {
		slog.Error("Failed to generate signed URL", "jobID", jobID, "error", err)
		http.Error(w, "Audio access error", http.StatusInternalServerError)
		return
	}

	// クライアントを直接 GCS へ飛ばす
	http.Redirect(w, r, signedURL, http.StatusFound)
}
