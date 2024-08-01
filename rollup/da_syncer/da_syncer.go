package da_syncer

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
)

var (
	ErrBlockTooLow  = fmt.Errorf("block number is too low")
	ErrBlockTooHigh = fmt.Errorf("block number is too high")
)

type DASyncer struct {
	blockchain *core.BlockChain
}

func NewDASyncer(blockchain *core.BlockChain) *DASyncer {
	return &DASyncer{
		blockchain: blockchain,
	}
}

func (s *DASyncer) SyncOneBlock(block *da.PartialBlock) error {
	parentBlock := s.blockchain.CurrentBlock()
	if block.PartialHeader.Number <= parentBlock.NumberU64() {
		log.Debug("block number is too low", "block number", block.PartialHeader.Number, "parent block number", parentBlock.NumberU64())
		return ErrBlockTooLow
	} else if block.PartialHeader.Number > parentBlock.NumberU64()+1 {
		log.Debug("block number is too high", "block number", block.PartialHeader.Number, "parent block number", parentBlock.NumberU64())
		return ErrBlockTooHigh
	}
	if parentBlock.NumberU64()+1 != block.PartialHeader.Number {
		return fmt.Errorf("not consecutive block, number: %d, chain height: %d", block.PartialHeader.Number, parentBlock.NumberU64())
	}

	if _, err := s.blockchain.BuildAndWriteBlock(parentBlock, block.PartialHeader.ToHeader(), block.Transactions); err != nil {
		return fmt.Errorf("failed building and writing block, number: %d, error: %v", block.PartialHeader.Number, err)
	}

	if s.blockchain.CurrentBlock().Header().Number.Uint64()%1000 == 0 {
		log.Info("L1 sync progress", "blockhain height", s.blockchain.CurrentBlock().Header().Number, "block hash", s.blockchain.CurrentBlock().Header().Hash(), "root", s.blockchain.CurrentBlock().Header().Root)
	}

	return nil
}
