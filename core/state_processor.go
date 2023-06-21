// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/consensus/misc"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/circuitcapacitychecker"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rollup/withdrawtrie"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts    types.Receipts
		usedGas     = new(uint64)
		header      = block.Header()
		blockHash   = block.Hash()
		blockNumber = block.Number()
		allLogs     []*types.Log
		gp          = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	blockContext := NewEVMBlockContext(header, p.bc, nil)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, p.config, cfg)
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		msg, err := tx.AsMessage(types.MakeSigner(p.config, header.Number), header.BaseFee)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		statedb.Prepare(tx.Hash(), i)
		receipt, err := applyTransaction(msg, p.config, p.bc, nil, gp, statedb, blockNumber, blockHash, tx, usedGas, vmenv)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles())

	return receipts, allLogs, *usedGas, nil
}

func applyTransaction(msg types.Message, config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	evm.Reset(txContext, statedb)

	// Apply the transaction to the current state (included in the env).
	result, err := ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	// Update the state with pending changes.
	var root []byte
	if config.IsByzantium(blockNumber) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(blockNumber)).Bytes()
	}
	*usedGas += result.UsedGas

	// If the result contains a revert reason, return it.
	returnVal := result.Return()
	if len(result.Revert()) > 0 {
		returnVal = result.Revert()
	}
	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas, ReturnValue: returnVal}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(tx.Hash(), blockHash)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	receipt.L1Fee = result.L1Fee
	return receipt, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number), header.BaseFee)
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, config, cfg)
	return applyTransaction(msg, config, bc, author, gp, statedb, header.Number, header.Hash(), tx, usedGas, vmenv)
}

func applyTransactionWithCircuitCheck(msg types.Message, config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, signer types.Signer, tx *types.Transaction, usedGas *uint64, evm *vm.EVM,
	tracer *vm.StructLogger, proofCaches map[string]*circuitcapacitychecker.ProofCache, checker *circuitcapacitychecker.CircuitCapacityChecker) (*types.Receipt, error) {
	// reset StructLogger to avoid OOM
	tracer.Reset()

	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	evm.Reset(txContext, statedb)

	var traceCoinbase *common.Address
	if config.Scroll.FeeVaultEnabled() {
		traceCoinbase = config.Scroll.FeeVaultAddress
	} else {
		traceCoinbase = author
	}
	from, _ := types.Sender(signer, tx)
	to := tx.To()
	sender := &types.AccountWrapper{
		Address:          from,
		Nonce:            statedb.GetNonce(from),
		Balance:          (*hexutil.Big)(statedb.GetBalance(from)),
		KeccakCodeHash:   statedb.GetKeccakCodeHash(from),
		PoseidonCodeHash: statedb.GetPoseidonCodeHash(from),
		CodeSize:         statedb.GetCodeSize(from),
	}
	var receiver *types.AccountWrapper
	if to != nil {
		receiver = &types.AccountWrapper{
			Address:          *to,
			Nonce:            statedb.GetNonce(*to),
			Balance:          (*hexutil.Big)(statedb.GetBalance(*to)),
			KeccakCodeHash:   statedb.GetKeccakCodeHash(*to),
			PoseidonCodeHash: statedb.GetPoseidonCodeHash(*to),
			CodeSize:         statedb.GetCodeSize(*to),
		}
	}

	// Apply the transaction to the current state (included in the env).
	result, err := ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	// If the result contains a revert reason, return it.
	returnVal := result.Return()
	if len(result.Revert()) > 0 {
		returnVal = result.Revert()
	}

	// currently `RootBefore` & `RootAfter` are not used
	txStorageTrace := &types.StorageTrace{
		Proofs:        make(map[string][]hexutil.Bytes),
		StorageProofs: make(map[string]map[string][]hexutil.Bytes),
	}

	proofAccounts := tracer.UpdatedAccounts()
	proofAccounts[*traceCoinbase] = struct{}{}
	proofAccounts[rcfg.L1GasPriceOracleAddress] = struct{}{}
	for addr := range proofAccounts {
		addrStr := addr.String()
		proofCache, existed := proofCaches[addrStr]
		if !existed {
			proofCache = circuitcapacitychecker.NewProofCache(statedb, addr)
			proof, err := statedb.GetProof(addr)
			if err != nil {
				log.Error("Proof not available", "address", addrStr, "error", err)
				// but we still mark the proofs map with nil array
			}
			proofCache.AccountProof = types.WrapProof(proof)
			proofCaches[addrStr] = proofCache

		}
		txStorageTrace.Proofs[addrStr] = proofCache.AccountProof
	}

	proofStorages := tracer.UpdatedStorages()
	proofStorages[rcfg.L1GasPriceOracleAddress] = vm.Storage(
		map[common.Hash]common.Hash{
			rcfg.L1BaseFeeSlot: {},
			rcfg.OverheadSlot:  {},
			rcfg.ScalarSlot:    {},
		})
	for addr, keys := range proofStorages {
		addrStr := addr.String()
		txStorageTrace.StorageProofs[addrStr] = make(map[string][]hexutil.Bytes)
		proofCache, existed := proofCaches[addrStr]
		if !existed {
			panic("any storage proof under an account must come along with account proof")
		}

		if proofCache.StorageTrie == nil {
			// we have no storage proof available (maybe the account is not existed yet),
			// just continue to next address
			log.Info("Storage trie not available", "address", addr)
			continue
		}

		for key, values := range keys {
			keyStr := key.String()
			stgProof, existed := proofCache.StorageProof[keyStr]
			if !existed {
				var proof [][]byte
				var err error
				if proofCache.TrieTracer.Available() {
					proof, err = statedb.GetSecureTrieProof(proofCache.TrieTracer, key)
				} else {
					proof, err = statedb.GetSecureTrieProof(proofCache.StorageTrie, key)
				}
				if err != nil {
					log.Error("Storage proof not available", "error", err, "address", addrStr, "key", keyStr)
					// but we still mark the proofs map with nil array
				}
				stgProof = types.WrapProof(proof)
				proofCache.StorageProof[keyStr] = stgProof
			}
			// isDelete
			if proofCache.TrieTracer.Available() && bytes.Equal(values.Bytes(), common.Hash{}.Bytes()) {
				proofCache.TrieTracer.MarkDeletion(key)
			}
			txStorageTrace.StorageProofs[addrStr][keyStr] = stgProof
		}

		// build dummy per-tx deletion proof
		if proofCache.TrieTracer.Available() {
			delProofs, err := proofCache.TrieTracer.GetDeletionProofs()
			if err != nil {
				log.Error("deletion proof failure", "error", err)
			} else {
				for _, proof := range delProofs {
					txStorageTrace.DeletionProofs = append(txStorageTrace.DeletionProofs, proof)
				}
			}
		}
	}

	createdAcc := tracer.CreatedAccount()
	if to == nil {
		if createdAcc == nil {
			return nil, errors.New("unexpected tx: address for created contract unavailable")
		}
		to = &createdAcc.Address
	}
	var after []*types.AccountWrapper
	// collect affected account after tx being applied
	for _, acc := range []common.Address{from, *to, *traceCoinbase} {
		after = append(after, &types.AccountWrapper{
			Address:          acc,
			Nonce:            statedb.GetNonce(acc),
			Balance:          (*hexutil.Big)(statedb.GetBalance(acc)),
			KeccakCodeHash:   statedb.GetKeccakCodeHash(acc),
			PoseidonCodeHash: statedb.GetPoseidonCodeHash(acc),
			CodeSize:         statedb.GetCodeSize(acc),
		})
	}

	traces := &types.BlockTrace{
		ChainID: config.ChainID.Uint64(),
		Version: params.ArchiveVersion(params.CommitHash),
		Header:  header,
		Coinbase: &types.AccountWrapper{
			Address:          *traceCoinbase,
			Nonce:            statedb.GetNonce(*traceCoinbase),
			Balance:          (*hexutil.Big)(statedb.GetBalance(*traceCoinbase)),
			KeccakCodeHash:   statedb.GetKeccakCodeHash(*traceCoinbase),
			PoseidonCodeHash: statedb.GetPoseidonCodeHash(*traceCoinbase),
			CodeSize:         statedb.GetCodeSize(*traceCoinbase),
		},
		WithdrawTrieRoot: withdrawtrie.ReadWTRSlot(rcfg.L2MessageQueueAddress, statedb),
		Transactions: []*types.TransactionData{
			types.NewTransactionData(tx, header.Number.Uint64(), config),
		},
		ExecutionResults: []*types.ExecutionResult{
			{
				From:           sender,
				To:             receiver,
				AccountCreated: createdAcc,
				AccountsAfter:  after,
				Gas:            result.UsedGas,
				Failed:         result.Failed(),
				ReturnValue:    fmt.Sprintf("%x", common.CopyBytes(returnVal)),
				StructLogs:     vm.FormatLogs(tracer.StructLogs()),
			},
		},
		StorageTrace:   txStorageTrace,
		TxStorageTrace: []*types.StorageTrace{txStorageTrace},
	}

	if result.L1Fee != nil {
		traces.ExecutionResults[0].L1Fee = result.L1Fee.Uint64()
	}

	// probably a Contract Call
	if len(tx.Data()) != 0 && tx.To() != nil {
		traces.ExecutionResults[0].ByteCode = hexutil.Encode(statedb.GetCode(*tx.To()))
		// Get tx.to address's code hash.
		codeHash := statedb.GetPoseidonCodeHash(*tx.To())
		traces.ExecutionResults[0].PoseidonCodeHash = &codeHash
	} else if tx.To() == nil { // Contract is created.
		traces.ExecutionResults[0].ByteCode = hexutil.Encode(tx.Data())
	}

	if err := checker.ApplyTransaction(traces); err != nil {
		return nil, err
	}

	// Update the state with pending changes.
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas, ReturnValue: returnVal}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(tx.Hash(), header.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = header.Hash()
	receipt.BlockNumber = header.Number
	receipt.TransactionIndex = uint(statedb.TxIndex())
	receipt.L1Fee = result.L1Fee
	return receipt, err
}

// ApplyTransactionWithCircuitCheck attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransactionWithCircuitCheck(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, signer types.Signer, tx *types.Transaction, usedGas *uint64, cfg vm.Config,
	proofCaches map[string]*circuitcapacitychecker.ProofCache, checker *circuitcapacitychecker.CircuitCapacityChecker) (*types.Receipt, error) {
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number), header.BaseFee)
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, config, cfg)
	return applyTransactionWithCircuitCheck(msg, config, bc, author, gp, statedb, header, signer, tx, usedGas, vmenv, cfg.Tracer.(*vm.StructLogger), proofCaches, checker)
}
