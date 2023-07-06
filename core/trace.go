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
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rollup/withdrawtrie"
	// "github.com/scroll-tech/go-ethereum/trie"
	"github.com/scroll-tech/go-ethereum/trie/zkproof"
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

func CreateTraceEnv(chainConfig *params.ChainConfig, chainContext ChainContext, engine consensus.Engine, statedb *state.StateDB, parent *types.Block, block *types.Block) (*TraceEnv, error) {
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
		BlockCtx:    NewEVMBlockContext(block.Header(), chainContext, nil),
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

// FillBlockTrace content after all the txs are finished running.
func (env *TraceEnv) FillBlockTrace(block *types.Block) (*types.BlockTrace, error) {
	statedb := env.State

	txs := make([]*types.TransactionData, block.Transactions().Len())
	for i, tx := range block.Transactions() {
		txs[i] = types.NewTransactionData(tx, block.NumberU64(), env.ChainConfig)
	}

	intrinsicStorageProofs := map[common.Address][]common.Hash{
		rcfg.L2MessageQueueAddress: {rcfg.WithdrawTrieRootSlot},
		rcfg.L1GasPriceOracleAddress: {
			rcfg.L1BaseFeeSlot,
			rcfg.OverheadSlot,
			rcfg.ScalarSlot,
		},
	}

	for addr, storages := range intrinsicStorageProofs {
		if _, existed := env.Proofs[addr.String()]; !existed {
			if proof, err := statedb.GetProof(addr); err != nil {
				log.Error("Proof for intrinstic address not available", "error", err, "address", addr)
			} else {
				wrappedProof := make([]hexutil.Bytes, len(proof))
				for i, bt := range proof {
					wrappedProof[i] = bt
				}
				env.Proofs[addr.String()] = wrappedProof
			}
		}

		if _, existed := env.StorageProofs[addr.String()]; !existed {
			env.StorageProofs[addr.String()] = make(map[string][]hexutil.Bytes)
		}

		for _, slot := range storages {
			if _, existed := env.StorageProofs[addr.String()][slot.String()]; !existed {
				if trie, err := statedb.GetStorageTrieForProof(addr); err != nil {
					log.Error("Storage proof for intrinstic address not available", "error", err, "address", addr)
				} else if proof, _ := statedb.GetSecureTrieProof(trie, slot); err != nil {
					log.Error("Get storage proof for intrinstic address failed", "error", err, "address", addr, "slot", slot)
				} else {
					wrappedProof := make([]hexutil.Bytes, len(proof))
					for i, bt := range proof {
						wrappedProof[i] = bt
					}
					env.StorageProofs[addr.String()][slot.String()] = wrappedProof
				}
			}
		}
	}

	blockTrace := &types.BlockTrace{
		ChainID: env.ChainConfig.ChainID.Uint64(),
		Version: params.ArchiveVersion(params.CommitHash),
		Coinbase: &types.AccountWrapper{
			Address:          env.Coinbase,
			Nonce:            statedb.GetNonce(env.Coinbase),
			Balance:          (*hexutil.Big)(statedb.GetBalance(env.Coinbase)),
			KeccakCodeHash:   statedb.GetKeccakCodeHash(env.Coinbase),
			PoseidonCodeHash: statedb.GetPoseidonCodeHash(env.Coinbase),
			CodeSize:         statedb.GetCodeSize(env.Coinbase),
		},
		Header:           block.Header(),
		StorageTrace:     env.StorageTrace,
		ExecutionResults: env.ExecutionResults,
		TxStorageTraces:  env.TxStorageTraces,
		Transactions:     txs,
	}

	for i, tx := range block.Transactions() {
		evmTrace := env.ExecutionResults[i]
		// probably a Contract Call
		if len(tx.Data()) != 0 && tx.To() != nil {
			evmTrace.ByteCode = hexutil.Encode(statedb.GetCode(*tx.To()))
			// Get tx.to address's code hash.
			codeHash := statedb.GetPoseidonCodeHash(*tx.To())
			evmTrace.PoseidonCodeHash = &codeHash
		} else if tx.To() == nil { // Contract is created.
			evmTrace.ByteCode = hexutil.Encode(tx.Data())
		}
	}

	// only zktrie model has the ability to get `mptwitness`.
	if env.ChainConfig.Scroll.ZktrieEnabled() {
		// we use MPTWitnessNothing by default and do not allow switch among MPTWitnessType atm.
		// MPTWitness will be removed from traces in the future.
		if err := zkproof.FillBlockTraceForMPTWitness(zkproof.MPTWitnessNothing, blockTrace); err != nil {
			log.Error("fill mpt witness fail", "error", err)
		}
	}

	blockTrace.WithdrawTrieRoot = withdrawtrie.ReadWTRSlot(rcfg.L2MessageQueueAddress, env.State)

	return blockTrace, nil
}
