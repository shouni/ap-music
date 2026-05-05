package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"

	"golang.org/x/sync/singleflight"

	"ap-music/internal/domain"
)

func singleflightKey(namespace string, parts ...string) string {
	hasher := sha256.New()
	for _, part := range parts {
		hasher.Write([]byte(strconv.Itoa(len(part))))
		hasher.Write([]byte{0})
		hasher.Write([]byte(part))
		hasher.Write([]byte{0})
	}

	return namespace + ":" + hex.EncodeToString(hasher.Sum(nil))
}

func singleflightSeedKey(seed *int64) string {
	if seed == nil {
		return "seed:nil"
	}
	return "seed:" + strconv.FormatInt(*seed, 10)
}

func calculateImagesHash(images []domain.ImagePayload) string {
	hasher := sha256.New()
	lengthBuf := make([]byte, 8)
	for _, image := range images {
		if len(image.Data) == 0 {
			continue
		}

		mimeType := image.MIMEType
		hasher.Write([]byte(mimeType))
		hasher.Write([]byte{0})
		binary.LittleEndian.PutUint64(lengthBuf, uint64(len(image.Data)))
		hasher.Write(lengthBuf)
		hasher.Write(image.Data)
		hasher.Write([]byte{0})
	}

	return "images:" + hex.EncodeToString(hasher.Sum(nil))
}

func doSingleflight[T any](ctx context.Context, group *singleflight.Group, key string, fn func(execCtx context.Context) (T, error)) (T, error) {
	execCtx := context.WithoutCancel(ctx)
	ch := group.DoChan(key, func() (any, error) {
		return fn(execCtx)
	})

	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case result := <-ch:
		if result.Err != nil {
			var zero T
			return zero, result.Err
		}

		value, ok := result.Val.(T)
		if !ok {
			var zero T
			return zero, fmt.Errorf("singleflight result type mismatch for key %s", key)
		}
		return value, nil
	}
}

func cloneLyricsDraft(src *domain.LyricsDraft) *domain.LyricsDraft {
	if src == nil {
		return nil
	}

	dst := *src
	dst.Keywords = append([]string(nil), src.Keywords...)
	return &dst
}

func cloneMusicRecipe(src *domain.MusicRecipe) *domain.MusicRecipe {
	if src == nil {
		return nil
	}

	dst := *src
	dst.Instruments = append([]string(nil), src.Instruments...)
	if src.Sections != nil {
		dst.Sections = make([]domain.MusicSection, len(src.Sections))
		for i, sec := range src.Sections {
			dst.Sections[i] = sec
		}
	}
	dst.Lyrics = cloneLyricsDraft(src.Lyrics)
	if src.AIModels.Seed != nil {
		dst.AIModels.Seed = new(*src.AIModels.Seed)
	}
	return &dst
}

func cloneBytes(src []byte) []byte {
	return append([]byte(nil), src...)
}
