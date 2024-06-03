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
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rollup/fees"
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
	var (
		context = NewEVMBlockContext(header, p.bc, p.config, nil)
		vmenv   = vm.NewEVM(context, vm.TxContext{}, statedb, p.config, cfg)
		signer  = types.MakeSigner(p.config, header.Number, header.Time)
	)
	if beaconRoot := block.BeaconRoot(); beaconRoot != nil {
		ProcessBeaconBlockRoot(*beaconRoot, vmenv, statedb)
	}

	if block.Number().Uint64() == 6 {
		log.Info("signer", "block", block.Number(), "IsCurie", p.config.IsCurie(block.Number()))
		log.Info("signer", "block", block.Number(), "IsCancun", p.config.IsCancun(block.Number(), block.Time()))
		log.Info("signer", "block", block.Number(), "IsLondon", p.config.IsLondon(block.Number()))
		log.Info("signer", "block", block.Number(), "IsBerlin", p.config.IsBerlin(block.Number()))
		log.Info("signer", "block", block.Number(), "gp", gp)
	}

	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {

		if block.Number().Uint64() == 6 && i == 0 {
			params.Debug = true
		} else {
			params.Debug = false
		}

		// if params.Debug {
		// 	parent := p.bc.GetBlockByHash(block.Header().ParentHash).Header()
		// 	traceEnv, err := tracing.CreateTraceEnv(p.config, p.bc, p.engine, p.bc.db, statedb, parent,
		// 		// new block with a placeholder tx, for traceEnv's ExecutionResults length & TxStorageTraces length
		// 		types.NewBlockWithHeader(header).WithBody([]*types.Transaction{types.NewTx(&types.LegacyTx{})}, nil),
		// 		false)
		// 	if err != nil {
		// 		log.Error("failed to create traceEnv", "err", err)
		// 	}

		// 	traces, err := traceEnv.GetBlockTrace(
		// 		types.NewBlockWithHeader(block.Header()).WithBody([]*types.Transaction{tx}, nil),
		// 	)
		// 	if err != nil {
		// 		log.Error("failed to get BlockTrace", "err", err)
		// 	}

		// 	log.Info("tracing", "traces", traces)
		// }

		if params.Debug {
			log.Info("tx", "i", i, "tx.AccessList()", tx.AccessList())
			log.Info("tx", "i", i, "tx.BlobGas()", tx.BlobGas())
			log.Info("tx", "i", i, "tx.BlobGasFeeCap()", tx.BlobGasFeeCap())
			log.Info("tx", "i", i, "tx.BlobHashes()", tx.BlobHashes())
			log.Info("tx", "i", i, "tx.BlobTxSidecar()", tx.BlobTxSidecar())
			log.Info("tx", "i", i, "tx.ChainId()", tx.ChainId())
			log.Info("tx", "i", i, "tx.Cost()", tx.Cost())
			log.Info("tx", "i", i, "tx.Data()", hexutil.Encode(tx.Data()))
			log.Info("tx", "i", i, "tx.Gas()", tx.Gas())
			log.Info("tx", "i", i, "tx.GasFeeCap()", tx.GasFeeCap())
			log.Info("tx", "i", i, "tx.GasPrice()", tx.GasPrice())
			log.Info("tx", "i", i, "tx.GasTipCap()", tx.GasTipCap())
			log.Info("tx", "i", i, "tx.Hash()", tx.Hash().Hex())
			log.Info("tx", "i", i, "tx.IsL1MessageTx()", tx.IsL1MessageTx())
			log.Info("tx", "i", i, "tx.Nonce()", tx.Nonce())
			log.Info("tx", "i", i, "tx.Protected()", tx.Protected())
			log.Info("tx", "i", i, "tx.Size()", tx.Size())
			log.Info("tx", "i", i, "tx.Time()", tx.Time())
			log.Info("tx", "i", i, "tx.To()", tx.To())
			log.Info("tx", "i", i, "tx.Type()", tx.Type())
			log.Info("tx", "i", i, "tx.Value()", tx.Value())
		}

		msg, err := TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}

		if params.Debug {
			log.Info("msg", "i", i, "msg.AccessList", msg.AccessList)
			log.Info("msg", "i", i, "msg.BlobGasFeeCap", msg.BlobGasFeeCap)
			log.Info("msg", "i", i, "msg.BlobHashes", msg.BlobHashes)
			log.Info("msg", "i", i, "msg.Data", hexutil.Encode(msg.Data))
			log.Info("msg", "i", i, "msg.From", msg.From)
			log.Info("msg", "i", i, "msg.GasFeeCap", msg.GasFeeCap)
			log.Info("msg", "i", i, "msg.GasLimit", msg.GasLimit)
			log.Info("msg", "i", i, "msg.GasPrice", msg.GasPrice)
			log.Info("msg", "i", i, "msg.GasTipCap", msg.GasTipCap)
			log.Info("msg", "i", i, "msg.IsL1MessageTx", msg.IsL1MessageTx)
			log.Info("msg", "i", i, "msg.Nonce", msg.Nonce)
			log.Info("msg", "i", i, "msg.To", msg.To)
			log.Info("msg", "i", i, "msg.Value", msg.Value)
		}

		statedb.SetTxContext(tx.Hash(), i)
		receipt, err := applyTransaction(msg, p.config, gp, statedb, blockNumber, blockHash, tx, usedGas, vmenv)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("could not apply tx %d [%v]: %w", i, tx.Hash().Hex(), err)
		}

		if params.Debug {
			log.Info("receipt", "i", i, "receipt.BlobGasPrice", receipt.BlobGasPrice)
			log.Info("receipt", "i", i, "receipt.BlobGasUsed", receipt.BlobGasUsed)
			log.Info("receipt", "i", i, "receipt.BlockHash", receipt.BlockHash)
			log.Info("receipt", "i", i, "receipt.BlockNumber", receipt.BlockNumber)
			log.Info("receipt", "i", i, "receipt.Bloom", hexutil.Encode(receipt.Bloom.Bytes()))
			log.Info("receipt", "i", i, "receipt.ContractAddress", receipt.ContractAddress)
			log.Info("receipt", "i", i, "receipt.CumulativeGasUsed", receipt.CumulativeGasUsed)
			log.Info("receipt", "i", i, "receipt.EffectiveGasPrice", receipt.EffectiveGasPrice)
			log.Info("receipt", "i", i, "receipt.GasUsed", receipt.GasUsed)
			log.Info("receipt", "i", i, "receipt.L1Fee", receipt.L1Fee)
			log.Info("receipt", "i", i, "receipt.Logs", receipt.Logs)
			log.Info("receipt", "i", i, "receipt.PostState", hexutil.Encode(receipt.PostState))
			log.Info("receipt", "i", i, "receipt.Status", receipt.Status)
			log.Info("receipt", "i", i, "receipt.TransactionIndex", receipt.TransactionIndex)
			log.Info("receipt", "i", i, "receipt.TxHash", receipt.TxHash.Hex())
			log.Info("receipt", "i", i, "receipt.Type", receipt.Type)
		}

		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Fail if Shanghai not enabled and len(withdrawals) is non-zero.
	withdrawals := block.Withdrawals()
	if len(withdrawals) > 0 && !p.config.IsShanghai(block.Number(), block.Time()) {
		return nil, nil, 0, errors.New("withdrawals before shanghai")
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles(), withdrawals)

	return receipts, allLogs, *usedGas, nil
}

func applyTransaction(msg *Message, config *params.ChainConfig, gp *GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment.
	txContext := NewEVMTxContext(msg)
	evm.Reset(txContext, statedb)

	l1DataFee, err := fees.CalculateL1DataFee(tx, statedb)
	if err != nil {
		return nil, err
	}

	if params.Debug {
		log.Info("applyTransaction", "tx", tx.Hash().Hex(), "l1DataFee", l1DataFee)
	}

	// Apply the transaction to the current state (included in the env).
	result, err := ApplyMessage(evm, msg, gp, l1DataFee)
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

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	if tx.Type() == types.BlobTxType {
		receipt.BlobGasUsed = uint64(len(tx.BlobHashes()) * params.BlobTxBlobGasPerBlob)
		receipt.BlobGasPrice = evm.Context.BlobBaseFee
	}

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(tx.Hash(), blockNumber.Uint64(), blockHash)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	receipt.L1Fee = result.L1DataFee
	return receipt, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	msg, err := TransactionToMessage(tx, types.MakeSigner(config, header.Number, header.Time), header.BaseFee)
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, config, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{BlobHashes: tx.BlobHashes()}, statedb, config, cfg)
	return applyTransaction(msg, config, gp, statedb, header.Number, header.Hash(), tx, usedGas, vmenv)
}

// ProcessBeaconBlockRoot applies the EIP-4788 system call to the beacon block root
// contract. This method is exported to be used in tests.
func ProcessBeaconBlockRoot(beaconRoot common.Hash, vmenv *vm.EVM, statedb *state.StateDB) {
	// If EIP-4788 is enabled, we need to invoke the beaconroot storage contract with
	// the new root
	msg := &Message{
		From:      params.SystemAddress,
		GasLimit:  30_000_000,
		GasPrice:  common.Big0,
		GasFeeCap: common.Big0,
		GasTipCap: common.Big0,
		To:        &params.BeaconRootsStorageAddress,
		Data:      beaconRoot[:],
	}
	vmenv.Reset(NewEVMTxContext(msg), statedb)
	statedb.AddAddressToAccessList(params.BeaconRootsStorageAddress)
	_, _, _ = vmenv.Call(vm.AccountRef(msg.From), *msg.To, msg.Data, 30_000_000, common.Big0)
	statedb.Finalise(true)
}
