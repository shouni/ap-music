package worker

import (
	"encoding/json"
	"net/http"

	"ap-music/internal/domain"
	"ap-music/internal/pipeline"
)

// Handler は Worker エンドポイントを提供します。
type Handler struct {
	Workflow pipeline.Workflow
}

// ExecuteTask はタスクを実行します。
func (h Handler) ExecuteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var task domain.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.Workflow.Execute(r.Context(), task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
