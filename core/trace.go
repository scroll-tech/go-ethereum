package core

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/fees"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rollup/withdrawtrie"
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

func (env *TraceEnv) GetBlockTrace(block *types.Block) (*types.BlockTrace, error) {
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
			defer pend.Done()
			// Fetch and execute the next transaction trace tasks
			for task := range jobs {
				if err := env.getTxResult(task.statedb, task.index, block); err != nil {
					select {
					case errCh <- err:
					default:
					}
					log.Error("failed to trace tx", "txHash", txs[task.index].Hash().String())
				}
			}
		}()
	}

	// Feed the transactions into the tracers and return
	var failed error
	for i, tx := range txs {
		// Send the trace task over for execution
		jobs <- &txTraceTask{statedb: env.State.Copy(), index: i}

		// Generate the next state snapshot fast without tracing
		msg, _ := tx.AsMessage(env.Signer, block.BaseFee())
		env.State.Prepare(tx.Hash(), i)
		vmenv := vm.NewEVM(env.BlockCtx, NewEVMTxContext(msg), env.State, env.ChainConfig, vm.Config{})
		l1DataFee, err := fees.CalculateL1DataFee(tx, env.State)
		if err != nil {
			failed = err
			break
		}
		if _, err = ApplyMessage(vmenv, msg, new(GasPool).AddGas(msg.Gas()), l1DataFee); err != nil {
			failed = err
			break
		}
		// Finalize the state so any modifications are written to the trie
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		env.State.Finalise(vmenv.ChainConfig().IsEIP158(block.Number()))
	}
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
	msg, _ := tx.AsMessage(env.Signer, block.BaseFee())
	from, _ := types.Sender(env.Signer, tx)
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

	tracer := vm.NewStructLogger(env.LogConfig)
	// Run the transaction with tracing enabled.
	vmenv := vm.NewEVM(env.BlockCtx, NewEVMTxContext(msg), state, env.ChainConfig, vm.Config{Debug: true, Tracer: tracer, NoBaseFee: true})

	// Call Prepare to clear out the statedb access list
	state.Prepare(txctx.TxHash, txctx.TxIndex)

	// Computes the new state by applying the given message.
	l1DataFee, err := fees.CalculateL1DataFee(tx, state)
	if err != nil {
		return fmt.Errorf("tracing failed: %w", err)
	}
	result, err := ApplyMessage(vmenv, msg, new(GasPool).AddGas(msg.Gas()), l1DataFee)
	if err != nil {
		return fmt.Errorf("tracing failed: %w", err)
	}
	// If the result contains a revert reason, return it.
	returnVal := result.Return()
	if len(result.Revert()) > 0 {
		returnVal = result.Revert()
	}

	createdAcc := tracer.CreatedAccount()
	var after []*types.AccountWrapper
	if to == nil {
		if createdAcc == nil {
			return errors.New("unexpected tx: address for created contract unavailable")
		}
		to = &createdAcc.Address
	}
	// collect affected account after tx being applied
	for _, acc := range []common.Address{from, *to, env.Coinbase} {
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
	} else if index == len(block.Transactions())-1 {
		txStorageTrace.RootAfter = block.Root()
	}

	// merge required proof data
	proofAccounts := tracer.UpdatedAccounts()
	proofAccounts[vmenv.FeeRecipient()] = struct{}{}
	for addr := range proofAccounts {
		addrStr := addr.String()

		env.PMu.Lock()
		checkedProof, existed := env.Proofs[addrStr]
		if existed {
			txStorageTrace.Proofs[addrStr] = checkedProof
		}
		env.PMu.Unlock()
		if existed {
			continue
		}
		proof, err := state.GetProof(addr)
		if err != nil {
			log.Error("Proof not available", "address", addrStr, "error", err)
			// but we still mark the proofs map with nil array
		}
		wrappedProof := make([]hexutil.Bytes, len(proof))
		for i, bt := range proof {
			wrappedProof[i] = bt
		}
		env.PMu.Lock()
		env.Proofs[addrStr] = wrappedProof
		txStorageTrace.Proofs[addrStr] = wrappedProof
		env.PMu.Unlock()
	}

	proofStorages := tracer.UpdatedStorages()
	for addr, keys := range proofStorages {
		if _, existed := txStorageTrace.StorageProofs[addr.String()]; !existed {
			txStorageTrace.StorageProofs[addr.String()] = make(map[string][]hexutil.Bytes)
		}

		env.SMu.Lock()
		trie, err := state.GetStorageTrieForProof(addr)
		if err != nil {
			// but we still continue to next address
			log.Error("Storage trie not available", "error", err, "address", addr)
			env.SMu.Unlock()
			continue
		}
		zktrieTracer := state.NewProofTracer(trie)
		env.SMu.Unlock()

		for key, values := range keys {
			addrStr := addr.String()
			keyStr := key.String()
			isDelete := bytes.Equal(values.Bytes(), common.Hash{}.Bytes())

			txm := txStorageTrace.StorageProofs[addrStr]
			env.SMu.Lock()
			m, existed := env.StorageProofs[addrStr]
			if !existed {
				m = make(map[string][]hexutil.Bytes)
				env.StorageProofs[addrStr] = m
				if zktrieTracer.Available() {
					env.ZkTrieTracer[addrStr] = state.NewProofTracer(trie)
				}
			} else if proof, existed := m[keyStr]; existed {
				txm[keyStr] = proof
				// still need to touch tracer for deletion
				if isDelete && zktrieTracer.Available() {
					env.ZkTrieTracer[addrStr].MarkDeletion(key)
				}
				env.SMu.Unlock()
				continue
			}
			env.SMu.Unlock()

			var proof [][]byte
			var err error
			if zktrieTracer.Available() {
				proof, err = state.GetSecureTrieProof(zktrieTracer, key)
			} else {
				proof, err = state.GetSecureTrieProof(trie, key)
			}
			if err != nil {
				log.Error("Storage proof not available", "error", err, "address", addrStr, "key", keyStr)
				// but we still mark the proofs map with nil array
			}
			wrappedProof := make([]hexutil.Bytes, len(proof))
			for i, bt := range proof {
				wrappedProof[i] = bt
			}
			env.SMu.Lock()
			txm[keyStr] = wrappedProof
			m[keyStr] = wrappedProof
			if zktrieTracer.Available() {
				if isDelete {
					zktrieTracer.MarkDeletion(key)
				}
				env.ZkTrieTracer[addrStr].Merge(zktrieTracer)
			}
			env.SMu.Unlock()
		}
	}

	env.ExecutionResults[index] = &types.ExecutionResult{
		From:           sender,
		To:             receiver,
		AccountCreated: createdAcc,
		AccountsAfter:  after,
		L1DataFee:      (*hexutil.Big)(result.L1DataFee),
		Gas:            result.UsedGas,
		Failed:         result.Failed(),
		ReturnValue:    fmt.Sprintf("%x", returnVal),
		StructLogs:     vm.FormatLogs(tracer.StructLogs()),
	}
	env.TxStorageTraces[index] = txStorageTrace

	return nil
}

// fillBlockTrace content after all the txs are finished running.
func (env *TraceEnv) fillBlockTrace(block *types.Block) (*types.BlockTrace, error) {
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
