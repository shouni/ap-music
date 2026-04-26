package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"strconv"
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

	var seedPtr *int64
	seedValStr := strings.TrimSpace(r.FormValue("seed"))

	if seedValStr != "" {
		// ユーザー指定がある場合
		if val, err := strconv.ParseInt(seedValStr, 10, 64); err == nil {
			seedPtr = &val
		} else {
			slog.Warn("Seed parse error, fallback to random", "input", seedValStr, "error", err)
		}
	}

	if seedPtr == nil {
		n, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
		var finalSeed int64
		if err == nil {
			finalSeed = n.Int64()
		} else {
			// 万が一のフォールバック
			finalSeed = time.Now().UnixNano()
		}
		seedPtr = &finalSeed
		slog.Info("Explicitly generated seed for new task", "seed", *seedPtr)
	}

	task := domain.Task{
		JobID:      r.FormValue("job_id"),
		RequestURL: r.FormValue("url"),
		InputText:  r.FormValue("text"),
		ImageURL:   r.FormValue("image"),
		CreatedAt:  time.Now().UTC(),
		AIModels: domain.AIModels{
			TextModel:   strings.TrimSpace(r.FormValue("lyrics_model")),
			AudioModel:  strings.TrimSpace(r.FormValue("compose_model")),
			ComposeMode: strings.TrimSpace(r.FormValue("compose_mode")),
			Seed:        seedPtr,
		},
	}

	// JobID が空の場合は自動生成
	if task.JobID == "" {
		task.JobID = fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102150405"), uuid.New().String()[:8])
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
