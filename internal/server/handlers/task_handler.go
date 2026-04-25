package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	lyricsModel := strings.TrimSpace(r.FormValue("lyrics_model"))
	composeModel := strings.TrimSpace(r.FormValue("compose_model"))

	// デフォルト値のフォールバック
	if lyricsModel == "" {
		lyricsModel = h.cfg.GeminiModel
	}
	if composeModel == "" {
		composeModel = h.cfg.LyriaModel
	}

	task := domain.Task{
		JobID:      r.FormValue("job_id"),
		RequestURL: r.FormValue("url"),
		InputText:  r.FormValue("text"),
		ImageURL:   r.FormValue("image"),
		CreatedAt:  time.Now().UTC(),
		AIModels: domain.AIModels{
			LyricsModel:  lyricsModel,
			ComposeModel: composeModel,
		},
	}

	// JobID が空の場合はタイムスタンプから生成
	if task.JobID == "" {
		task.JobID = time.Now().UTC().Format("20060102150405")
	}

	// Cloud Tasks 等へのエンキュー実行
	if err := h.taskEnqueuer.Enqueue(r.Context(), task); err != nil {
		http.Error(w, fmt.Sprintf("failed to enqueue task: %v", err), http.StatusInternalServerError)
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
