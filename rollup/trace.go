package rollup

import (
    "sync"

    "github.com/scroll-tech/go-ethereum/common"
    "github.com/scroll-tech/go-ethereum/core/state"
    "github.com/scroll-tech/go-ethereum/core/types"
    "github.com/scroll-tech/go-ethereum/core/vm"
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
