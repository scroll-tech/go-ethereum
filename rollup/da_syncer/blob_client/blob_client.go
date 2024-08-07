package blob_client

import (
	"context"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

const (
	okStatusCode int = 200
	lenBlobBytes int = 131072
)

type BlobClient interface {
	GetBlobByVersionedHash(ctx context.Context, versionedHash common.Hash) (*kzg4844.Blob, error)
}
