package da

import (
	"context"

	"github.com/scroll-tech/da-codec/encoding/codecv4"

	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

type CommitBatchDAV4 struct {
	*CommitBatchDAV1
}

func NewCommitBatchDAV4(ctx context.Context, msgStorage *l1.MsgStorage,
	l1Reader *l1.Reader,
	blobClient blob_client.BlobClient,
	commitEvent *l1.CommitBatchEvent,
	version uint8,
	batchIndex uint64,
	parentBatchHeader []byte,
	chunks [][]byte,
	skippedL1MessageBitmap []byte,
) (*CommitBatchDAV2, error) {

	v1, err := NewCommitBatchDAV1WithBlobDecodeFunc(ctx, msgStorage, l1Reader, blobClient, commitEvent, version, batchIndex, parentBatchHeader, chunks, skippedL1MessageBitmap, codecv4.DecodeTxsFromBlob)
	if err != nil {
		return nil, err
	}

	return &CommitBatchDAV2{v1}, nil
}

func (c *CommitBatchDAV4) Type() Type {
	return CommitBatchV4Type
}
