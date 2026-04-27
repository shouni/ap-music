package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
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
	if err := h.crossOriginProtection.Check(r); err != nil {
		http.Error(w, "cross-origin request forbidden", http.StatusForbidden)
		return
	}

	task := h.taskFactory.Build(r.Form)
	if err := task.ValidateSubmission(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Cloud Tasks 等へのエンキュー実行
	if err := h.taskEnqueuer.Enqueue(r.Context(), task); err != nil {
		slog.Error("failed to enqueue task", "job_id", task.JobID, "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
