package adapters

import (
	"context"
	"testing"

	"github.com/shouni/go-gemini-client/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/time/rate"
	"google.golang.org/genai"

	"ap-music/internal/domain"
)

// --- Mocks ---

type MockGeminiClient struct {
	mock.Mock
}

func (m *MockGeminiClient) GenerateContent(ctx context.Context, model, prompt string) (*gemini.Response, error) {
	args := m.Called(ctx, model, prompt)
	if res, ok := args.Get(0).(*gemini.Response); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

// *gemini.GenerateOptions ではなく gemini.GenerateOptions (値渡し) に修正
func (m *MockGeminiClient) GenerateWithParts(ctx context.Context, modelName string, parts []*genai.Part, opts gemini.GenerateOptions) (*gemini.Response, error) {
	args := m.Called(ctx, modelName, parts, opts)
	if res, ok := args.Get(0).(*gemini.Response); ok {
		return res, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockGeminiClient) IsVertexAI() bool {
	args := m.Called()
	return args.Bool(0)
}

type MockPromptGen struct {
	mock.Mock
}

func (m *MockPromptGen) GenerateLyrics(input string) (string, error) {
	args := m.Called(input)
	return args.String(0), args.Error(1)
}

func (m *MockPromptGen) GenerateRecipe(mode string, lyrics *domain.LyricsDraft) (string, error) {
	args := m.Called(mode, lyrics)
	return args.String(0), args.Error(1)
}

// --- Tests ---

func TestLyriaAdapter_Run(t *testing.T) {
	ctx := context.Background()
	mAI := new(MockGeminiClient)
	mPrompt := new(MockPromptGen)

	seedVal := int64(42)

	// テスト対象のアダプターを構築
	// 内部で呼び出される sub-struct (lyriaLyricistなど) にモックを注入
	adapter := &LyriaAdapter{
		lyricist: &lyriaLyricist{
			aiClient:     mAI,
			promptGen:    mPrompt,
			defaultModel: "gemini-flash",
		},
		composer: &lyriaComposer{
			aiClient:     mAI,
			promptGen:    mPrompt,
			defaultModel: "gemini-flash",
		},
		audio: &lyriaAudioGenerator{
			aiClient:          mAI,
			defaultLyriaModel: "lyria-3",
			limiter:           rate.NewLimiter(rate.Inf, 0), // テストなので待ち時間なし
			promptBuilder:     lyriaAudioPromptBuilder{},
		},
	}

	task := domain.Task{
		JobID: "job-123",
		AIModels: domain.AIModels{
			TextModel:   "custom-text-model",
			AudioModel:  "lyria-custom-v1",
			ComposeMode: "jazz",
			Seed:        &seedVal,
		},
	}
	contextText := "雨のアムステルダム"

	// 期待される中間データ
	expectedLyrics := &domain.LyricsDraft{
		Title:  "Rainy Amsterdam",
		Theme:  "Neon reflection on canals",
		Lyrics: "Canals reflect the neon lights...",
	}

	// AIからのレスポンス想定
	// マークダウンタグが含まれていても cleanJSONResponse で処理されることを想定
	lyricsJSON := `{"title": "Rainy Amsterdam", "theme": "Neon reflection on canals", "lyrics": "Canals reflect the neon lights..."}`
	recipeJSON := `{"title": "Rainy Amsterdam", "tempo": 85, "mood": "melancholic"}`

	fakeWav := []byte("RIFF....WAVEfmt....data") // 簡略化したWAVヘッダ

	// 1. 作詞プロンプト生成 -> AI実行
	mPrompt.On("GenerateLyrics", contextText).Return("prompt-lyrics-text", nil)
	mAI.On("GenerateContent", mock.Anything, "custom-text-model", "prompt-lyrics-text").Return(&gemini.Response{
		Text: "```json\n" + lyricsJSON + "\n```",
	}, nil)

	// 2. 作曲レシピ生成 -> AI実行
	mPrompt.On("GenerateRecipe", "jazz", expectedLyrics).Return("prompt-recipe-text", nil)
	mAI.On("GenerateContent", mock.Anything, "custom-text-model", "prompt-recipe-text").Return(&gemini.Response{
		Text: recipeJSON,
	}, nil)

	// 3. 音声生成実行
	// GenerateWithParts の第4引数は mock.Anything または具体的条件
	mAI.On("GenerateWithParts", mock.Anything, "lyria-custom-v1", mock.Anything, mock.Anything).Return(&gemini.Response{
		Audios: [][]byte{fakeWav},
	}, nil)

	// 実行
	recipe, wav, err := adapter.Run(ctx, task, contextText)

	// 検証
	assert.NoError(t, err)
	assert.NotNil(t, recipe)
	assert.Equal(t, "Rainy Amsterdam", recipe.Title)
	assert.Equal(t, 85, recipe.Tempo)
	assert.Equal(t, fakeWav, wav)

	// Seed値の伝搬確認
	if assert.NotNil(t, recipe.AIModels.Seed) {
		assert.Equal(t, int64(42), *recipe.AIModels.Seed)
	}

	mPrompt.AssertExpectations(t)
	mAI.AssertExpectations(t)
}

func TestLyriaAdapter_Compose(t *testing.T) {
	ctx := context.Background()
	mAI := new(MockGeminiClient)
	mPrompt := new(MockPromptGen)

	adapter := &LyriaAdapter{
		composer: &lyriaComposer{
			aiClient:     mAI,
			promptGen:    mPrompt,
			defaultModel: "gemini-flash",
		},
	}

	lyrics := &domain.LyricsDraft{Title: "Lofi Beats", Lyrics: "Chill vibes only"}
	mode := "lofi"
	expectedPrompt := "Build me a lofi recipe"
	rawJSON := `{"title": "Lofi Chill", "tempo": 70, "mood": "relaxed"}`

	mPrompt.On("GenerateRecipe", mode, lyrics).Return(expectedPrompt, nil)
	mAI.On("GenerateContent", mock.Anything, "gemini-flash", expectedPrompt).Return(&gemini.Response{
		Text: rawJSON,
	}, nil)

	// 実行（モデル指定なしの場合は defaultModel が使われる）
	recipe, err := adapter.Compose(ctx, lyrics, "", mode)

	assert.NoError(t, err)
	assert.NotNil(t, recipe)
	assert.Equal(t, 70, recipe.Tempo)
	assert.Equal(t, "Lofi Chill", recipe.Title)

	mPrompt.AssertExpectations(t)
	mAI.AssertExpectations(t)
}
