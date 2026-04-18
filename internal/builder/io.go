package builder

import (
	"fmt"

	"ap-music/internal/app"

	"github.com/shouni/go-remote-io/remoteio"
)

// buildRemoteIO は、I/O コンポーネントを初期化します。
func buildRemoteIO(storage remoteio.IOFactory) (*app.RemoteIO, error) {
	if storage == nil {
		return &app.RemoteIO{
			Writer: remoteio.NewUniversalIOWriter(nil, nil),
		}, nil
	}

	r, err := storage.Reader()
	if err != nil {
		return nil, fmt.Errorf("failed to create input reader: %w", err)
	}
	w, err := storage.Writer()
	if err != nil {
		return nil, fmt.Errorf("failed to create output writer: %w", err)
	}
	s, err := storage.URLSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to create URL signer: %w", err)
	}
	return &app.RemoteIO{
		Factory: storage,
		Reader:  r,
		Writer:  w,
		Signer:  s,
	}, nil
}
