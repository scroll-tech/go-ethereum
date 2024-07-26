package da_syncer

import (
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

type DASyncer struct {
	blockchain *core.BlockChain
}

func NewDASyncer(blockchain *core.BlockChain) *DASyncer {
	return &DASyncer{
		blockchain: blockchain,
	}
}

func (s *DASyncer) SyncOneBlock(block *types.Block) error {
	parentBlock := s.blockchain.CurrentBlock()
	if big.NewInt(0).Add(parentBlock.Number(), common.Big1).Cmp(block.Number()) != 0 {
		return fmt.Errorf("not consecutive block, number: %d", block.Number())
	}

	header := block.Header()
	header.Difficulty = common.Big1
	header.BaseFee = nil // TODO: after Curie we need to fill this correctly
	header.ParentHash = parentBlock.Hash()

	if _, err := s.blockchain.BuildAndWriteBlock(parentBlock, header, block.Transactions()); err != nil {
		return fmt.Errorf("failed building and writing block, number: %d, error: %v", block.Number(), err)
	}

	if s.blockchain.CurrentBlock().Header().Number.Uint64()%100 == 0 {
		log.Info("inserted block", "blockhain height", s.blockchain.CurrentBlock().Header().Number, "block hash", s.blockchain.CurrentBlock().Header().Hash(), "root", s.blockchain.CurrentBlock().Header().Root)
	}
	return nil
}
