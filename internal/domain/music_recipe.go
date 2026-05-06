package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LyricsDraft は作詞フェーズの出力です。
type LyricsDraft struct {
	Title     string   `json:"title"`
	Theme     string   `json:"theme"`
	Hook      string   `json:"hook"`
	Lyrics    string   `json:"lyrics"`
	Keywords  []string `json:"keywords,omitempty"`
	Mood      string   `json:"mood,omitempty"`
	Narrative string   `json:"narrative,omitempty"`
}

// MusicRecipe は楽曲設計図です。
type MusicRecipe struct {
	Title       string         `json:"title"`
	Theme       string         `json:"theme"`
	Mood        string         `json:"mood"`
	Tempo       int            `json:"tempo"`
	Instruments []string       `json:"instruments"`
	Sections    []MusicSection `json:"sections"`
	Lyrics      *LyricsDraft   `json:"lyrics,omitempty"`
	AIModels
}

// DecodeMusicRecipeJSON parses user-submitted JSON into a MusicRecipe.
func DecodeMusicRecipeJSON(raw string) (*MusicRecipe, error) {
	recipeText := strings.TrimSpace(raw)
	if recipeText == "" {
		return nil, fmt.Errorf("music recipe json is required")
	}

	var recipe MusicRecipe
	if err := json.Unmarshal([]byte(recipeText), &recipe); err != nil {
		return nil, fmt.Errorf("invalid music recipe json: %w", err)
	}
	if err := recipe.ValidateForGeneration(); err != nil {
		return nil, err
	}

	return &recipe, nil
}

// ValidateForGeneration checks that the recipe has enough musical direction to
// send to the audio generator.
func (r MusicRecipe) ValidateForGeneration() error {
	if strings.TrimSpace(r.Title) == "" &&
		strings.TrimSpace(r.Theme) == "" &&
		len(r.Instruments) == 0 &&
		len(r.Sections) == 0 &&
		r.Lyrics == nil {
		return fmt.Errorf("music recipe must include title, theme, instruments, sections, or lyrics")
	}

	return nil
}

// MusicSection は曲内セクションです。
type MusicSection struct {
	Name     string `json:"name"`
	Duration int    `json:"duration_seconds"`
	Prompt   string `json:"prompt"`
}

// MusicHistory は一覧画面の表示
type MusicHistory struct {
	JobID       string `json:"job_id"`
	Title       string `json:"title"`
	Mood        string `json:"mood,omitempty"`
	Tempo       int    `json:"tempo,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	ComposeMode string `json:"compose_mode,omitempty"`
	Seed        string `json:"seed,omitempty"`
}
