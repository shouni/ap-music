package adapters

import (
	"strings"
	"testing"

	"ap-music/internal/domain"
)

func TestSlackContentIncludesCommand(t *testing.T) {
	t.Parallel()

	adapter := &SlackAdapter{serviceURL: "https://example.com"}
	content := adapter.buildSlackContent(&domain.PublishResult{
		JobID:      "job-1",
		StorageURI: "gs://bucket/job-1.wav",
	}, domain.NotificationRequest{
		Command: string(domain.TaskCommandGenerateFromRecipe),
	})

	if !strings.Contains(content, "*Command:* `generate_from_recipe`") {
		t.Fatalf("expected command in slack content, got %q", content)
	}
}

func TestSlackMetadataIncludesCommand(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	writeSlackRequestMetadata(&sb, domain.NotificationRequest{
		Command: string(domain.TaskCommandCompose),
		Title:   "Midnight Recipe",
		Mode:    "rave",
	})

	got := sb.String()
	if !strings.Contains(got, "*Command:* `compose`") {
		t.Fatalf("expected command in slack metadata, got %q", got)
	}
	if !strings.Contains(got, "*Title:* Midnight Recipe") {
		t.Fatalf("expected title in slack metadata, got %q", got)
	}
	if !strings.Contains(got, "*Mode:* `rave`") {
		t.Fatalf("expected mode in slack metadata, got %q", got)
	}
}
