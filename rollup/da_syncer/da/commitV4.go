package da

import (
	"context"

	"github.com/scroll-tech/da-codec/encoding/codecv4"

	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/rollup_sync_service"

	"github.com/scroll-tech/go-ethereum/core/types"
)

type CommitBatchDAV4 struct {
	*CommitBatchDAV1
}

func NewCommitBatchDAV4(ctx context.Context, db ethdb.Database,
	l1Client *rollup_sync_service.L1Client,
	blobClient blob_client.BlobClient,
	vLog *types.Log,
	version uint8,
	batchIndex uint64,
	parentBatchHeader []byte,
	chunks [][]byte,
	skippedL1MessageBitmap []byte,
) (*CommitBatchDAV2, error) {

	v1, err := NewCommitBatchDAV1WithBlobDecodeFunc(ctx, db, l1Client, blobClient, vLog, version, batchIndex, parentBatchHeader, chunks, skippedL1MessageBitmap, codecv4.DecodeTxsFromBlob)
	if err != nil {
		return nil, err
	}

	return &CommitBatchDAV2{v1}, nil
}

func (c *CommitBatchDAV4) Type() Type {
	return CommitBatchV4Type
}
