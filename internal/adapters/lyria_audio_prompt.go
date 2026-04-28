package adapters

import (
	"fmt"
	"strings"

	"ap-music/internal/domain"
)

type lyriaAudioPromptBuilder struct{}

// lyriaSectionDirections は、同じセクションを曲全体生成と単体生成でどう指示するかを保持します。
type lyriaSectionDirections struct {
	fullSong string
	section  string
}

// BuildFullSong は、MusicRecipe 全体を 1 回の Lyria 呼び出しで生成するためのプロンプトを組み立てます。
func (lyriaAudioPromptBuilder) BuildFullSong(recipe *domain.MusicRecipe) string {
	var pb strings.Builder
	pb.WriteString("Task: Generate a full high-fidelity song.\n")
	pb.WriteString(buildLyriaSongContext(recipe))

	if len(recipe.Sections) > 0 {
		pb.WriteString("Detailed Song Structure & Multi-Stage Vocal Directions:\n")
		for _, sec := range recipe.Sections {
			directions := buildLyriaSectionDirections(sec)
			pb.WriteString(fmt.Sprintf("- [%s] (%d sec): %s %s\n", sec.Name, sec.Duration, directions.fullSong, sec.Prompt))
		}
		pb.WriteString("\n")
	}

	pb.WriteString("[Final Executive Constraints]\n")
	pb.WriteString("- Total Duration: Exactly 180 seconds. Do not end early.\n")
	pb.WriteString("- Seamless Flow: Ensure each long section evolves naturally into the next without any energy drops.\n")
	pb.WriteString("- Zero Silence: Maintain a continuous, high-fidelity sonic wall. No gaps or unintentional pauses.\n")
	pb.WriteString("- Vocal Purity: Clear, passionate Japanese vocals throughout. Absolute priority on lyrical clarity.")

	return pb.String()
}

// BuildSection は、MusicRecipe のうち指定された 1 セクションだけを生成するためのプロンプトを組み立てます。
func (lyriaAudioPromptBuilder) BuildSection(recipe *domain.MusicRecipe, sec domain.MusicSection) string {
	var pb strings.Builder
	pb.WriteString("Task: Generate only the current song section.\n")
	pb.WriteString(fmt.Sprintf("Current Section: [%s]. Duration: %d seconds.\n", sec.Name, sec.Duration))
	pb.WriteString(buildLyriaSongContext(recipe))
	pb.WriteString(buildLyriaSectionDirections(sec).section)
	pb.WriteString("Clear Japanese vocals with passionate enunciation. No silence.")

	pb.WriteString(fmt.Sprintf(
		"\n[Section Audio Generation Constraints]\n- Music Detail: %s",
		sec.Prompt,
	))

	return pb.String()
}

// buildLyriaSongContext は、全体生成とセクション生成で共有する曲全体の文脈を組み立てます。
func buildLyriaSongContext(recipe *domain.MusicRecipe) string {
	var pb strings.Builder
	pb.WriteString(fmt.Sprintf("Title: '%s'.\n", recipe.Title))
	pb.WriteString(fmt.Sprintf("Style & Mood: %s\n", recipe.Mood))
	pb.WriteString(fmt.Sprintf("Tempo: %d BPM. Instruments: %s.\n\n", recipe.Tempo, strings.Join(recipe.Instruments, ", ")))

	if recipe.Lyrics != nil && recipe.Lyrics.Lyrics != "" {
		pb.WriteString("Lyrics (Perform with clear Japanese vocals and passionate enunciation):\n")
		pb.WriteString(recipe.Lyrics.Lyrics)
		pb.WriteString("\n\n")
	}

	return pb.String()
}

// buildLyriaSectionDirections は、セクション名に応じたボーカル方針を返します。
func buildLyriaSectionDirections(sec domain.MusicSection) lyriaSectionDirections {
	switch sec.Name {
	case "Verse":
		return lyriaSectionDirections{
			fullSong: "Vocal Strategy: Focus on singing the [Verse] section. Start narratively and build tension for the next phase.",
			section:  "Vocal Direction: Focus on singing the [Verse] section. Build tension for the next phase. ",
		}
	case "Chorus":
		return lyriaSectionDirections{
			fullSong: "Vocal Strategy: Max energy! Perform the [Chorus] and Hook with high-octane passion. Keep the heat throughout this long climax.",
			section:  "Vocal Direction: Max energy! Perform the [Chorus] and Hook with high-octane passion. ",
		}
	case "Outro":
		return lyriaSectionDirections{
			fullSong: "Vocal Strategy: Emotional digital fade-out for the [Outro]. Let the Japanese vocals dissolve into a cybernetic echo.",
			section:  "Vocal Direction: Emotional digital fade-out for the [Outro]. Leave a cybernetic echo. ",
		}
	default:
		return lyriaSectionDirections{
			fullSong: fmt.Sprintf("Vocal Strategy: Adapt your energy to sustain this %d-second section with consistent Japanese vocal quality.", sec.Duration),
			section:  fmt.Sprintf("Vocal Direction: Perform the [%s] section with clear Japanese vocals and appropriate energy for the track. ", sec.Name),
		}
	}
}
