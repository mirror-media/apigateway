package featureflag

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
)

type Bucket struct {
	Object *storage.ObjectHandle
}

func (b *Bucket) Retrieve(ctx context.Context) ([]byte, error) {
	rc, err := b.Object.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	return data, err

}
