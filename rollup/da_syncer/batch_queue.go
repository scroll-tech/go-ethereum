package da_syncer

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
)

type BatchQueue struct {
	DAQueue *DAQueue
	db      ethdb.Database
	batches *common.Heap[da.Entry]
}

func NewBatchQueue(DAQueue *DAQueue, db ethdb.Database) *BatchQueue {
	return &BatchQueue{
		DAQueue: DAQueue,
		db:      db,
		batches: common.NewHeap[da.Entry](),
	}
}

// NextBatch finds next finalized batch and returns data, that was committed in that batch
func (bq *BatchQueue) NextBatch(ctx context.Context) (da.Entry, error) {
	if batch := bq.getFinalizedBatch(); batch != nil {
		return batch, nil
	}

	for {
		daEntry, err := bq.DAQueue.NextDA(ctx)
		if err != nil {
			return nil, err
		}
		switch daEntry.Type() {
		case da.CommitBatchV0Type, da.CommitBatchV1Type, da.CommitBatchV2Type:
			bq.batches.Push(daEntry)
		case da.RevertBatchType:
			bq.deleteBatch(daEntry.BatchIndex())
		case da.FinalizeBatchType:
			// TODO: eventually we should match finalized batch with the one that was committed via batch header
			if batch := bq.getFinalizedBatch(); batch != nil {
				return batch, nil
			}
		default:
			return nil, fmt.Errorf("unexpected type of daEntry: %T", daEntry)
		}
	}
}

// getFinalizedBatch returns next finalized batch if there is available
func (bq *BatchQueue) getFinalizedBatch() da.Entry {
	if bq.batches.Len() == 0 {
		return nil
	}

	batch := bq.batches.Peek()
	bq.deleteBatch(batch.BatchIndex())

	return batch
}

// deleteBatch deletes data committed in the batch from map, because this batch is reverted or finalized
// updates DASyncedL1BlockNumber
func (bq *BatchQueue) deleteBatch(batchIndex uint64) {
	var batch da.Entry
	for batch = bq.batches.Peek(); batch.BatchIndex() <= batchIndex; {
		bq.batches.Pop()

		if bq.batches.Len() == 0 {
			break
		}
		batch = bq.batches.Peek()
	}

	// we store here min height of currently loaded batches to be able to start syncing from the same place in case of restart
	// TODO: we should store this information when the batch is done being processed to avoid inconsistencies
	rawdb.WriteDASyncedL1BlockNumber(bq.db, batch.L1BlockNumber()-1)
}

func (bq *BatchQueue) Reset(height uint64) {
	bq.batches = common.NewHeap[da.Entry]()
	bq.DAQueue.Reset(height)
}
