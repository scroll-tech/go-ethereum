package da_syncer

import (
	"context"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

type BlobClient interface {
	GetBlobByVersionedHash(ctx context.Context, versionedHash common.Hash) (*kzg4844.Blob, error)
}
