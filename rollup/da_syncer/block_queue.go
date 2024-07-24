package da_syncer

import (
	"context"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/core/types"
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
	switch daEntry := daEntry.(type) {
	case *CommitBatchDAV0:
		bq.blocks, err = bq.processDAV0ToBlocks(daEntry)
		if err != nil {
			return err
		}
	case *CommitBatchDAV1:
		bq.blocks, err = bq.processDAV1ToBlocks(daEntry)
		if err != nil {
			return err
		}
	case *CommitBatchDAV2:
		bq.blocks, err = bq.processDAV2ToBlocks(daEntry)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unexpected type of daEntry: %T", daEntry)
	}
	return nil
}

func (bq *BlockQueue) processDAV0ToBlocks(daEntry *CommitBatchDAV0) ([]*types.Block, error) {
	var blocks []*types.Block
	l1TxPointer := 0
	var curL1TxIndex uint64 = daEntry.ParentTotalL1MessagePopped
	for _, chunk := range daEntry.Chunks {
		for blockId, daBlock := range chunk.Blocks {
			// create header
			header := types.Header{
				Number:   big.NewInt(0).SetUint64(daBlock.BlockNumber),
				Time:     daBlock.Timestamp,
				BaseFee:  daBlock.BaseFee,
				GasLimit: daBlock.GasLimit,
			}
			// create txs
			// var txs types.Transactions
			txs := make(types.Transactions, 0, daBlock.NumTransactions)
			// insert l1 msgs
			for l1TxPointer < len(daEntry.L1Txs) && daEntry.L1Txs[l1TxPointer].QueueIndex < curL1TxIndex+uint64(daBlock.NumL1Messages) {
				l1Tx := types.NewTx(daEntry.L1Txs[l1TxPointer])
				txs = append(txs, l1Tx)
				l1TxPointer++
			}
			curL1TxIndex += uint64(daBlock.NumL1Messages)
			// insert l2 txs
			txs = append(txs, chunk.Transactions[blockId]...)
			block := types.NewBlockWithHeader(&header).WithBody(txs, make([]*types.Header, 0))
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}

func (bq *BlockQueue) processDAV1ToBlocks(daEntry *CommitBatchDAV1) ([]*types.Block, error) {
	var blocks []*types.Block
	l1TxPointer := 0
	var curL1TxIndex uint64 = daEntry.ParentTotalL1MessagePopped
	for _, chunk := range daEntry.Chunks {
		for blockId, daBlock := range chunk.Blocks {
			// create header
			header := types.Header{
				Number:   big.NewInt(0).SetUint64(daBlock.BlockNumber),
				Time:     daBlock.Timestamp,
				BaseFee:  daBlock.BaseFee,
				GasLimit: daBlock.GasLimit,
			}
			// create txs
			// var txs types.Transactions
			txs := make(types.Transactions, 0, daBlock.NumTransactions)
			// insert l1 msgs
			for l1TxPointer < len(daEntry.L1Txs) && daEntry.L1Txs[l1TxPointer].QueueIndex < curL1TxIndex+uint64(daBlock.NumL1Messages) {
				l1Tx := types.NewTx(daEntry.L1Txs[l1TxPointer])
				txs = append(txs, l1Tx)
				l1TxPointer++
			}
			curL1TxIndex += uint64(daBlock.NumL1Messages)
			// insert l2 txs
			txs = append(txs, chunk.Transactions[blockId]...)
			block := types.NewBlockWithHeader(&header).WithBody(txs, make([]*types.Header, 0))
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}

func (bq *BlockQueue) processDAV2ToBlocks(daEntry *CommitBatchDAV2) ([]*types.Block, error) {
	var blocks []*types.Block
	l1TxPointer := 0
	var curL1TxIndex uint64 = daEntry.ParentTotalL1MessagePopped
	for _, chunk := range daEntry.Chunks {
		for blockId, daBlock := range chunk.Blocks {
			// create header
			header := types.Header{
				Number:   big.NewInt(0).SetUint64(daBlock.BlockNumber),
				Time:     daBlock.Timestamp,
				BaseFee:  daBlock.BaseFee,
				GasLimit: daBlock.GasLimit,
			}
			// create txs
			// var txs types.Transactions
			txs := make(types.Transactions, 0, daBlock.NumTransactions)
			// insert l1 msgs
			for l1TxPointer < len(daEntry.L1Txs) && daEntry.L1Txs[l1TxPointer].QueueIndex < curL1TxIndex+uint64(daBlock.NumL1Messages) {
				l1Tx := types.NewTx(daEntry.L1Txs[l1TxPointer])
				txs = append(txs, l1Tx)
				l1TxPointer++
			}
			curL1TxIndex += uint64(daBlock.NumL1Messages)
			// insert l2 txs
			txs = append(txs, chunk.Transactions[blockId]...)
			block := types.NewBlockWithHeader(&header).WithBody(txs, make([]*types.Header, 0))
			blocks = append(blocks, block)
		}
	}
	return blocks, nil
}
