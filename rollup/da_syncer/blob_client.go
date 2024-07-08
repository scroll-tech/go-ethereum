package da_syncer

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

type BLobSource int

const (
	// BlobScan
	BlobScan BLobSource = iota
	// BlockNative
	BlockNative
)

func (src BLobSource) IsValid() bool {
	return src >= BlobScan && src <= BlockNative
}

// String implements the stringer interface.
func (src BLobSource) String() string {
	switch src {
	case BlobScan:
		return "blobscan"
	case BlockNative:
		return "blocknative"
	default:
		return "unknown"
	}
}

func (src BLobSource) MarshalText() ([]byte, error) {
	switch src {
	case BlobScan:
		return []byte("blobscan"), nil
	case BlockNative:
		return []byte("blocknative"), nil
	default:
		return nil, fmt.Errorf("unknown blob source %d", src)
	}
}

func (src *BLobSource) UnmarshalText(text []byte) error {
	switch string(text) {
	case "blobscan":
		*src = BlobScan
	case "blocknative":
		*src = BlockNative
	default:
		return fmt.Errorf(`unknown blob source %q, want "blobscan" or "blocknative"`, text)
	}
	return nil
}
