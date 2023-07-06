package core

import (
	// "errors"
	// "fmt"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/consensus"
	// "github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	// "github.com/scroll-tech/go-ethereum/rollup/circuitcapacitychecker"
	// "github.com/scroll-tech/go-ethereum/trie"
)

type TraceEnv struct {
	LogConfig   *vm.LogConfig
	ChainConfig *params.ChainConfig

	Coinbase common.Address

	// rMu lock is used to protect txs executed in parallel.
	Signer   types.Signer
	State    *state.StateDB
	BlockCtx vm.BlockContext

	// pMu lock is used to protect Proofs' read and write mutual exclusion,
	// since txs are executed in parallel, so this lock is required.
	PMu sync.Mutex
	// sMu is required because of txs are executed in parallel,
	// this lock is used to protect StorageTrace's read and write mutual exclusion.
	SMu sync.Mutex
	*types.StorageTrace
	TxStorageTraces []*types.StorageTrace
	// zktrie tracer is used for zktrie storage to build additional deletion proof
	ZkTrieTracer     map[string]state.ZktrieProofTracer
	ExecutionResults []*types.ExecutionResult
}

func CreateTraceEnv(chainConfig *params.ChainConfig, bc *BlockChain, engine consensus.Engine, statedb *state.StateDB, parent *types.Block, block *types.Block) (*TraceEnv, error) {
	var coinbase common.Address
	var err error
	if chainConfig.Scroll.FeeVaultEnabled() {
		coinbase = *chainConfig.Scroll.FeeVaultAddress
	} else {
		coinbase, err = engine.Author(block.Header())
		if err != nil {
			return nil, err
		}
	}

	env := &TraceEnv{
		LogConfig: &vm.LogConfig{
			EnableMemory:     false,
			EnableReturnData: true,
		},
		ChainConfig: chainConfig,
		Coinbase:    coinbase,
		Signer:      types.MakeSigner(chainConfig, block.Number()),
		State:       statedb,
		BlockCtx:    NewEVMBlockContext(block.Header(), bc, nil),
		StorageTrace: &types.StorageTrace{
			RootBefore:    parent.Root(),
			RootAfter:     block.Root(),
			Proofs:        make(map[string][]hexutil.Bytes),
			StorageProofs: make(map[string]map[string][]hexutil.Bytes),
		},
		ZkTrieTracer:     make(map[string]state.ZktrieProofTracer),
		ExecutionResults: make([]*types.ExecutionResult, block.Transactions().Len()),
		TxStorageTraces:  make([]*types.StorageTrace, block.Transactions().Len()),
	}

	key := coinbase.String()
	if _, exist := env.Proofs[key]; !exist {
		proof, err := env.State.GetProof(coinbase)
		if err != nil {
			log.Error("Proof for coinbase not available", "coinbase", coinbase, "error", err)
			// but we still mark the proofs map with nil array
		}
		wrappedProof := make([]hexutil.Bytes, len(proof))
		for i, bt := range proof {
			wrappedProof[i] = bt
		}
		env.Proofs[key] = wrappedProof
	}

	return env, nil
}
