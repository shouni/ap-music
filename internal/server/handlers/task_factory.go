package handlers

import (
	"fmt"
	"math/rand/v2"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ap-music/internal/domain"
)

type taskFactory struct {
	now      func() time.Time
	newSeed  func() int64
	newJobID func(time.Time) string
}

func newTaskFactory() *taskFactory {
	return &taskFactory{
		now:     func() time.Time { return time.Now().UTC() },
		newSeed: rand.Int64,
		newJobID: func(now time.Time) string {
			return fmt.Sprintf("%s-%s", now.Format("20060102150405"), uuid.New().String()[:8])
		},
	}
}

func (f *taskFactory) Build(form url.Values) domain.Task {
	createdAt := f.now()

	task := domain.Task{
		JobID:      strings.TrimSpace(form.Get("job_id")),
		RequestURL: strings.TrimSpace(form.Get("url")),
		InputText:  strings.TrimSpace(form.Get("text")),
		ImageURL:   strings.TrimSpace(form.Get("image")),
		CreatedAt:  createdAt,
		AIModels: domain.AIModels{
			TextModel:   strings.TrimSpace(form.Get("lyrics_model")),
			AudioModel:  strings.TrimSpace(form.Get("compose_model")),
			ComposeMode: strings.TrimSpace(form.Get("compose_mode")),
			Seed:        parseSeed(form.Get("seed"), f.newSeed),
		},
	}

	if task.JobID == "" {
		task.JobID = f.newJobID(createdAt)
	}

	return task
}

func parseSeed(raw string, fallback func() int64) *int64 {
	seedText := strings.TrimSpace(raw)
	if seedText != "" {
		if val, err := strconv.ParseInt(seedText, 10, 64); err == nil {
			return &val
		}
	}

	seed := fallback()
	return &seed
}
