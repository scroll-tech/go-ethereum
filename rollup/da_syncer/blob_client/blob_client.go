package blob_client

import (
	"context"
	"fmt"

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

type BlobSource int

const (
	// AnyBlobSource
	AnyBlobSource BlobSource = iota
	// BlobScan
	BlobScan
	// BlockNative
	BlockNative
)

func (src BlobSource) IsValid() bool {
	return src >= AnyBlobSource && src <= BlockNative
}

// String implements the stringer interface.
func (src BlobSource) String() string {
	switch src {
	case AnyBlobSource:
		return "any"
	case BlobScan:
		return "blobscan"
	case BlockNative:
		return "blocknative"
	default:
		return "unknown"
	}
}

func (src BlobSource) MarshalText() ([]byte, error) {
	switch src {
	case AnyBlobSource:
		return []byte("any"), nil
	case BlobScan:
		return []byte("blobscan"), nil
	case BlockNative:
		return []byte("blocknative"), nil
	default:
		return nil, fmt.Errorf("unknown blob source %d", src)
	}
}

func (src *BlobSource) UnmarshalText(text []byte) error {
	switch string(text) {
	case "any":
		*src = AnyBlobSource
	case "blobscan":
		*src = BlobScan
	case "blocknative":
		*src = BlockNative
	default:
		return fmt.Errorf(`unknown blob source %q, want "any", "blobscan" or "blocknative"`, text)
	}
	return nil
}
