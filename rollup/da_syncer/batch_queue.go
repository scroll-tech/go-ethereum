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
	DAQueue                 *DAQueue
	db                      ethdb.Database
	lastFinalizedBatchIndex uint64
	batches                 *common.Heap[da.Entry]
}

func NewBatchQueue(DAQueue *DAQueue, db ethdb.Database) *BatchQueue {
	return &BatchQueue{
		DAQueue:                 DAQueue,
		db:                      db,
		lastFinalizedBatchIndex: 0,
		batches:                 common.NewHeap[da.Entry](),
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
			if daEntry.BatchIndex() > bq.lastFinalizedBatchIndex {
				bq.lastFinalizedBatchIndex = daEntry.BatchIndex()
			}

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

	if bq.batches.Len() == 0 {
		curBatchL1Height := batch.L1BlockNumber()
		rawdb.WriteDASyncedL1BlockNumber(bq.db, curBatchL1Height)
		return
	}

	// we store here min height of currently loaded batches to be able to start syncing from the same place in case of restart
	rawdb.WriteDASyncedL1BlockNumber(bq.db, bq.batches.Peek().L1BlockNumber()-1)
}
