package featureflag

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
)

type Bucket struct {
	Object   *storage.ObjectHandle
	checksum string
	data     []byte
}

func (b *Bucket) Retrieve(ctx context.Context) ([]byte, error) {
	if b.Object == nil {
		return nil, errors.New("object is nil for retriever")
	}

	attr, err := b.Object.Attrs(ctx)
	if err != nil {
		err = errors.Wrapf(err, "cannot read attributes of object(%s)", b.Object.ObjectName())
		return nil, err
	}

	if hex.EncodeToString(attr.MD5) == b.checksum {
		return b.data, nil
	}

	rc, err := b.Object.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		err = errors.Wrapf(err, "cannot read data of object(%s)", b.Object.ObjectName())
		return nil, err
	}

	sum := md5.Sum(data)
	b.checksum = hex.EncodeToString(sum[:])
	b.data = data

	return data, err
}
