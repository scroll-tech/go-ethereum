package da_syncer

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
)

type BlockQueue struct {
	batchQueue *BatchQueue
	blocks     []*types.Block
}

func NewBlockQueue(batchQueue *BatchQueue) *BlockQueue {
	return &BlockQueue{
		batchQueue: batchQueue,
		blocks:     []*types.Block{},
	}
}

func (bq *BlockQueue) NextBlock(ctx context.Context) (*types.Block, error) {
	for len(bq.blocks) == 0 {
		err := bq.getBlocksFromBatch(ctx)
		if err != nil {
			return nil, err
		}
	}
	block := bq.blocks[0]
	bq.blocks = bq.blocks[1:]
	return block, nil
}

func (bq *BlockQueue) getBlocksFromBatch(ctx context.Context) error {
	daEntry, err := bq.batchQueue.NextBatch(ctx)
	if err != nil {
		return err
	}

	entryWithBlocks, ok := daEntry.(da.EntryWithBlocks)
	// this should never happen because we only receive CommitBatch entries
	if !ok {
		return fmt.Errorf("unexpected type of daEntry: %T", daEntry)
	}

	bq.blocks, err = entryWithBlocks.Blocks()
	if err != nil {
		return fmt.Errorf("failed to get blocks from daEntry: %w", err)
	}

	return nil
}
