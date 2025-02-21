package da_syncer

import (
	"context"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
)

// BatchQueue is a pipeline stage that reads all batch events from DAQueue and provides only finalized batches to the next stage.
type BatchQueue struct {
	DAQueue                 *DAQueue
	db                      ethdb.Database
	lastFinalizedBatchIndex uint64
	batches                 *common.Heap[da.Entry]
	batchesMap              *common.ShrinkingMap[uint64, *common.HeapElement[da.Entry]]

	previousBatch *rawdb.DAProcessedBatchMeta
}

func NewBatchQueue(DAQueue *DAQueue, db ethdb.Database, lastProcessedBatch *rawdb.DAProcessedBatchMeta) *BatchQueue {
	return &BatchQueue{
		DAQueue:                 DAQueue,
		db:                      db,
		lastFinalizedBatchIndex: lastProcessedBatch.BatchIndex,
		batches:                 common.NewHeap[da.Entry](),
		batchesMap:              common.NewShrinkingMap[uint64, *common.HeapElement[da.Entry]](1000),
		previousBatch:           lastProcessedBatch,
	}
}

// NextBatch finds next finalized batch and returns data, that was committed in that batch
func (bq *BatchQueue) NextBatch(ctx context.Context) (da.EntryWithBlocks, error) {
	if batch := bq.getFinalizedBatch(); batch != nil {
		return batch, nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		daEntry, err := bq.DAQueue.NextDA(ctx)
		if err != nil {
			return nil, err
		}
		switch daEntry.Type() {
		case da.CommitBatchV0Type, da.CommitBatchWithBlobType:
			bq.addBatch(daEntry)
		case da.RevertBatchType:
			bq.processAndDeleteBatch(daEntry)
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
func (bq *BatchQueue) getFinalizedBatch() da.EntryWithBlocks {
	if bq.batches.Len() == 0 {
		return nil
	}

	batch := bq.batches.Peek().Value()
	// we process all batches smaller or equal to the last finalized batch index -> this reflects bundles of multiple batches
	// where we only receive the finalize event for the last batch of the bundle.
	if batch.BatchIndex() <= bq.lastFinalizedBatchIndex {
		return bq.processAndDeleteBatch(batch)
	} else {
		return nil
	}
}

func (bq *BatchQueue) addBatch(batch da.Entry) {
	heapElement := bq.batches.Push(batch)
	bq.batchesMap.Set(batch.BatchIndex(), heapElement)
}

// processAndDeleteBatch deletes data committed in the batch from map, because this batch is reverted or finalized
// updates DASyncedL1BlockNumber
func (bq *BatchQueue) processAndDeleteBatch(batch da.Entry) da.EntryWithBlocks {
	batchHeapElement, exists := bq.batchesMap.Get(batch.BatchIndex())
	if !exists {
		return nil
	}

	bq.batchesMap.Delete(batch.BatchIndex())
	bq.batches.Remove(batchHeapElement)

	entryWithBlocks, ok := batch.(da.EntryWithBlocks)
	if !ok {
		// this should only happen if we delete a reverted batch
		return nil
	}

	// sanity check that the next batch is the one we expect
	if bq.previousBatch.BatchIndex > 0 && bq.previousBatch.BatchIndex+1 != entryWithBlocks.BatchIndex() {
		log.Info("BatchQueue: skipping batch ", "currentBatch", entryWithBlocks.BatchIndex(), "previousBatch", bq.previousBatch.BatchIndex)
		return nil
	}

	// carry forward the total L1 messages popped from the previous batch
	entryWithBlocks.SetParentTotalL1MessagePopped(bq.previousBatch.TotalL1MessagesPopped)

	// we store the previous batch as it has been completely processed which we know because the next batch is requested within the pipeline.
	// In case of a restart or crash we can continue from the last processed batch (and its metadata).
	rawdb.WriteDAProcessedBatchMeta(bq.db, bq.previousBatch)

	log.Info("processing batch", "batchIndex", entryWithBlocks.BatchIndex(), "L1BlockNumber", entryWithBlocks.L1BlockNumber(), "totalL1MessagesPopped", entryWithBlocks.TotalL1MessagesPopped(), "previousBatch", bq.previousBatch.BatchIndex, "previousL1BlockNumber", bq.previousBatch.L1BlockNumber, "previous TotalL1MessagesPopped", bq.previousBatch.TotalL1MessagesPopped)

	bq.previousBatch = &rawdb.DAProcessedBatchMeta{
		L1BlockNumber:         entryWithBlocks.L1BlockNumber(),
		BatchIndex:            entryWithBlocks.BatchIndex(),
		TotalL1MessagesPopped: entryWithBlocks.TotalL1MessagesPopped(),
	}

	return entryWithBlocks
}

func (bq *BatchQueue) Reset(lastProcessedBatchMeta *rawdb.DAProcessedBatchMeta) {
	bq.batches.Clear()
	bq.batchesMap.Clear()
	bq.lastFinalizedBatchIndex = lastProcessedBatchMeta.BatchIndex
	bq.previousBatch = lastProcessedBatchMeta
	bq.DAQueue.Reset(lastProcessedBatchMeta)
}
