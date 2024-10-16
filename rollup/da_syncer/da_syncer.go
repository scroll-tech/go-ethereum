package da_syncer

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
)

var (
	ErrBlockTooLow  = fmt.Errorf("block number is too low")
	ErrBlockTooHigh = fmt.Errorf("block number is too high")
)

type DASyncer struct {
	l2EndBlock uint64
	blockchain *core.BlockChain
}

func NewDASyncer(blockchain *core.BlockChain, l2EndBlock uint64) *DASyncer {
	return &DASyncer{
		l2EndBlock: l2EndBlock,
		blockchain: blockchain,
	}
}

// SyncOneBlock receives a PartialBlock, makes sure it's the next block in the chain, executes it and inserts it to the blockchain.
func (s *DASyncer) SyncOneBlock(block *da.PartialBlock, override bool, sign bool) error {
	currentBlock := s.blockchain.CurrentBlock()

	// we expect blocks to be consecutive. block.PartialHeader.Number == parentBlock.Number+1.
	// if override is true, we allow blocks to be lower than the current block number and replace the blocks.
	if !override && block.PartialHeader.Number <= currentBlock.Number.Uint64() {
		log.Debug("block number is too low", "block number", block.PartialHeader.Number, "parent block number", currentBlock.Number.Uint64())
		return ErrBlockTooLow
	} else if block.PartialHeader.Number > currentBlock.Number.Uint64()+1 {
		log.Debug("block number is too high", "block number", block.PartialHeader.Number, "parent block number", currentBlock.Number.Uint64())
		return ErrBlockTooHigh
	}

	parentBlockNumber := currentBlock.Number.Uint64()
	if override {
		parentBlockNumber = block.PartialHeader.Number - 1
	}

	parentBlock := s.blockchain.GetBlockByNumber(parentBlockNumber)
	if parentBlock == nil {
		return fmt.Errorf("failed getting parent block, number: %d", parentBlockNumber)
	}

	if _, err := s.blockchain.BuildAndWriteBlock(parentBlock, block.PartialHeader.ToHeader(), block.Transactions, sign); err != nil {
		return fmt.Errorf("failed building and writing block, number: %d, error: %v", block.PartialHeader.Number, err)
	}

	currentBlock = s.blockchain.CurrentBlock()
	if override && block.PartialHeader.Number != currentBlock.Number.Uint64() && block.PartialHeader.Number%100 == 0 {
		newBlock := s.blockchain.GetHeaderByNumber(block.PartialHeader.Number)
		log.Info("L1 sync progress", "processed block ", newBlock.Number.Uint64(), "block hash", newBlock.Hash(), "root", newBlock.Root)
		log.Info("L1 sync progress", "blockhain height", currentBlock.Number.Uint64(), "block hash", currentBlock.Hash(), "root", currentBlock.Root)
	} else if currentBlock.Number.Uint64()%100 == 0 {
		log.Info("L1 sync progress", "blockhain height", currentBlock.Number.Uint64(), "block hash", currentBlock.Hash(), "root", currentBlock.Root)
	}

	if s.l2EndBlock > 0 && s.l2EndBlock == block.PartialHeader.Number {
		newBlock := s.blockchain.GetHeaderByNumber(block.PartialHeader.Number)
		log.Warn("L1 sync reached L2EndBlock: you can terminate recovery mode now", "L2EndBlock", newBlock.Number.Uint64(), "block hash", newBlock.Hash(), "root", newBlock.Root)
		return serrors.Terminated
	}

	return nil
}
