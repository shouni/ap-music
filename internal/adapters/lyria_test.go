package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/shouni/go-gemini-client/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	adapter := &LyriaAdapter{
		aiClient:          mAI,
		promptGen:         mPrompt,
		defaultModel:      "gemini-2.0-flash",
		defaultLyriaModel: "lyria-3",
	}

	task := domain.Task{
		JobID:     "job-123",
		CreatedAt: time.Now(),
		AIModels: domain.AIModels{
			TextModel:   "custom-text-model",
			AudioModel:  "lyria-custom-v1",
			ComposeMode: "jazz",
		},
	}
	contextText := "雨のアムステルダム"

	// domain.LyricsDraft の新しい定義に合わせて初期化
	expectedLyrics := &domain.LyricsDraft{
		Title:  "Rainy Amsterdam",
		Theme:  "Neon reflection on canals",
		Lyrics: "Canals reflect the neon lights...",
	}

	// JSON文字列も構造体に合わせて調整
	lyricsJSON := `{"title": "Rainy Amsterdam", "theme": "Neon reflection on canals", "lyrics": "Canals reflect the neon lights..."}`
	recipeJSON := `{"title": "Rainy Amsterdam", "tempo": 85, "mood": "melancholic", "sections": [{"name": "Main", "duration_seconds": 30, "prompt": "jazz piano"}]}`
	fakeWav := []byte{0x52, 0x49, 0x46, 0x46, 0x00}

	// Mock 設定
	mPrompt.On("GenerateLyrics", contextText).Return("prompt-lyrics-text", nil)
	mAI.On("GenerateContent", ctx, "custom-text-model", "prompt-lyrics-text").Return(&gemini.Response{
		Text: "```json\n" + lyricsJSON + "\n```",
	}, nil)

	mPrompt.On("GenerateRecipe", "jazz", expectedLyrics).Return("prompt-recipe-text", nil)
	mAI.On("GenerateContent", ctx, "custom-text-model", "prompt-recipe-text").Return(&gemini.Response{
		Text: recipeJSON,
	}, nil)

	mAI.On("GenerateWithParts", ctx, "lyria-custom-v1", mock.Anything, mock.Anything).Return(&gemini.Response{
		Audios: [][]byte{fakeWav},
	}, nil)

	// 実行
	recipe, wav, err := adapter.Run(ctx, task, contextText)

	// 検証
	assert.NoError(t, err)
	assert.NotNil(t, recipe)
	assert.Equal(t, fakeWav, wav)
	assert.Equal(t, "Rainy Amsterdam", recipe.Title)
	assert.Equal(t, 85, recipe.Tempo)
	assert.Equal(t, expectedLyrics, recipe.Lyrics)
	assert.Equal(t, task.AIModels, recipe.AIModels)

	mPrompt.AssertExpectations(t)
	mAI.AssertExpectations(t)
}

func TestLyriaAdapter_Compose(t *testing.T) {
	ctx := context.Background()
	mAI := new(MockGeminiClient)
	mPrompt := new(MockPromptGen)

	adapter := &LyriaAdapter{
		aiClient:     mAI,
		promptGen:    mPrompt,
		defaultModel: "gemini-flash",
	}

	lyrics := &domain.LyricsDraft{Title: "Lofi Beats", Lyrics: "Chill vibes only"}
	mode := "lofi"
	expectedPrompt := "Build me a lofi recipe"
	rawJSON := `{"title": "Lofi Chill", "tempo": 70, "mood": "relaxed"}`

	mPrompt.On("GenerateRecipe", mode, lyrics).Return(expectedPrompt, nil)
	mAI.On("GenerateContent", ctx, "gemini-flash", expectedPrompt).Return(&gemini.Response{
		Text: rawJSON,
	}, nil)

	// 実行
	recipe, err := adapter.Compose(ctx, lyrics, "", mode)

	assert.NoError(t, err)
	assert.NotNil(t, recipe)
	assert.Equal(t, 70, recipe.Tempo)
	assert.Equal(t, lyrics, recipe.Lyrics)

	mPrompt.AssertExpectations(t)
	mAI.AssertExpectations(t)
}
