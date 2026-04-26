package domain

const (
	NotAvailable = "N/A"
)

// NotificationRequest は Slack 等の通知コンポーネントで共有されるデータ構造です。
// 生成された漫画のメタデータを通知先に伝えるために使用します。
type NotificationRequest struct {
	// SourceURL は、元になった記事やスクリプトのURLです。
	SourceURL string `json:"source_url"`

	// OutputCategory は、出力先の種別です。(例: "manga-output", "character-design")
	OutputCategory string `json:"output_category"`

	// Seed は、生成に使用された（または生成によって決定された）乱数シード値です。
	Seed *int64 `json:"seed,omitempty"`
}
