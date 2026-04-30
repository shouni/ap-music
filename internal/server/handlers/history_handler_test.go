package handlers

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"ap-music/internal/domain"
)

type stubMusicRepository struct {
	recipe *domain.MusicRecipe
}

func (r *stubMusicRepository) ListHistory(context.Context, string) ([]domain.MusicHistory, error) {
	return nil, nil
}

func (r *stubMusicRepository) GetRecipe(context.Context, string) (*domain.MusicRecipe, error) {
	return r.recipe, nil
}

func TestServeDetailsRendersRecipeJSONAsUTF8(t *testing.T) {
	t.Parallel()

	recipe := &domain.MusicRecipe{
		Title: "朝焼け<テスト>&構成",
		Theme: "日本語ボーカル",
		Mood:  "透明感",
		Sections: []domain.MusicSection{
			{Name: "サビ", Duration: 30, Prompt: "高揚感のある歌声"},
		},
		Lyrics: &domain.LyricsDraft{
			Lyrics: "きみの声が\n朝に溶ける",
		},
	}
	h, err := NewHandler(nil, nil, nil, &stubMusicRepository{recipe: recipe})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/web/history/job-utf8", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("jobID", "job-utf8")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rec := httptest.NewRecorder()

	h.ServeDetails(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "朝焼け") || !strings.Contains(body, "高揚感のある歌声") {
		t.Fatalf("response body does not contain expected Japanese text: %s", body)
	}
	if strings.Contains(body, `\u003c`) || strings.Contains(body, `\u003e`) || strings.Contains(body, `\u0026`) {
		t.Fatalf("display JSON contains avoidable HTML unicode escapes: %s", body)
	}

	jsonText := extractCodeText(t, body)
	var rendered domain.MusicRecipe
	if err := json.Unmarshal([]byte(jsonText), &rendered); err != nil {
		t.Fatalf("rendered JSON is invalid: %v\n%s", err, jsonText)
	}
	if rendered.Title != recipe.Title {
		t.Fatalf("rendered title = %q, want %q", rendered.Title, recipe.Title)
	}
}

func extractCodeText(t *testing.T, body string) string {
	t.Helper()

	preStart := strings.Index(body, `<pre id="json-raw"`)
	if preStart < 0 {
		t.Fatalf("json pre block not found")
	}
	codeStart := strings.Index(body[preStart:], "<code>")
	if codeStart < 0 {
		t.Fatalf("json code block not found")
	}
	codeStart += preStart + len("<code>")
	codeEnd := strings.Index(body[codeStart:], "</code>")
	if codeEnd < 0 {
		t.Fatalf("json code block end not found")
	}
	return html.UnescapeString(body[codeStart : codeStart+codeEnd])
}
