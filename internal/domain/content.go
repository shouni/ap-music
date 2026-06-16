package domain

const (
	// AudioFileExtension は公開成果物として保存する音声ファイルの拡張子です。
	AudioFileExtension = ".mp3"
	// LegacyAudioFileExtension はMP3移行前に保存されていた音声ファイルの拡張子です。
	LegacyAudioFileExtension = ".wav"
	// AudioContentType は公開成果物として保存する音声ファイルの MIME タイプです。
	AudioContentType = "audio/mpeg"
)

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
