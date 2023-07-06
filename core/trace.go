package core

import (
    // "errors"
    // "fmt"
    "sync"

    "github.com/scroll-tech/go-ethereum/common"
    // "github.com/scroll-tech/go-ethereum/common/hexutil"
    // "github.com/scroll-tech/go-ethereum/consensus"
    // "github.com/scroll-tech/go-ethereum/core/rawdb"
    "github.com/scroll-tech/go-ethereum/core/state"
    "github.com/scroll-tech/go-ethereum/core/types"
    "github.com/scroll-tech/go-ethereum/core/vm"
    // "github.com/scroll-tech/go-ethereum/log"
    // "github.com/scroll-tech/go-ethereum/params"
    // "github.com/scroll-tech/go-ethereum/rollup/circuitcapacitychecker"
    // "github.com/scroll-tech/go-ethereum/trie"
)

type traceEnv struct {
    logConfig *vm.LogConfig

    coinbase common.Address

    // rMu lock is used to protect txs executed in parallel.
    signer   types.Signer
    state    *state.StateDB
    blockCtx vm.BlockContext

    // pMu lock is used to protect Proofs' read and write mutual exclusion,
    // since txs are executed in parallel, so this lock is required.
    pMu sync.Mutex
    // sMu is required because of txs are executed in parallel,
    // this lock is used to protect StorageTrace's read and write mutual exclusion.
    sMu sync.Mutex
    *types.StorageTrace
    txStorageTraces []*types.StorageTrace
    // zktrie tracer is used for zktrie storage to build additional deletion proof
    zkTrieTracer     map[string]state.ZktrieProofTracer
    executionResults []*types.ExecutionResult
}
