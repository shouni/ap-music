package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/shouni/gcp-kit/tasks"

	"ap-music/assets"
	"ap-music/internal/app"
	"ap-music/internal/config"
	"ap-music/internal/domain"
)

const titleSuffix = " - AP Music"

type Handler struct {
	cfg           *config.Config
	templateCache map[string]*template.Template
	taskEnqueuer  *tasks.Enqueuer[domain.Task]
	remoteIO      *app.RemoteIO
	composeModes  []string
}

// Home はトップ画面を表示します。
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := struct {
		ComposeModes []string
	}{
		ComposeModes: h.composeModes,
	}

	h.render(w, http.StatusOK, "compose_form.html", "Compose", data)
}

// NewHandler は指定された構成に基づいて新しいハンドラーを初期化します。
// テンプレートをコンパイルし、レイアウトファイルが存在することを確認します。
func NewHandler(
	cfg *config.Config,
	taskEnqueuer *tasks.Enqueuer[domain.Task],
	remoteIO *app.RemoteIO,
) (*Handler, error) {
	cache := make(map[string]*template.Template)

	// 共通関数
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}

	// assets.Templates 内の "templates" ディレクトリを走査
	entries, err := fs.ReadDir(assets.Templates, "templates")
	if err != nil {
		return nil, fmt.Errorf("テンプレートディレクトリの読み込み失敗: %w", err)
	}

	// レイアウトファイルのパス定義
	layoutPath := "templates/layout.html"

	// 後続の ParseFS でのエラー混同を防ぎ、原因を特定しやすくします
	if _, err := fs.Stat(assets.Templates, layoutPath); err != nil {
		return nil, fmt.Errorf("レイアウトテンプレートが見つかりません: %s", layoutPath)
	}

	for _, entry := range entries {
		// ディレクトリ、または既に存在確認済みの layout.html 自体はスキップ
		if entry.IsDir() || entry.Name() == "layout.html" {
			continue
		}

		pageName := entry.Name()
		pagePath := "templates/" + pageName

		// ParseFS を使い、埋め込まれたファイルからパース
		// レイアウトと各ページを結合して一つのテンプレートセットとしてキャッシュします
		tmpl, err := template.New(pageName).
			Funcs(funcMap).
			ParseFS(assets.Templates, layoutPath, pagePath)

		if err != nil {
			return nil, fmt.Errorf("テンプレート %s の解析失敗: %w", pageName, err)
		}
		cache[pageName] = tmpl
	}

	// プロンプトファイル名からモードリストを動的に読み込む
	composePrompts, err := assets.LoadComposeFiles()
	if err != nil {
		return nil, fmt.Errorf("composeプロンプトの読み込み失敗: %w", err)
	}

	modes := make([]string, 0, len(composePrompts))
	for k := range composePrompts {
		modes = append(modes, k)
	}

	return &Handler{
		cfg:           cfg,
		templateCache: cache,
		taskEnqueuer:  taskEnqueuer,
		remoteIO:      remoteIO,
		composeModes:  modes,
	}, nil
}

// render は HTML テンプレートをレンダリングし、レスポンスを書き込みます。
func (h *Handler) render(w http.ResponseWriter, status int, pageName string, title string, data any) {
	tmpl, ok := h.templateCache[pageName]
	if !ok {
		slog.Error("キャッシュ内にテンプレートが見つかりません", "page", pageName)
		http.Error(w, "システムエラーが発生しました（テンプレート未定義）", http.StatusInternalServerError)
		return
	}

	renderData := struct {
		Title string
		Data  any
	}{
		Title: title + titleSuffix,
		Data:  data,
	}

	var buf bytes.Buffer
	// レイアウトファイルをベースに実行します
	if err := tmpl.ExecuteTemplate(&buf, "layout.html", renderData); err != nil {
		slog.Error("テンプレートのレンダリングに失敗しました", "page", pageName, "error", err)
		http.Error(w, "画面の表示中にエラーが発生しました", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Error("レスポンスの書き込みに失敗しました", "error", err)
	}
}
