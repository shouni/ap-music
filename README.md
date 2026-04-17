# 🎼 AP Music

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/ap-music)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/ap-music)](https://github.com/shouni/ap-music/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - 音楽生成のWebオーケストレーター

**AP Music** は、音楽生成コア機能（`Lyria 3` + `Music Recipe`）を  
**Cloud Run** および **Google Cloud Tasks** で Web アプリケーション化し、非同期にオーケストレーションするためのプロジェクトです。

Web フォーム経由で入力された URL / テキスト / 画像をジョブとしてキュー投入し、ワーカーで楽曲生成を実行します。完了時には **Slack 通知** と **署名付き URL** を発行し、生成結果を即座に共有できます。

---

## 🎨 ワークフロー (Workflows)

用途に応じて、Web UI から以下の機能を使い分けます。

| 画面 (Command) | 役割 | 主な入力 / 出力 |
| --- | --- | --- |
| **Compose** | URL / 文章 / 画像から楽曲設計図（Music Recipe）を生成 | URL・Text・Image / JSON (Recipe) |
| **Generate** | Recipe から音楽を生成 | Recipe / MP3 |
| **Publish** | 生成結果の保存と共有リンク発行 | MP3 / Signed URL |
| **Notify** | 実行完了を通知 | Job Result / Slack Message |

### 💻 実行フロー (Workflow)

1. **Request**: ユーザーが Web フォームから入力を送信。
2. **Enqueue**: `CloudTasksAdapter` がジョブを非同期投入。
3. **Worker**: Worker Handler がタスクを受信し `MusicPipeline` を起動。
4. **Pipeline**:
    - **Phase 1: Collect**: `go-web-reader` で入力コンテキスト収集。
    - **Phase 2: Compose**: LLM で `MusicRecipe` 生成。
    - **Phase 3: Generate**: `Lyria 3` で MP3 生成。
    - **Phase 4: Publish/Notify**: GCS/Local 保存、Signed URL 発行、Slack 通知。

---

## 🏗 アーキテクチャ設計 (Architecture)

本プロジェクトは、**Hexagonal Architecture (Ports and Adapters)** と  
**Serverless Orchestration (Cloud Run + Cloud Tasks)** を組み合わせた構成です。

1. **Domain 層 (The Core)**
   - `MusicRecipe`, `Task`, `PublishResult` など、外部技術に依存しないドメインモデルとポートを定義。
2. **Pipeline 層 (Orchestrator)**
   - Collect → Compose → Generate → Publish を統制し、処理順序とエラーハンドリングを担う。
3. **Server 層 (Entry Points)**
   - Web Handler: リクエスト受付とタスク投入。
   - Worker Handler: Cloud Tasks から受けたジョブを実行。
4. **Adapters 層 (Infrastructure)**
   - Lyria API / GCS / Slack / Cloud Tasks など外部サービスとの接続実装。
5. **Builder 層 (Dependency Injection)**
   - Web 実行系と Worker 実行系を用途別に組み立てる DI コンテナ。

---

## 🏗 プロジェクトレイアウト (Project Layout)

```text
ap-music/
├── internal/
│   ├── adapters/              # 外部サービス・SDKの具象実装
│   │   ├── lyria.go           # Lyria 3 API を叩く音楽生成クライアント
│   │   ├── reader.go          # go-web-reader を用いたコンテンツ収集
│   │   └── slack.go           # Slack Webhook 通知
│   ├── app/                   # アプリケーション共通のライフサイクル管理
│   │   └── app.go
│   ├── builder/               # DI Container（依存関係の注入）
│   │   └── builder.go         # Web/Worker 用の Pipeline や Handler を組み立て
│   ├── config/                # 環境変数管理
│   │   └── config.go          # Config 構造体と LoadConfig 関数
│   ├── controllers/           # エントリポイントごとのハンドラー
│   │   ├── web/               # ユーザー向け Web UI / Form 受付
│   │   │   └── handler.go
│   │   └── worker/            # Cloud Tasks から呼び出されるジョブ実行口
│   │       └── handler.go
│   ├── domain/                # ドメインモデルとインターフェース (Ports)
│   │   ├── music_recipe.go    # 楽曲設計図の構造体
│   │   ├── task.go            # ジョブ・タスクの定義
│   │   ├── repository.go      # 保存・通知などの Interface 定義
│   │   └── service.go         # 音楽生成エンジンの Interface 定義
│   ├── pipeline/              # ワークフローのオーケストレーション
│   │   ├── workflow.go        # Interface 定義
│   │   └── music_pipeline.go  # Collect -> Compose -> Generate -> Publish の統制
│   ├── prompts/               # LLM用プロンプト管理
│   │   └── recipe_builder.go  # コンテキストから MusicRecipe を構築するロジック
│   └── server/                # HTTP サーバーのルーティング設定
│       └── router.go          # chi 等を用いたルーティング定義
├── assets/                    # 静的ファイル、固定プロンプト素材、Web UI 用の HTML テンプレート
├── docs/                      # アーキテクチャ図、API仕様
└── main.go            # 【起点】アプリのブートストラップ（初期化・起動）
```

---

## 🔄 シーケンスフロー (Sequence Flow)

```mermaid
sequenceDiagram
    participant User as User (Web UI)
    participant Web as Web (Cloud Run)
    participant Queue as Cloud Tasks
    participant Worker as Worker (Cloud Run)
    participant Pipeline as Music Pipeline
    participant Collector as Collector (go-web-reader)
    participant Composer as Composer (Recipe Builder)
    participant Lyria as Lyria 3 API
    participant GCS as Cloud Storage
    participant Slack as Slack Notification

    User->>Web: フォーム送信 (URL/Text/Image/Model)
    Web->>Queue: タスクをエンキュー (Async)
    Web-->>User: 受付完了レスポンス

    Queue->>Worker: HTTP POST (タスク実行)
    Worker->>Pipeline: Execute() 起動

    Pipeline->>Collector: 入力コンテキスト収集
    Collector-->>Pipeline: Text/Image Context
    Pipeline->>Composer: MusicRecipe を生成
    Composer-->>Pipeline: Recipe(JSON)

    rect rgba(240, 240, 240, 0.1)
        Note over Pipeline, Lyria: 生成フェーズ
        Pipeline->>Lyria: 音楽生成リクエスト
        Lyria-->>Pipeline: MP3 Binary
    end

    Pipeline->>GCS: 成果物保存
    Pipeline->>GCS: Signed URL 発行
    Pipeline->>Slack: 完了通知 (閲覧URL / JobID)
```

---

## ✨ 技術スタック (Technology Stack)

| 要素 | 技術 / ライブラリ | 役割 |
| --- | --- | --- |
| **言語** | **Go (Golang)** | Web サーバーおよびワーカー実装 |
| **Web** | **Cloud Run** | Web UI/API と Worker の実行基盤 |
| **非同期実行** | **Google Cloud Tasks** | 楽曲生成ジョブの非同期キューイング |
| **コンテキスト収集** | **go-web-reader** | URL / 画像の収集と抽出 |
| **音楽生成** | **Lyria 3 API** | Recipe ベースの音楽生成 |
| **結果保存** | **go-remote-io / GCS** | MP3 保存、署名付き URL 発行 |
| **通知** | **Slack Webhook** | 実行完了通知 |

---

## 🚀 使い方 (Usage)

### 1. Web 経由の基本フロー

1. Web UI で入力（URL/Text/Image）とモデルを指定。
2. ジョブ送信後、Cloud Tasks へ非同期投入。
3. Worker が楽曲生成し、保存先 URI と Signed URL を発行。
4. Slack 通知で結果を受け取る。

### 2. 主要な環境変数

| 環境変数 | 必須 | 説明 |
| --- | :---: | --- |
| `SERVICE_URL` | 必須 | アプリの公開 URL |
| `GCP_PROJECT_ID` | 必須 | GCP プロジェクト ID |
| `GCP_LOCATION_ID` | 必須 | 使用リージョン |
| `CLOUD_TASKS_QUEUE_ID` | 必須 | Cloud Tasks キュー名 |
| `SERVICE_ACCOUNT_EMAIL` | 必須 | タスク実行に使うサービスアカウント |
| `TASK_AUDIENCE_URL` | 任意 | OIDC Audience (認証が必要な場合) |
| `GCS_MUSIC_BUCKET` | 必須 | 生成 MP3 の保存先バケット |
| `LYRIA_MODEL` | 任意 | 使用する Lyria モデル名 (デフォルト値がある場合) |
| `SLACK_WEBHOOK_URL` | 任意 | 完了通知先 Webhook URL |

---

## 🔗 エコシステム連携 (Evolution)

- **[AP Chain](https://github.com/shouni/ap-chain) 連携**: 構造化ドキュメントからテーマ曲を自動生成。
- **[AP Voice](https://github.com/shouni/ap-voice) 連携**: ナレーション音声と BGM を合成し音声コンテンツ化。
- **[AP Manga Web](https://github.com/shouni/ap-manga-web) 連携**: 作品ページやシーンごとのBGMを非同期生成。

---

## 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
