package adapters

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	"github.com/shouni/go-remote-io/remoteio"
)

// StorageCleaner は、GCS またはローカルファイルのベストエフォート削除を提供します。
type StorageCleaner struct {
	gcsClient *storage.Client
}

func NewStorageCleaner(gcsClient *storage.Client) *StorageCleaner {
	return &StorageCleaner{gcsClient: gcsClient}
}

func (c *StorageCleaner) Delete(ctx context.Context, uri string) error {
	if remoteio.IsGCSURI(uri) {
		if c.gcsClient == nil {
			return fmt.Errorf("gcs client is required to delete %s", uri)
		}

		bucketName, objectPath, err := remoteio.ParseRemoteURI(uri)
		if err != nil {
			return fmt.Errorf("failed to parse gcs uri %s: %w", uri, err)
		}

		if err := c.gcsClient.Bucket(bucketName).Object(objectPath).Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete gcs object %s: %w", uri, err)
		}

		return nil
	}

	if err := os.Remove(uri); err != nil {
		return fmt.Errorf("failed to delete local file %s: %w", uri, err)
	}

	return nil
}
