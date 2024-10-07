package blob_client

import (
	"context"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

const (
	lenBlobBytes     int = 131072
	lenKZGCommitment int = 48
)

type BlobClient interface {
	GetBlobByVersionedHashAndBlockTime(ctx context.Context, versionedHash common.Hash, blockTime uint64) (*kzg4844.Blob, error)
}
