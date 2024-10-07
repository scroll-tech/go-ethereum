package l1

import (
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// scrollChainMetaData contains ABI of the ScrollChain contract.
var ScrollChainMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\": false,\"inputs\": [{\"indexed\": true,\"internalType\": \"uint256\",\"name\": \"batchIndex\",\"type\": \"uint256\"},{\"indexed\": true,\"internalType\": \"bytes32\",\"name\": \"batchHash\",\"type\": \"bytes32\"}],\"name\": \"CommitBatch\",\"type\": \"event\"},{\"anonymous\": false,\"inputs\": [{\"indexed\": true,\"internalType\": \"uint256\",\"name\": \"batchIndex\",\"type\": \"uint256\"},{\"indexed\": true,\"internalType\": \"bytes32\",\"name\": \"batchHash\",\"type\": \"bytes32\"},{\"indexed\": false,\"internalType\": \"bytes32\",\"name\": \"stateRoot\",\"type\": \"bytes32\"},{\"indexed\": false,\"internalType\": \"bytes32\",\"name\": \"withdrawRoot\",\"type\": \"bytes32\"}],\"name\": \"FinalizeBatch\",\"type\": \"event\"},{\"anonymous\": false,\"inputs\": [{\"indexed\": true,\"internalType\": \"uint256\",\"name\": \"batchIndex\",\"type\": \"uint256\"},{\"indexed\": true,\"internalType\": \"bytes32\",\"name\": \"batchHash\",\"type\": \"bytes32\"}],\"name\": \"RevertBatch\",\"type\": \"event\"},{\"anonymous\": false,\"inputs\": [{\"indexed\": false,\"internalType\": \"uint256\",\"name\": \"oldMaxNumTxInChunk\",\"type\": \"uint256\"},{\"indexed\": false,\"internalType\": \"uint256\",\"name\": \"newMaxNumTxInChunk\",\"type\": \"uint256\"}],\"name\": \"UpdateMaxNumTxInChunk\",\"type\": \"event\"},{\"anonymous\": false,\"inputs\": [{\"indexed\": true,\"internalType\": \"address\",\"name\": \"account\",\"type\": \"address\"},{\"indexed\": false,\"internalType\": \"bool\",\"name\": \"status\",\"type\": \"bool\"}],\"name\": \"UpdateProver\",\"type\": \"event\"},{\"anonymous\": false,\"inputs\": [{\"indexed\": true,\"internalType\": \"address\",\"name\": \"account\",\"type\": \"address\"},{\"indexed\": false,\"internalType\": \"bool\",\"name\": \"status\",\"type\": \"bool\"}],\"name\": \"UpdateSequencer\",\"type\": \"event\"},{\"inputs\": [{\"internalType\": \"uint8\",\"name\": \"version\",\"type\": \"uint8\"},{\"internalType\": \"bytes\",\"name\": \"parentBatchHeader\",\"type\": \"bytes\"},{\"internalType\": \"bytes[]\",\"name\": \"chunks\",\"type\": \"bytes[]\"},{\"internalType\": \"bytes\",\"name\": \"skippedL1MessageBitmap\",\"type\": \"bytes\"}],\"name\": \"commitBatch\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"uint8\",\"name\": \"version\",\"type\": \"uint8\"},{\"internalType\": \"bytes\",\"name\": \"parentBatchHeader\",\"type\": \"bytes\"},{\"internalType\": \"bytes[]\",\"name\": \"chunks\",\"type\": \"bytes[]\"},{\"internalType\": \"bytes\",\"name\": \"skippedL1MessageBitmap\",\"type\": \"bytes\"},{\"internalType\": \"bytes\",\"name\": \"blobDataProof\",\"type\": \"bytes\"}],\"name\": \"commitBatchWithBlobProof\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"uint256\",\"name\": \"batchIndex\",\"type\": \"uint256\"}],\"name\": \"committedBatches\",\"outputs\": [{\"internalType\": \"bytes32\",\"name\": \"\",\"type\": \"bytes32\"}],\"stateMutability\": \"view\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"bytes\",\"name\": \"batchHeader\",\"type\": \"bytes\"},{\"internalType\": \"bytes32\",\"name\": \"prevStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"postStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"withdrawRoot\",\"type\": \"bytes32\"}],\"name\": \"finalizeBatch\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"bytes\",\"name\": \"batchHeader\",\"type\": \"bytes\"},{\"internalType\": \"bytes32\",\"name\": \"prevStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"postStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"withdrawRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes\",\"name\": \"blobDataProof\",\"type\": \"bytes\"}],\"name\": \"finalizeBatch4844\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"bytes\",\"name\": \"batchHeader\",\"type\": \"bytes\"},{\"internalType\": \"bytes32\",\"name\": \"prevStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"postStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"withdrawRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes\",\"name\": \"aggrProof\",\"type\": \"bytes\"}],\"name\": \"finalizeBatchWithProof\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"bytes\",\"name\": \"batchHeader\",\"type\": \"bytes\"},{\"internalType\": \"bytes32\",\"name\": \"prevStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"postStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"withdrawRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes\",\"name\": \"blobDataProof\",\"type\": \"bytes\"},{\"internalType\": \"bytes\",\"name\": \"aggrProof\",\"type\": \"bytes\"}],\"name\": \"finalizeBatchWithProof4844\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"bytes\",\"name\": \"batchHeader\",\"type\": \"bytes\"},{\"internalType\": \"bytes32\",\"name\": \"postStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"withdrawRoot\",\"type\": \"bytes32\"}],\"name\": \"finalizeBundle\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"bytes\",\"name\": \"batchHeader\",\"type\": \"bytes\"},{\"internalType\": \"bytes32\",\"name\": \"postStateRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes32\",\"name\": \"withdrawRoot\",\"type\": \"bytes32\"},{\"internalType\": \"bytes\",\"name\": \"aggrProof\",\"type\": \"bytes\"}],\"name\": \"finalizeBundleWithProof\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"uint256\",\"name\": \"batchIndex\",\"type\": \"uint256\"}],\"name\": \"finalizedStateRoots\",\"outputs\": [{\"internalType\": \"bytes32\",\"name\": \"\",\"type\": \"bytes32\"}],\"stateMutability\": \"view\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"bytes\",\"name\": \"_batchHeader\",\"type\": \"bytes\"},{\"internalType\": \"bytes32\",\"name\": \"_stateRoot\",\"type\": \"bytes32\"}],\"name\": \"importGenesisBatch\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"uint256\",\"name\": \"batchIndex\",\"type\": \"uint256\"}],\"name\": \"isBatchFinalized\",\"outputs\": [{\"internalType\": \"bool\",\"name\": \"\",\"type\": \"bool\"}],\"stateMutability\": \"view\",\"type\": \"function\"},{\"inputs\": [],\"name\": \"lastFinalizedBatchIndex\",\"outputs\": [{\"internalType\": \"uint256\",\"name\": \"\",\"type\": \"uint256\"}],\"stateMutability\": \"view\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"bytes\",\"name\": \"batchHeader\",\"type\": \"bytes\"},{\"internalType\": \"uint256\",\"name\": \"count\",\"type\": \"uint256\"}],\"name\": \"revertBatch\",\"outputs\": [],\"stateMutability\": \"nonpayable\",\"type\": \"function\"},{\"inputs\": [{\"internalType\": \"uint256\",\"name\": \"batchIndex\",\"type\": \"uint256\"}],\"name\": \"withdrawRoots\",\"outputs\": [{\"internalType\": \"bytes32\",\"name\": \"\",\"type\": \"bytes32\"}],\"stateMutability\": \"view\",\"type\": \"function\"}]",
}

const (
	// CommitEventType contains data of event of commit batch
	CommitEventType int = iota
	// RevertEventType contains data of event of revert batch
	RevertEventType
	// FinalizeEventType contains data of event of finalize batch
	FinalizeEventType
)

// RollupEvent represents a single rollup event (commit, revert, finalize)
type RollupEvent interface {
	Type() int
	BatchIndex() *big.Int
	BatchHash() common.Hash
}

type RollupEvents []RollupEvent

// CommitBatchEventUnpacked represents a CommitBatch event raised by the ScrollChain contract.
type CommitBatchEventUnpacked struct {
	batchIndex *big.Int
	batchHash  common.Hash
}

// CommitBatchEvent represents a CommitBatch event raised by the ScrollChain contract with additional fields.
type CommitBatchEvent struct {
	batchIndex  *big.Int
	batchHash   common.Hash
	txHash      common.Hash
	blockHash   common.Hash
	blockNumber uint64
}

func (c *CommitBatchEvent) Type() int {
	return CommitEventType
}

func (c *CommitBatchEvent) BatchIndex() *big.Int {
	return c.batchIndex
}

func (c *CommitBatchEvent) BatchHash() common.Hash {
	return c.batchHash
}

func (c *CommitBatchEvent) TxHash() common.Hash {
	return c.txHash
}

func (c *CommitBatchEvent) BlockHash() common.Hash {
	return c.blockHash
}

func (c *CommitBatchEvent) BlockNumber() uint64 {
	return c.blockNumber
}

// RevertBatchEvent represents a RevertBatch event raised by the ScrollChain contract.
type RevertBatchEvent struct {
	batchIndex *big.Int
	batchHash  common.Hash
}

func (r *RevertBatchEvent) Type() int {
	return RevertEventType
}

func (r *RevertBatchEvent) BatchIndex() *big.Int {
	return r.batchIndex
}

func (r *RevertBatchEvent) BatchHash() common.Hash {
	return r.batchHash
}

// FinalizeBatchEvent represents a FinalizeBatch event raised by the ScrollChain contract.
type FinalizeBatchEvent struct {
	batchIndex   *big.Int
	batchHash    common.Hash
	stateRoot    common.Hash
	withdrawRoot common.Hash
}

func (r *FinalizeBatchEvent) Type() int {
	return FinalizeEventType
}

func (r *FinalizeBatchEvent) BatchIndex() *big.Int {
	return r.batchIndex
}

func (r *FinalizeBatchEvent) BatchHash() common.Hash {
	return r.batchHash
}

func (r *FinalizeBatchEvent) StateRoot() common.Hash {
	return r.stateRoot
}

func (r *FinalizeBatchEvent) WithdrawRoot() common.Hash {
	return r.withdrawRoot
}

// UnpackLog unpacks a retrieved log into the provided output structure.
func UnpackLog(c *abi.ABI, out interface{}, event string, log types.Log) error {
	if log.Topics[0] != c.Events[event].ID {
		return fmt.Errorf("event signature mismatch")
	}
	if len(log.Data) > 0 {
		if err := c.UnpackIntoInterface(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed abi.Arguments
	for _, arg := range c.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return abi.ParseTopics(out, indexed, log.Topics[1:])
}
