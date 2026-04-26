package handlers

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ap-music/internal/domain"
)

func TestTaskFactoryBuildUsesProvidedValues(t *testing.T) {
	now := time.Date(2026, 4, 26, 10, 30, 0, 0, time.UTC)
	factory := &taskFactory{
		now:      func() time.Time { return now },
		newSeed:  func() int64 { return 99 },
		newJobID: func(time.Time) string { return "generated-job" },
	}

	task := factory.Build(url.Values{
		"job_id":        {"job-123"},
		"url":           {" https://example.com "},
		"text":          {" hello "},
		"image":         {" https://example.com/image.png "},
		"lyrics_model":  {" gemini-text "},
		"compose_model": {" lyria-audio "},
		"compose_mode":  {" rave "},
		"seed":          {"42"},
	})

	expected := domain.Task{
		JobID:      "job-123",
		RequestURL: "https://example.com",
		InputText:  "hello",
		ImageURL:   "https://example.com/image.png",
		CreatedAt:  now,
		AIModels: domain.AIModels{
			TextModel:   "gemini-text",
			AudioModel:  "lyria-audio",
			ComposeMode: "rave",
			Seed:        ptrInt64(42),
		},
	}

	assert.Equal(t, expected, task)
}

func TestTaskFactoryBuildGeneratesFallbacks(t *testing.T) {
	now := time.Date(2026, 4, 26, 10, 30, 0, 0, time.UTC)
	factory := &taskFactory{
		now:      func() time.Time { return now },
		newSeed:  func() int64 { return 1234 },
		newJobID: func(time.Time) string { return "generated-job" },
	}

	task := factory.Build(url.Values{
		"seed": {"not-a-number"},
	})

	if assert.NotNil(t, task.Seed) {
		assert.Equal(t, int64(1234), *task.Seed)
	}
	assert.Equal(t, "generated-job", task.JobID)
	assert.Equal(t, now, task.CreatedAt)
}

func ptrInt64(v int64) *int64 {
	return &v
}
