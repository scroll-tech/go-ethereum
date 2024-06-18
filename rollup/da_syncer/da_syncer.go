package da_syncer

import (
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/trie"
)

type DaSyncer struct {
	blockchain *core.BlockChain
}

func NewDaSyncer(blockchain *core.BlockChain) *DaSyncer {
	return &DaSyncer{
		blockchain: blockchain,
	}
}

func (s *DaSyncer) SyncOneBlock(block *types.Block) error {
	prevHash := s.blockchain.CurrentBlock().Hash()
	if big.NewInt(0).Add(s.blockchain.CurrentBlock().Number(), common.Big1).Cmp(block.Number()) != 0 {
		return fmt.Errorf("not consecutive block, number: %d", block.Number())
	}
	header := block.Header()
	txs := block.Transactions()

	// fill header with all necessary fields
	var err error
	header.ReceiptHash, header.Bloom, header.Root, header.GasUsed, err = s.blockchain.PreprocessBlock(block)
	if err != nil {
		return fmt.Errorf("block preprocessing failed, block number: %d, error: %v", block.Number(), err)
	}
	header.UncleHash = common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347")
	header.Difficulty = common.Big1
	header.BaseFee = nil
	header.TxHash = types.DeriveSha(txs, trie.NewStackTrie(nil))
	header.ParentHash = prevHash

	fullBlock := types.NewBlockWithHeader(header).WithBody(txs, make([]*types.Header, 0))

	if _, err := s.blockchain.InsertChainWithoutSealVerification(fullBlock); err != nil {
		return fmt.Errorf("cannot insert block, number: %d, error: %v", block.Number(), err)
	}
	if s.blockchain.CurrentBlock().Header().Number.Uint64()%100 == 0 {
		log.Info("inserted block", "blockhain height", s.blockchain.CurrentBlock().Header().Number, "block hash", s.blockchain.CurrentBlock().Header().Hash())
	}
	return nil
}
