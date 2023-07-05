package rollup

import (
    "sync"

    "github.com/scroll-tech/go-ethereum/common"
    "github.com/scroll-tech/go-ethereum/core/state"
    "github.com/scroll-tech/go-ethereum/core/types"
    "github.com/scroll-tech/go-ethereum/core/vm"
    "github.com/scroll-tech/go-ethereum/core"
    "github.com/scroll-tech/go-ethereum/common/hexutil"
    "github.com/scroll-tech/go-ethereum/log"
    "github.com/scroll-tech/go-ethereum/params"
)

type TraceEnv struct {
    LogConfig *vm.LogConfig

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

func CreateTraceEnv(logConfig *vm.LogConfig,chainConfig *params.ChainConfig, parent *types.Block, block *types.Block, coinbase common.Address, chainContext core.ChainContext, statedb *state.StateDB) (*TraceEnv, error) {
    env := &TraceEnv{
        LogConfig: logConfig,
        Coinbase:  coinbase,
        Signer:    types.MakeSigner(chainConfig, block.Number()),
        State:     statedb,
        BlockCtx:  core.NewEVMBlockContext(block.Header(), chainContext, nil),
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