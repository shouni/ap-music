package handlers

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ap-music/internal/domain"
)

type taskFactory struct {
	now          func() time.Time
	newSeed      func() int64
	newJobID     func(time.Time) string
	allowedModes []string
}

// newTaskFactory は許可されたモードリストを受け取って初期化するように変更
func newTaskFactory(allowedModes []string) *taskFactory {
	return &taskFactory{
		now: func() time.Time { return time.Now().UTC() },
		newSeed: func() int64 {
			return int64(rand.Uint32() & math.MaxInt32)
		},
		newJobID: func(now time.Time) string {
			return fmt.Sprintf("%s-%s", now.Format("20060102150405"), uuid.New().String()[:8])
		},
		allowedModes: allowedModes,
	}
}

// Build はフォームデータからタスクを構築します。フォームデータのバリデーションとタスクの構築を実行します。
func (f *taskFactory) Build(form url.Values) domain.Task {
	createdAt := f.now()

	rawMode := strings.TrimSpace(form.Get("compose_mode"))
	validatedMode := ""
	if slices.Contains(f.allowedModes, rawMode) {
		validatedMode = rawMode
	}

	task := domain.Task{
		JobID:      strings.TrimSpace(form.Get("job_id")),
		RequestURL: strings.TrimSpace(form.Get("url")),
		InputText:  strings.TrimSpace(form.Get("text")),
		ImageURL:   strings.TrimSpace(form.Get("image")),
		CreatedAt:  createdAt,
		AIModels: domain.AIModels{
			TextModel:   strings.TrimSpace(form.Get("lyrics_model")),
			AudioModel:  strings.TrimSpace(form.Get("compose_model")),
			ComposeMode: validatedMode, // バリデーション済みの値をセット
			Seed:        parseSeed(form.Get("seed"), f.newSeed),
		},
	}

	if task.JobID == "" {
		task.JobID = f.newJobID(createdAt)
	}

	return task
}

// parseSeed 指定された生のシード文字列を解析してint64ポインタに変換します。解析に失敗した場合は、フォールバック関数を使用します。
// 解析された値がint32の最大値を超える場合は、int32の制限内に収まるように値をラップします。
func parseSeed(raw string, fallback func() int64) *int64 {
	seedText := strings.TrimSpace(raw)
	if seedText != "" {
		if val, err := strconv.ParseInt(seedText, 10, 64); err == nil {
			if val > math.MaxInt32 {
				val = val % (math.MaxInt32 + 1)
			}
			return &val
		}
	}

	return new(fallback())
}
