package stateprocessor

import (
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/rollup/circuitcapacitychecker"
	"github.com/scroll-tech/go-ethereum/rollup/pipeline"
)

var _ core.Processor = (*Processor)(nil)

type Processor struct {
	chain *core.BlockChain
	ccc   *circuitcapacitychecker.CircuitCapacityChecker
}

func NewProcessor(bc *core.BlockChain) *Processor {
	return &Processor{
		chain: bc,
		ccc:   circuitcapacitychecker.NewCircuitCapacityChecker(true),
	}
}

func (p *Processor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	if block.Transactions().Len() == 0 {
		return types.Receipts{}, []*types.Log{}, 0, nil
	}

	header := block.Header()
	header.GasUsed = 0

	nextL1MsgIndex := uint64(0)
	// assume L1 message indexes were validated by block validator
	if block.Transactions().Len() > 0 {
		if l1Msg := block.Transactions()[0].AsL1MessageTx(); l1Msg != nil {
			nextL1MsgIndex = l1Msg.QueueIndex
		}
	}

	pl := pipeline.NewPipeline(p.chain, cfg, statedb, header, nextL1MsgIndex, p.ccc).WithReplayMode()
	pl.Start(time.Now().Add(time.Minute))
	defer pl.Release()

	for _, tx := range block.Transactions() {
		res, err := pl.TryPushTxn(tx)
		if err != nil && !errors.Is(err, pipeline.ErrUnexpectedL1MessageIndex) {
			return nil, nil, 0, err
		}

		if res != nil {
			return nil, nil, 0, fmt.Errorf("pipeline ended prematurely %v", res.CCCErr)
		}
	}

	pl.Stop()
	res := <-pl.ResultCh
	if res.CCCErr != nil {
		return nil, nil, 0, res.CCCErr
	}

	return res.FinalBlock.Receipts, res.FinalBlock.CoalescedLogs, res.FinalBlock.Header.GasUsed, nil
}
