package tracing

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/crypto/codehash"
	"github.com/scroll-tech/go-ethereum/eth/tracers"
	"github.com/scroll-tech/go-ethereum/eth/tracers/logger"
	"github.com/scroll-tech/go-ethereum/eth/tracers/native"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/fees"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rollup/withdrawtrie"
	"github.com/scroll-tech/go-ethereum/trie"
	"github.com/scroll-tech/go-ethereum/trie/zkproof"
)

var (
	getTxResultTimer             = metrics.NewRegisteredTimer("rollup/tracing/get_tx_result", nil)
	getTxResultApplyMessageTimer = metrics.NewRegisteredTimer("rollup/tracing/get_tx_result/apply_message", nil)
	getTxResultZkTrieBuildTimer  = metrics.NewRegisteredTimer("rollup/tracing/get_tx_result/zk_trie_build", nil)
	getTxResultTracerResultTimer = metrics.NewRegisteredTimer("rollup/tracing/get_tx_result/tracer_result", nil)
	feedTxToTracerTimer          = metrics.NewRegisteredTimer("rollup/tracing/feed_tx_to_tracer", nil)
	fillBlockTraceTimer          = metrics.NewRegisteredTimer("rollup/tracing/fill_block_trace", nil)
)

// TracerWrapper implements ScrollTracerWrapper interface
type TracerWrapper struct{}

// TracerWrapper creates a new TracerWrapper
func NewTracerWrapper() *TracerWrapper {
	return &TracerWrapper{}
}

// CreateTraceEnvAndGetBlockTrace wraps the whole block tracing logic for a block
func (tw *TracerWrapper) CreateTraceEnvAndGetBlockTrace(chainConfig *params.ChainConfig, chainContext core.ChainContext, engine consensus.Engine, chaindb ethdb.Database, statedb *state.StateDB, parentHeader *types.Header, block *types.Block, finaliseStateAfterApply bool) (*types.BlockTrace, error) {
	traceEnv, err := CreateTraceEnv(chainConfig, chainContext, engine, chaindb, statedb, parentHeader, block, finaliseStateAfterApply)
	if err != nil {
		return nil, err
	}

	return traceEnv.GetBlockTrace(block)
}

type TraceEnv struct {
	logConfig               *logger.Config
	finaliseStateAfterApply bool
	chainConfig             *params.ChainConfig

	coinbase common.Address

	signer   types.Signer
	state    *state.StateDB
	blockCtx vm.BlockContext

	// The following Mutexes are used to protect against parallel read/write,
	// since txs are executed in parallel.
	pMu sync.Mutex // for `TraceEnv.StorageTrace.Proofs`
	sMu sync.Mutex // for `TraceEnv.state`
	cMu sync.Mutex // for `TraceEnv.Codes`

	ExecutionResults []*types.ExecutionResult
	*types.StorageTrace
	TxStorageTraces []*types.StorageTrace
	Codes           map[common.Hash]logger.CodeInfo
	// zktrie tracer is used for zktrie storage to build additional deletion proof
	ZkTrieTracer map[string]*ProofTracer

	// StartL1QueueIndex is the next L1 message queue index that this block can process.
	// Example: If the parent block included QueueIndex=9, then StartL1QueueIndex will
	// be 10.
	StartL1QueueIndex uint64
}

// Context is the same as Context in eth/tracers/tracers.go
type Context struct {
	BlockHash common.Hash
	TxIndex   int
	TxHash    common.Hash
}

// txTraceTask is the same as txTraceTask in eth/tracers/api.go
type txTraceTask struct {
	statedb *state.StateDB
	index   int
}

func CreateTraceEnvHelper(chainConfig *params.ChainConfig, logConfig *logger.Config, blockCtx vm.BlockContext, startL1QueueIndex uint64, coinbase common.Address, statedb *state.StateDB, rootBefore common.Hash, block *types.Block, finaliseStateAfterApply bool) *TraceEnv {
	return &TraceEnv{
		logConfig:               logConfig,
		finaliseStateAfterApply: finaliseStateAfterApply,
		chainConfig:             chainConfig,
		coinbase:                coinbase,
		signer:                  types.MakeSigner(chainConfig, block.Number(), block.Time()),
		state:                   statedb,
		blockCtx:                blockCtx,
		ExecutionResults:        make([]*types.ExecutionResult, block.Transactions().Len()),
		StorageTrace: &types.StorageTrace{
			RootBefore:    rootBefore,
			RootAfter:     block.Root(),
			Proofs:        make(map[string][]hexutil.Bytes),
			StorageProofs: make(map[string]map[string][]hexutil.Bytes),
		},
		TxStorageTraces:   make([]*types.StorageTrace, block.Transactions().Len()),
		Codes:             make(map[common.Hash]logger.CodeInfo),
		ZkTrieTracer:      make(map[string]*ProofTracer),
		StartL1QueueIndex: startL1QueueIndex,
	}
}

func CreateTraceEnv(chainConfig *params.ChainConfig, chainContext core.ChainContext, engine consensus.Engine, chaindb ethdb.Database, statedb *state.StateDB, parentHeader *types.Header, block *types.Block, finaliseStateAfterApply bool) (*TraceEnv, error) {
	var coinbase common.Address

	var err error
	if chainConfig.Scroll.FeeVaultEnabled() {
		coinbase = *chainConfig.Scroll.FeeVaultAddress
	} else {
		coinbase, err = engine.Author(block.Header())
		if err != nil {
			log.Warn("recover coinbase in CreateTraceEnv fail. using zero-address", "err", err, "blockNumber", block.Header().Number, "headerHash", block.Header().Hash())
		}
	}

	// Collect start queue index, we should always have this value for blocks
	// that have been executed.
	// FIXME: This value will be incorrect on the signer, since we reuse this
	// DB entry to signal which index the worker should continue from.
	// Example: Ledger A <-- B <-- C. Block `A` contains up to `QueueIndex=9`.
	// For block `B`, the worker skips 10 messages and includes 0.
	// `ReadFirstQueueIndexNotInL2Block(B)` will then return `20` on the
	// signer to avoid re-processing the same 10 transactions again for
	// block `C`.
	// `ReadFirstQueueIndexNotInL1Block(B)` will return the correct value
	// `10` on follower nodes.
	startL1QueueIndex := rawdb.ReadFirstQueueIndexNotInL2Block(chaindb, parentHeader.Hash())
	if startL1QueueIndex == nil {
		log.Error("missing FirstQueueIndexNotInL2Block for block during trace call", "number", parentHeader.Number.Uint64(), "hash", parentHeader.Hash())
		return nil, fmt.Errorf("missing FirstQueueIndexNotInL2Block for block during trace call: hash=%v, parentHash=%vv", block.Hash(), parentHeader.Hash())
	}
	env := CreateTraceEnvHelper(
		chainConfig,
		&logger.Config{
			DisableStorage:   true,
			DisableStack:     true,
			EnableMemory:     false,
			EnableReturnData: true,
			Debug:            true,
		},
		core.NewEVMBlockContext(block.Header(), chainContext, chainConfig, nil),
		*startL1QueueIndex,
		coinbase,
		statedb,
		parentHeader.Root,
		block,
		finaliseStateAfterApply,
	)

	key := coinbase.String()
	if _, exist := env.Proofs[key]; !exist {
		proof, err := env.state.GetProof(coinbase)
		if err != nil {
			log.Error("Proof for coinbase not available", "coinbase", coinbase, "error", err)
			// but we still mark the proofs map with nil array
		}
		env.Proofs[key] = types.WrapProof(proof)
	}

	return env, nil
}

func (env *TraceEnv) GetBlockTrace(block *types.Block) (*types.BlockTrace, error) {
	if env == nil {
		log.Warn("running in light mode? trace env is nil and do not support `GetBlockTrace`")
		return nil, nil
	}

	// Execute all the transaction contained within the block concurrently
	var (
		txs   = block.Transactions()
		pend  = new(sync.WaitGroup)
		jobs  = make(chan *txTraceTask, len(txs))
		errCh = make(chan error, 1)
	)
	threads := runtime.NumCPU()
	if threads > len(txs) {
		threads = len(txs)
	}
	for th := 0; th < threads; th++ {
		pend.Add(1)
		go func() {
			defer func(t time.Time) {
				pend.Done()
				getTxResultTimer.Update(time.Since(t))
			}(time.Now())

			// Fetch and execute the next transaction trace tasks
			for task := range jobs {
				if err := env.getTxResult(task.statedb, task.index, block); err != nil {
					select {
					case errCh <- err:
					default:
					}
					log.Error(
						"failed to trace tx",
						"txHash", txs[task.index].Hash().String(),
						"blockHash", block.Hash().String(),
						"blockNumber", block.NumberU64(),
						"err", err,
					)
				}
			}
		}()
	}

	// Feed the transactions into the tracers and return
	var failed error
	common.WithTimer(feedTxToTracerTimer, func() {
		for i, tx := range txs {
			// Send the trace task over for execution
			jobs <- &txTraceTask{statedb: env.state.Copy(), index: i}

			// Generate the next state snapshot fast without tracing
			msg, _ := core.TransactionToMessage(tx, env.signer, block.BaseFee())
			env.state.SetTxContext(tx.Hash(), i)
			vmenv := vm.NewEVM(env.blockCtx, core.NewEVMTxContext(msg), env.state, env.chainConfig, vm.Config{})
			l1DataFee, err := fees.CalculateL1DataFee(tx, env.state, env.chainConfig, block.Number())
			if err != nil {
				failed = err
				break
			}
			if _, err = core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(msg.GasLimit), l1DataFee); err != nil {
				failed = err
				break
			}
			if env.finaliseStateAfterApply {
				env.state.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
			}
		}
	})
	close(jobs)
	pend.Wait()

	// after all tx has been traced, collect "deletion proof" for zktrie
	for _, tracer := range env.ZkTrieTracer {
		delProofs, err := tracer.GetDeletionProofs()
		if err != nil {
			log.Error("deletion proof failure", "error", err)
		} else {
			for _, proof := range delProofs {
				env.DeletionProofs = append(env.DeletionProofs, proof)
			}
		}
	}

	// build dummy per-tx deletion proof
	for _, txStorageTrace := range env.TxStorageTraces {
		if txStorageTrace != nil {
			txStorageTrace.DeletionProofs = env.DeletionProofs
		}
	}

	// If execution failed in between, abort
	select {
	case err := <-errCh:
		return nil, err
	default:
		if failed != nil {
			return nil, failed
		}
	}

	return env.fillBlockTrace(block)
}

func (env *TraceEnv) getTxResult(state *state.StateDB, index int, block *types.Block) error {
	tx := block.Transactions()[index]
	msg, _ := core.TransactionToMessage(tx, env.signer, block.BaseFee())
	from, _ := types.Sender(env.signer, tx)
	to := tx.To()

	txctx := &Context{
		BlockHash: block.TxHash(),
		TxIndex:   index,
		TxHash:    tx.Hash(),
	}

	sender := &types.AccountWrapper{
		Address:          from,
		Nonce:            state.GetNonce(from),
		Balance:          (*hexutil.Big)(state.GetBalance(from)),
		KeccakCodeHash:   state.GetKeccakCodeHash(from),
		PoseidonCodeHash: state.GetPoseidonCodeHash(from),
		CodeSize:         state.GetCodeSize(from),
	}
	var receiver *types.AccountWrapper
	if to != nil {
		receiver = &types.AccountWrapper{
			Address:          *to,
			Nonce:            state.GetNonce(*to),
			Balance:          (*hexutil.Big)(state.GetBalance(*to)),
			KeccakCodeHash:   state.GetKeccakCodeHash(*to),
			PoseidonCodeHash: state.GetPoseidonCodeHash(*to),
			CodeSize:         state.GetCodeSize(*to),
		}
	}

	applyMessageStart := time.Now()
	structLogger := logger.NewStructLogger(env.logConfig)
	tracerContext := tracers.Context{
		BlockHash: block.Hash(),
		TxIndex:   index,
		TxHash:    tx.Hash(),
	}
	callTracerConfig := native.CallTracerConfig{
		OnlyTopCall: false,
		WithLog:     true,
	}
	callTracer, err := native.NewCallTracerWithConfig(&tracerContext, callTracerConfig)
	if err != nil {
		return fmt.Errorf("failed to create callTracer: %w", err)
	}
	tracer := &native.MuxTracer{}
	tracer.Append("structLogger", structLogger)
	tracer.Append("callTracer", callTracer)

	// Run the transaction with tracing enabled.
	vmenv := vm.NewEVM(env.blockCtx, core.NewEVMTxContext(msg), state, env.chainConfig, vm.Config{Tracer: tracer, NoBaseFee: true})

	state.SetTxContext(txctx.TxHash, txctx.TxIndex)

	// Computes the new state by applying the given message.
	l1DataFee, err := fees.CalculateL1DataFee(tx, state, env.chainConfig, block.Number())
	if err != nil {
		return err
	}
	result, err := core.ApplyMessage(vmenv, msg, new(core.GasPool).AddGas(msg.GasLimit), l1DataFee)
	if err != nil {
		getTxResultApplyMessageTimer.UpdateSince(applyMessageStart)
		return err
	}
	getTxResultApplyMessageTimer.UpdateSince(applyMessageStart)
	// If the result contains a revert reason, return it.
	returnVal := result.Return()
	if len(result.Revert()) > 0 {
		returnVal = result.Revert()
	}

	createdAcc := structLogger.CreatedAccount()
	var after []*types.AccountWrapper
	if to == nil {
		if createdAcc == nil {
			return errors.New("unexpected tx: address for created contract unavailable")
		}
		to = &createdAcc.Address
	}
	// collect affected account after tx being applied
	for _, acc := range []common.Address{from, *to, env.coinbase} {
		after = append(after, &types.AccountWrapper{
			Address:          acc,
			Nonce:            state.GetNonce(acc),
			Balance:          (*hexutil.Big)(state.GetBalance(acc)),
			KeccakCodeHash:   state.GetKeccakCodeHash(acc),
			PoseidonCodeHash: state.GetPoseidonCodeHash(acc),
			CodeSize:         state.GetCodeSize(acc),
		})
	}

	txStorageTrace := &types.StorageTrace{
		Proofs:        make(map[string][]hexutil.Bytes),
		StorageProofs: make(map[string]map[string][]hexutil.Bytes),
	}
	// still we have no state root for per tx, only set the head and tail
	if index == 0 {
		txStorageTrace.RootBefore = state.GetRootHash()
	}
	if index == len(block.Transactions())-1 {
		txStorageTrace.RootAfter = block.Root()
	}

	// merge bytecodes
	env.cMu.Lock()
	for codeHash, codeInfo := range structLogger.TracedBytecodes() {
		if codeHash != (common.Hash{}) {
			env.Codes[codeHash] = codeInfo
		}
	}
	env.cMu.Unlock()

	// merge required proof data
	proofAccounts := structLogger.UpdatedAccounts()
	proofAccounts[vmenv.FeeRecipient()] = struct{}{}
	for addr := range proofAccounts {
		addrStr := addr.String()

		env.pMu.Lock()
		checkedProof, existed := env.Proofs[addrStr]
		if existed {
			txStorageTrace.Proofs[addrStr] = checkedProof
		}
		env.pMu.Unlock()
		if existed {
			continue
		}
		proof, err := state.GetProof(addr)
		if err != nil {
			log.Error("Proof not available", "address", addrStr, "error", err)
			// but we still mark the proofs map with nil array
		}
		wrappedProof := types.WrapProof(proof)
		env.pMu.Lock()
		env.Proofs[addrStr] = wrappedProof
		txStorageTrace.Proofs[addrStr] = wrappedProof
		env.pMu.Unlock()
	}

	zkTrieBuildStart := time.Now()
	proofStorages := structLogger.UpdatedStorages()
	for addr, keys := range proofStorages {
		if _, existed := txStorageTrace.StorageProofs[addr.String()]; !existed {
			txStorageTrace.StorageProofs[addr.String()] = make(map[string][]hexutil.Bytes)
		}

		env.sMu.Lock()
		storageTrie, err := state.Database().OpenStorageTrie(state.GetRootHash(), addr, state.GetOrNewStateObject(addr).Root())
		zkStorageTrie, isZk := storageTrie.(*trie.ZkTrie)
		if err != nil || !isZk {
			// but we still continue to next address
			log.Error("Storage trie not available", "error", err, "address", addr)
			env.sMu.Unlock()
			continue
		}
		zktrieTracer := NewProofTracer(zkStorageTrie)
		env.sMu.Unlock()

		for key := range keys {
			addrStr := addr.String()
			keyStr := key.String()
			value := state.GetState(addr, key)
			isDelete := bytes.Equal(value.Bytes(), common.Hash{}.Bytes())

			txm := txStorageTrace.StorageProofs[addrStr]
			env.sMu.Lock()
			m, existed := env.StorageProofs[addrStr]
			if !existed {
				m = make(map[string][]hexutil.Bytes)
				env.StorageProofs[addrStr] = m
			}
			if _, exists := env.ZkTrieTracer[addrStr]; !exists {
				env.ZkTrieTracer[addrStr] = zktrieTracer
			}

			if proof, existed := m[keyStr]; existed {
				txm[keyStr] = proof
				// still need to touch tracer for deletion
				if isDelete {
					env.ZkTrieTracer[addrStr].MarkDeletion(key.Bytes())
				}
				env.sMu.Unlock()
				continue
			}
			env.sMu.Unlock()

			var proof zkproof.ProofList
			if err = zkStorageTrie.Prove(key.Bytes(), &proof); err != nil {
				log.Error("Storage proof not available", "error", err, "address", addrStr, "key", keyStr)
				// but we still mark the proofs map with nil array
			}
			wrappedProof := types.WrapProof(proof)
			env.sMu.Lock()
			txm[keyStr] = wrappedProof
			m[keyStr] = wrappedProof
			if isDelete {
				zktrieTracer.MarkDeletion(key.Bytes())
			}
			env.ZkTrieTracer[addrStr].Merge(zktrieTracer)
			env.sMu.Unlock()
		}
	}
	getTxResultZkTrieBuildTimer.UpdateSince(zkTrieBuildStart)

	tracerResultTimer := time.Now()
	callTrace, err := callTracer.GetResult()
	if err != nil {
		return fmt.Errorf("failed to get callTracer result: %w", err)
	}
	getTxResultTracerResultTimer.UpdateSince(tracerResultTimer)

	env.ExecutionResults[index] = &types.ExecutionResult{
		From:           sender,
		To:             receiver,
		AccountCreated: createdAcc,
		AccountsAfter:  after,
		L1DataFee:      (*hexutil.Big)(result.L1DataFee),
		Gas:            result.UsedGas,
		Failed:         result.Failed(),
		ReturnValue:    fmt.Sprintf("%x", returnVal),
		StructLogs:     logger.FormatLogs(structLogger.StructLogs()),
		CallTrace:      callTrace,
	}
	env.TxStorageTraces[index] = txStorageTrace

	return nil
}

// fillBlockTrace content after all the txs are finished running.
func (env *TraceEnv) fillBlockTrace(block *types.Block) (*types.BlockTrace, error) {
	defer func(t time.Time) {
		fillBlockTraceTimer.Update(time.Since(t))
	}(time.Now())

	statedb := env.state

	txs := make([]*types.TransactionData, block.Transactions().Len())
	for i, tx := range block.Transactions() {
		txs[i] = types.NewTransactionData(tx, block.NumberU64(), block.Time(), env.chainConfig)
	}

	intrinsicStorageProofs := map[common.Address][]common.Hash{
		rcfg.L2MessageQueueAddress: {rcfg.WithdrawTrieRootSlot},
		rcfg.L1GasPriceOracleAddress: {
			rcfg.L1BaseFeeSlot,
			rcfg.OverheadSlot,
			rcfg.ScalarSlot,
			rcfg.L1BlobBaseFeeSlot,
			rcfg.CommitScalarSlot,
			rcfg.BlobScalarSlot,
			rcfg.IsCurieSlot,
		},
	}

	for addr, storages := range intrinsicStorageProofs {
		if _, existed := env.Proofs[addr.String()]; !existed {
			if proof, err := statedb.GetProof(addr); err != nil {
				log.Error("Proof for intrinstic address not available", "error", err, "address", addr)
			} else {
				env.Proofs[addr.String()] = types.WrapProof(proof)
			}
		}

		if _, existed := env.StorageProofs[addr.String()]; !existed {
			env.StorageProofs[addr.String()] = make(map[string][]hexutil.Bytes)
		}

		for _, slot := range storages {
			if _, existed := env.StorageProofs[addr.String()][slot.String()]; !existed {
				var proof zkproof.ProofList
				storageTrie, err := statedb.Database().OpenStorageTrie(statedb.GetRootHash(), addr, statedb.GetOrNewStateObject(addr).Root())
				zkStorageTrie, isZk := storageTrie.(*trie.ZkTrie)
				if err != nil || !isZk {
					// but we still continue to next address
					log.Error("Storage trie not available", "error", err, "address", addr)
				} else if err := zkStorageTrie.Prove(slot.Bytes(), &proof); err != nil {
					log.Error("Get storage proof for intrinstic address failed", "error", err, "address", addr, "slot", slot)
				} else {
					env.StorageProofs[addr.String()][slot.String()] = types.WrapProof(proof)
				}
			}
		}
	}

	var chainID uint64
	if env.chainConfig.ChainID != nil {
		chainID = env.chainConfig.ChainID.Uint64()
	}
	blockTrace := &types.BlockTrace{
		ChainID: chainID,
		Version: params.ArchiveVersion(params.CommitHash),
		Coinbase: &types.AccountWrapper{
			Address:          env.coinbase,
			Nonce:            statedb.GetNonce(env.coinbase),
			Balance:          (*hexutil.Big)(statedb.GetBalance(env.coinbase)),
			KeccakCodeHash:   statedb.GetKeccakCodeHash(env.coinbase),
			PoseidonCodeHash: statedb.GetPoseidonCodeHash(env.coinbase),
			CodeSize:         statedb.GetCodeSize(env.coinbase),
		},
		Header:            block.Header(),
		ExecutionResults:  env.ExecutionResults,
		StorageTrace:      env.StorageTrace,
		TxStorageTraces:   env.TxStorageTraces,
		Bytecodes:         make([]*types.BytecodeTrace, 0, len(env.Codes)),
		Transactions:      txs,
		StartL1QueueIndex: env.StartL1QueueIndex,
	}

	blockTrace.Bytecodes = append(blockTrace.Bytecodes, &types.BytecodeTrace{
		CodeSize:         0,
		KeccakCodeHash:   codehash.EmptyKeccakCodeHash,
		PoseidonCodeHash: codehash.EmptyPoseidonCodeHash,
		Code:             hexutil.Bytes{},
	})
	for _, codeInfo := range env.Codes {
		blockTrace.Bytecodes = append(blockTrace.Bytecodes, &types.BytecodeTrace{
			CodeSize:         codeInfo.CodeSize,
			KeccakCodeHash:   codeInfo.KeccakCodeHash,
			PoseidonCodeHash: codeInfo.PoseidonCodeHash,
			Code:             codeInfo.Code,
		})
	}

	blockTrace.WithdrawTrieRoot = withdrawtrie.ReadWTRSlot(rcfg.L2MessageQueueAddress, env.state)

	return blockTrace, nil
}
