package domain

// ImagePayload は、画像の構造体です
type ImagePayload struct {
	Data     []byte
	MIMEType string
}

// CollectedContent はマルチモーダルなコンテンツを表現する構造体です。
type CollectedContent struct {
	Prompt string
	Images []ImagePayload
}
