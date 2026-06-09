package adapters

import (
	"fmt"
	"strings"

	"github.com/shouni/go-gemini-client/lyria"
)

type lyriaAudioPromptBuilder struct{}

// NewDefaultLyriaAudioPromptBuilder returns the public showcase audio prompt builder.
func NewDefaultLyriaAudioPromptBuilder() lyria.AudioPromptBuilder {
	return lyriaAudioPromptBuilder{}
}

// lyriaSectionDirections は、同じセクションを曲全体生成と単体生成でどう指示するかを保持します。
type lyriaSectionDirections struct {
	fullSong string
	section  string
}

// BuildFullSong は、MusicRecipe 全体を 1 回の Lyria 呼び出しで生成するためのプロンプトを組み立てます。
func (lyriaAudioPromptBuilder) BuildFullSong(recipe *lyria.MusicRecipe) string {
	var pb strings.Builder
	pb.WriteString("Task: Generate a full song from the provided music recipe.\n")
	pb.WriteString(buildLyriaSongContext(recipe))

	if len(recipe.Sections) > 0 {
		pb.WriteString("Song Structure:\n")
		for _, sec := range recipe.Sections {
			directions := buildLyriaSectionDirections(sec.Name)
			pb.WriteString(fmt.Sprintf("- [%s] (%d sec): %s %s\n", sec.Name, sec.Duration, directions.fullSong, sec.Prompt))
		}
		pb.WriteString("\n")
	}

	pb.WriteString("[Generation Guidelines]\n")
	pb.WriteString("- Follow the provided title, mood, tempo, instruments, lyrics, and section structure.\n")
	pb.WriteString("- Keep transitions natural between sections.\n")
	pb.WriteString("- Preserve the intended musical direction from each section prompt.\n")
	pb.WriteString("- Avoid unintended long pauses, abrupt endings, or silent gaps between sections.\n")
	pb.WriteString("- Ensure clear vocal performance and proper enunciation throughout the track.")

	return pb.String()
}

// BuildSection は、MusicRecipe のうち指定された 1 セクションだけを生成するためのプロンプトを組み立てます。
func (lyriaAudioPromptBuilder) BuildSection(recipe *lyria.MusicRecipe, sec lyria.MusicSection) string {
	var pb strings.Builder
	pb.WriteString("Task: Generate only the current song section.\n")
	pb.WriteString(fmt.Sprintf("Current Section: [%s]. Duration: %d seconds.\n", sec.Name, sec.Duration))
	pb.WriteString(buildLyriaSongContext(recipe))
	pb.WriteString(buildLyriaSectionDirections(sec.Name).section)

	pb.WriteString(fmt.Sprintf(
		"\n[Section Generation Guidelines]\n- Music Detail: %s",
		sec.Prompt,
	))

	return pb.String()
}

// buildLyriaSongContext は、全体生成とセクション生成で共有する曲全体の文脈を組み立てます。
func buildLyriaSongContext(recipe *lyria.MusicRecipe) string {
	var pb strings.Builder
	pb.WriteString(fmt.Sprintf("Title: '%s'.\n", recipe.Title))
	pb.WriteString(fmt.Sprintf("Style & Mood: %s\n", recipe.Mood))
	pb.WriteString(fmt.Sprintf("Tempo: %d BPM. Instruments: %s.\n\n", recipe.Tempo, strings.Join(recipe.Instruments, ", ")))

	if recipe.Lyrics != nil && recipe.Lyrics.Lyrics != "" {
		pb.WriteString("Lyrics:\n")
		pb.WriteString(recipe.Lyrics.Lyrics)
		pb.WriteString("\n\n")
	}

	return pb.String()
}

// buildLyriaSectionDirections は、セクション名に応じた汎用的な生成方針を返します。
func buildLyriaSectionDirections(sectionName string) lyriaSectionDirections {
	return lyriaSectionDirections{
		fullSong: fmt.Sprintf("Section Direction: Use this section as the [%s] part of the full arrangement.", sectionName),
		section:  fmt.Sprintf("Section Direction: Generate the [%s] section according to the recipe context. ", sectionName),
	}
}
