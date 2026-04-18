package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"ap-music/internal/domain"
)

// EnqueueTask はフォーム入力をジョブ化してキューに積みます。
func (h *Handler) EnqueueTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	task := domain.Task{
		JobID:      r.FormValue("job_id"),
		RequestURL: r.FormValue("url"),
		InputText:  r.FormValue("text"),
		ImageURL:   r.FormValue("image"),
		Model:      r.FormValue("model"),
		CreatedAt:  time.Now().UTC(),
	}
	if task.JobID == "" {
		task.JobID = time.Now().UTC().Format("20060102150405")
	}

	if err := h.taskEnqueuer.Enqueue(r.Context(), task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "queued", "job_id": task.JobID})
}
