package assets

import (
	"embed"

	"github.com/shouni/go-prompt-kit/resource"
)

const (
	promptDir = "prompts"

	lyricsPrefix  = "lyrics_"
	composePrefix = "compose_"

	// ModeLyrics represents a constant string identifier for "lyrics".
	ModeLyrics = "default"
	// ModeCompose represents a constant string identifier for "compose".
	ModeCompose = "default"
)

var (
	// lyricsFiles はプロンプトテンプレートです。
	//go:embed prompts/lyrics_*.md
	lyricsFiles embed.FS

	// composeFiles はプロンプトテンプレートです。
	//go:embed prompts/compose_*.md
	composeFiles embed.FS

	// Templates は、すべてのHTMLテンプレートを保持します。
	//go:embed templates/*.html
	Templates embed.FS
)

// LoadLyricsFiles は埋め込まれたプロンプトファイルを読み込みます。
func LoadLyricsFiles() (map[string]string, error) {
	return resource.Load(lyricsFiles, promptDir, lyricsPrefix)
}

// LoadComposeFiles は埋め込まれたプロンプトファイルを読み込みます。
func LoadComposeFiles() (map[string]string, error) {
	return resource.Load(composeFiles, promptDir, composePrefix)
}
