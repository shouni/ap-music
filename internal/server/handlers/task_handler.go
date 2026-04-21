package handlers

import (
	"encoding/json"
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

	selectedModel := strings.TrimSpace(r.FormValue("model"))
	if selectedModel == "" {
		selectedModel = h.cfg.LyriaModel
	}

	task := domain.Task{
		JobID:      r.FormValue("job_id"),
		RequestURL: r.FormValue("url"),
		InputText:  r.FormValue("text"),
		ImageURL:   r.FormValue("image"),
		Model:      selectedModel,
		CreatedAt:  time.Now().UTC(),
	}
	if task.JobID == "" {
		task.JobID = time.Now().UTC().Format("20060102150405")
	}

	if err := h.taskEnqueuer.Enqueue(r.Context(), task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Acceptヘッダーを確認し、JSONを要求している場合は既存の挙動を維持する
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "queued", "job_id": task.JobID})
		return
	}

	h.render(w, http.StatusAccepted, "accepted.html", "タスク受付完了", task)
}
