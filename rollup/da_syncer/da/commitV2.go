package da

import (
	"context"

	"github.com/scroll-tech/da-codec/encoding/codecv2"

	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

type CommitBatchDAV2 struct {
	*CommitBatchDAV1
}

func NewCommitBatchDAV2(ctx context.Context, db ethdb.Database,
	l1Reader *l1.Reader,
	blobClient blob_client.BlobClient,
	commitEvent *l1.CommitBatchEvent,
	version uint8,
	batchIndex uint64,
	parentBatchHeader []byte,
	chunks [][]byte,
	skippedL1MessageBitmap []byte,
) (*CommitBatchDAV2, error) {

	v1, err := NewCommitBatchDAV1WithBlobDecodeFunc(ctx, db, l1Reader, blobClient, commitEvent, version, batchIndex, parentBatchHeader, chunks, skippedL1MessageBitmap, codecv2.DecodeTxsFromBlob)
	if err != nil {
		return nil, err
	}

	return &CommitBatchDAV2{v1}, nil
}

func (c *CommitBatchDAV2) Type() Type {
	return CommitBatchV2Type
}
