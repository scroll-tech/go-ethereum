package da_syncer

import (
	"context"
	"fmt"
	"math"

	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
)

type BatchQueue struct {
	// batches is map from batchIndex to batch blocks
	batches map[uint64]DAEntry
	daQueue *DaQueue
	db      ethdb.Database
}

func NewBatchQueue(daQueue *DaQueue, db ethdb.Database) *BatchQueue {
	return &BatchQueue{
		batches: make(map[uint64]DAEntry),
		daQueue: daQueue,
		db:      db,
	}
}

// NextBatch finds next finalized batch and returns data, that was committed in that batch
func (bq *BatchQueue) NextBatch(ctx context.Context) (DAEntry, error) {

	for {
		daEntry, err := bq.daQueue.NextDA(ctx)
		if err != nil {
			return nil, err
		}
		switch daEntry := daEntry.(type) {
		case *CommitBatchDaV0:
			bq.batches[daEntry.BatchIndex] = daEntry
		case *CommitBatchDaV1:
			bq.batches[daEntry.BatchIndex] = daEntry
		case *CommitBatchDaV2:
			bq.batches[daEntry.BatchIndex] = daEntry
		case *RevertBatchDA:
			bq.deleteBatch(daEntry.BatchIndex)
		case *FinalizeBatchDA:
			ret, ok := bq.batches[daEntry.BatchIndex]
			if !ok {
				// most probable, we met FinalizeBatch event for already committed batch after restart
				continue
			}
			bq.deleteBatch(daEntry.BatchIndex)
			return ret, nil
		default:
			return nil, fmt.Errorf("unexpected type of daEntry: %T", daEntry)
		}
	}
}

// deleteBatch deletes data committed in the batch, because this batch is reverted or finalized
// updates DASyncedL1BlockNumber
func (bq *BatchQueue) deleteBatch(batchIndex uint64) {
	batch, ok := bq.batches[batchIndex]
	if !ok {
		return
	}
	curBatchL1Height := batch.GetL1BlockNumber()
	delete(bq.batches, batchIndex)
	if len(bq.batches) == 0 {
		rawdb.WriteDASyncedL1BlockNumber(bq.db, curBatchL1Height)
		return
	}
	var minBatchL1Height uint64 = math.MaxUint64
	for _, val := range bq.batches {
		if val.GetL1BlockNumber() < minBatchL1Height {
			minBatchL1Height = val.GetL1BlockNumber()
		}
	}
	rawdb.WriteDASyncedL1BlockNumber(bq.db, curBatchL1Height-1)
}
