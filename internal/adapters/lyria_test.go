package adapters

import (
	"context"
	"testing"

	"github.com/shouni/go-gemini-client/gemini"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/genai"

	"ap-music/internal/domain"
)

// MockGeminiClient は gemini.Generator インターフェースのモックです。
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

// MockPromptGen は domain.PromptGenerator インターフェースのモックです。
type MockPromptGen struct {
	mock.Mock
}

func (m *MockPromptGen) GenerateLyrics(input string) (string, error) {
	args := m.Called(input)
	return args.String(0), args.Error(1)
}

func (m *MockPromptGen) GenerateRecipe(mode string, lyrics domain.LyricsDraft) (string, error) {
	args := m.Called(mode, lyrics)
	return args.String(0), args.Error(1)
}

// TestLyriaAdapter_Compose は MusicRecipe 生成のテストです。
func TestLyriaAdapter_Compose(t *testing.T) {
	ctx := context.Background()
	mAI := new(MockGeminiClient)
	mPrompt := new(MockPromptGen)

	defaultModelName := "gemini-flash"
	adapter := &LyriaAdapter{
		aiClient:     mAI,
		promptGen:    mPrompt,
		defaultModel: defaultModelName,
	}

	lyrics := domain.LyricsDraft{Title: "Digital Dream", Lyrics: "0101..."}
	mode := "rave"
	targetModel := "lyria-3-experimental" // 明示的に指定する場合
	expectedPrompt := "Build me a rave recipe"

	// 期待されるレスポンス JSON
	rawJSON := `{"bpm": 128, "key": "Am", "mood": "energetic"}`

	// Mock の挙動設定
	// PromptGenerator の引数順序は GenerateRecipe(mode, lyrics) と仮定
	mPrompt.On("GenerateRecipe", mode, lyrics).Return(expectedPrompt, nil)

	// aiClient.GenerateContent の呼び出しを期待
	mAI.On("GenerateContent", ctx, targetModel, expectedPrompt).Return(&gemini.Response{
		Text:        rawJSON,
		RawResponse: &genai.GenerateContentResponse{},
	}, nil)

	// 実行: 引数の順番を (ctx, lyrics, model, mode) に合わせる
	recipe, err := adapter.Compose(ctx, lyrics, targetModel, mode)

	// 検証
	assert.NoError(t, err)
	assert.NotNil(t, recipe)

	mPrompt.AssertExpectations(t)
	mAI.AssertExpectations(t)
}
