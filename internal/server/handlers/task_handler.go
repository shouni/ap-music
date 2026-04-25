package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"ap-music/internal/domain"
)

// EnqueueTask はフォーム入力をジョブ化してキューに積みます。
func (h *Handler) EnqueueTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	// 役割ごとにモデル入力を取得
	textModel := strings.TrimSpace(r.FormValue("lyrics_model"))
	audioModel := strings.TrimSpace(r.FormValue("compose_model"))

	// デフォルト値のフォールバック
	if textModel == "" {
		textModel = h.cfg.GeminiModel
	}
	if audioModel == "" {
		audioModel = h.cfg.LyriaModel
	}

	task := domain.Task{
		JobID:      r.FormValue("job_id"),
		RequestURL: r.FormValue("url"),
		InputText:  r.FormValue("text"),
		ImageURL:   r.FormValue("image"),
		CreatedAt:  time.Now().UTC(),
		AIModels: domain.AIModels{
			TextModel:  textModel,
			AudioModel: audioModel,
		},
	}

	if task.JobID == "" {
		task.JobID = uuid.New().String()
	}

	// Cloud Tasks 等へのエンキュー実行
	if err := h.taskEnqueuer.Enqueue(r.Context(), task); err != nil {
		slog.Error("failed to enqueue task", "error", err)
		http.Error(w, "Internal Server Error: failed to enqueue task", http.StatusInternalServerError)
		return
	}

	// Acceptヘッダーを確認し、JSONを要求している場合はJSONでレスポンス
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "queued",
			"job_id": task.JobID,
		})
		return
	}

	// HTML レンダリング
	h.render(w, http.StatusAccepted, "accepted.html", "タスク受付完了", task)
}
