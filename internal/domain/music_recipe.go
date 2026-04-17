package domain

// MusicRecipe は楽曲設計図です。
type MusicRecipe struct {
	Title       string            `json:"title"`
	Theme       string            `json:"theme"`
	Mood        string            `json:"mood"`
	Tempo       int               `json:"tempo"`
	Instruments []string          `json:"instruments"`
	Sections    []MusicSection    `json:"sections"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// MusicSection は曲内セクションです。
type MusicSection struct {
	Name     string `json:"name"`
	Duration int    `json:"duration_seconds"`
	Prompt   string `json:"prompt"`
}
