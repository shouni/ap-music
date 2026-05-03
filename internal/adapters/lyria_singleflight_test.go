package adapters

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shouni/go-gemini-client/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
	"google.golang.org/genai"

	"ap-music/internal/domain"
)

// staticPromptGen はテスト用の固定プロンプト生成器です。
type staticPromptGen struct {
	lyricsPrompt string
	recipePrompt string
}

// GenerateLyrics はインターフェース変更に合わせて (string, string) を受け取るように修正
func (g staticPromptGen) GenerateLyrics(mode string, content string) (string, error) {
	return g.lyricsPrompt, nil
}

// GenerateRecipe はインターフェース変更に合わせて (string, *domain.LyricsDraft) を受け取るように修正
func (g staticPromptGen) GenerateRecipe(mode string, lyrics *domain.LyricsDraft) (string, error) {
	return g.recipePrompt, nil
}

type blockingGeminiClient struct {
	contentCalls atomic.Int32
	partsCalls   atomic.Int32

	contentStarted sync.Once
	partsStarted   sync.Once

	contentStartedCh chan struct{}
	partsStartedCh   chan struct{}
	releaseContentCh chan struct{}
	releasePartsCh   chan struct{}

	contentResp *gemini.Response
	partsResp   *gemini.Response
}

func newBlockingGeminiClient() *blockingGeminiClient {
	return &blockingGeminiClient{
		contentStartedCh: make(chan struct{}),
		partsStartedCh:   make(chan struct{}),
		releaseContentCh: make(chan struct{}),
		releasePartsCh:   make(chan struct{}),
	}
}

func (c *blockingGeminiClient) GenerateContent(context.Context, string, string) (*gemini.Response, error) {
	c.contentCalls.Add(1)
	c.contentStarted.Do(func() { close(c.contentStartedCh) })
	<-c.releaseContentCh
	return c.contentResp, nil
}

func (c *blockingGeminiClient) GenerateWithParts(context.Context, string, []*genai.Part, gemini.GenerateOptions) (*gemini.Response, error) {
	c.partsCalls.Add(1)
	c.partsStarted.Do(func() { close(c.partsStartedCh) })
	<-c.releasePartsCh
	return c.partsResp, nil
}

func (c *blockingGeminiClient) IsVertexAI() bool {
	return false
}

func TestLyriaLyricistSingleflightDeduplicatesConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	client := newBlockingGeminiClient()
	client.contentResp = &gemini.Response{
		Text: `{"title":"Song","theme":"Theme","lyrics":"Words","keywords":["one"]}`,
	}

	lyricist := &lyriaLyricist{
		aiClient:     client,
		promptGen:    staticPromptGen{lyricsPrompt: "lyrics prompt"},
		defaultModel: "gemini-flash",
	}

	const callers = 5
	results := make([]*domain.LyricsDraft, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := range callers {
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = lyricist.GenerateLyrics(ctx, "same input", "default", "gemini-flash")
		}(i)
	}

	require.Eventually(t, func() bool {
		select {
		case <-client.contentStartedCh:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	close(client.releaseContentCh)
	wg.Wait()

	require.Equal(t, int32(1), client.contentCalls.Load())
	for _, err := range errs {
		require.NoError(t, err)
	}

	require.NotSame(t, results[0], results[1])

	results[0].Keywords[0] = "changed"
	assert.Equal(t, "one", results[1].Keywords[0])
}

func TestLyriaAudioGeneratorSingleflightDeduplicatesConcurrentCalls(t *testing.T) {
	ctx := context.Background()
	client := newBlockingGeminiClient()
	client.partsResp = &gemini.Response{Audios: [][]byte{{1, 2, 3}}}
	seed := int64(7)

	generator := &lyriaAudioGenerator{
		aiClient:          client,
		defaultLyriaModel: "lyria-3",
		limiter:           rate.NewLimiter(rate.Inf, 0),
		promptBuilder:     lyriaAudioPromptBuilder{},
	}
	recipe := &domain.MusicRecipe{
		Title:       "Song",
		Mood:        "Bright",
		Tempo:       140,
		Instruments: []string{"synth"},
		Sections: []domain.MusicSection{
			{Name: "Verse", Duration: 30, Prompt: "pulse"},
		},
		AIModels: domain.AIModels{Seed: &seed},
	}

	const callers = 5
	results := make([][]byte, callers)
	errs := make([]error, callers)
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := range callers {
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = generator.GenerateAudio(ctx, recipe)
		}(i)
	}

	require.Eventually(t, func() bool {
		select {
		case <-client.partsStartedCh:
			return true
		default:
			return false
		}
	}, time.Second, time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	close(client.releasePartsCh)
	wg.Wait()

	require.Equal(t, int32(1), client.partsCalls.Load())
	for _, err := range errs {
		require.NoError(t, err)
	}

	results[0][0] = 9
	assert.Equal(t, byte(1), results[1][0])
}
