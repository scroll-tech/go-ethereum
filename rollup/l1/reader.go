package l1

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
)

const (
	commitBatchEventName   = "CommitBatch"
	revertBatchEventName   = "RevertBatch"
	finalizeBatchEventName = "FinalizeBatch"
)

type Reader struct {
	ctx    context.Context
	config Config
	client sync_service.EthClient

	scrollChainABI                *abi.ABI
	l1CommitBatchEventSignature   common.Hash
	l1RevertBatchEventSignature   common.Hash
	l1FinalizeBatchEventSignature common.Hash
}

// Config is the configuration parameters of data availability syncing.
type Config struct {
	scrollChainAddress common.Address // address of ScrollChain contract
}

// NewReader initializes a new Reader instance
func NewReader(ctx context.Context, config Config, l1Client sync_service.EthClient) (*Reader, error) {
	if config.scrollChainAddress == (common.Address{}) {
		return nil, errors.New("must pass non-zero scrollChainAddress to L1Client")
	}

	scrollChainABI, err := ScrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}

	client := Reader{
		ctx:    ctx,
		config: config,
		client: l1Client,

		scrollChainABI:                scrollChainABI,
		l1CommitBatchEventSignature:   scrollChainABI.Events[commitBatchEventName].ID,
		l1RevertBatchEventSignature:   scrollChainABI.Events[revertBatchEventName].ID,
		l1FinalizeBatchEventSignature: scrollChainABI.Events[finalizeBatchEventName].ID,
	}

	return &client, nil
}

// FetchTxData fetches tx data corresponding to given event log
func (r *Reader) FetchTxData(vLog *types.Log) ([]byte, error) {
	tx, err := r.fetchTx(vLog)
	if err != nil {
		return nil, err
	}
	return tx.Data(), nil
}

// FetchRollupEventsInRange retrieves and parses commit/revert/finalize rollup events between block numbers: [from, to].
func (r *Reader) FetchRollupEventsInRange(from, to uint64) (RollupEvents, error) {
	log.Trace("L1Client fetchRollupEventsInRange", "fromBlock", from, "toBlock", to)

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(from)), // inclusive
		ToBlock:   big.NewInt(int64(to)),   // inclusive
		Addresses: []common.Address{
			r.config.scrollChainAddress,
		},
		Topics: make([][]common.Hash, 1),
	}
	query.Topics[0] = make([]common.Hash, 3)
	query.Topics[0][0] = r.l1CommitBatchEventSignature
	query.Topics[0][1] = r.l1RevertBatchEventSignature
	query.Topics[0][2] = r.l1FinalizeBatchEventSignature

	logs, err := r.client.FilterLogs(r.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to filter logs, err: %w", err)
	}
	return r.processLogsToRollupEvents(logs)
}

func (r *Reader) processLogsToRollupEvents(logs []types.Log) (RollupEvents, error) {
	var rollupEvents RollupEvents
	var rollupEvent RollupEvent
	var err error

	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case r.l1CommitBatchEventSignature:
			event := &CommitBatchEventUnpacked{}
			if err = UnpackLog(r.scrollChainABI, event, commitBatchEventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack commit rollup event log, err: %w", err)
			}
			log.Trace("found new CommitBatch event", "batch index", event.batchIndex.Uint64())
			rollupEvent = &CommitBatchEvent{
				batchIndex:  event.batchIndex,
				batchHash:   event.batchHash,
				txHash:      vLog.TxHash,
				blockHash:   vLog.BlockHash,
				blockNumber: vLog.BlockNumber,
			}

		case r.l1RevertBatchEventSignature:
			event := &RevertBatchEvent{}
			if err = UnpackLog(r.scrollChainABI, event, revertBatchEventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack revert rollup event log, err: %w", err)
			}
			log.Trace("found new RevertBatchType event", "batch index", event.batchIndex.Uint64())
			rollupEvent = event

		case r.l1FinalizeBatchEventSignature:
			event := &FinalizeBatchEvent{}
			if err = UnpackLog(r.scrollChainABI, event, finalizeBatchEventName, vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack finalized rollup event log, err: %w", err)
			}
			log.Trace("found new FinalizeBatchType event", "batch index", event.batchIndex.Uint64())
			rollupEvent = event

		default:
			return nil, fmt.Errorf("unknown event, topic: %v, tx hash: %v", vLog.Topics[0].Hex(), vLog.TxHash.Hex())
		}

		rollupEvents = append(rollupEvents, rollupEvent)
	}
	return rollupEvents, nil
}

// FetchBlockHeaderByNumber fetches the block header by number
func (r *Reader) FetchBlockHeaderByNumber(blockNumber *big.Int) (*types.Header, error) {
	return r.client.HeaderByNumber(r.ctx, blockNumber)
}

// fetchTx fetches tx corresponding to given event log
func (r *Reader) fetchTx(vLog *types.Log) (*types.Transaction, error) {
	tx, _, err := r.client.TransactionByHash(r.ctx, vLog.TxHash)
	if err != nil {
		log.Debug("failed to get transaction by hash, probably an unindexed transaction, fetching the whole block to get the transaction",
			"tx hash", vLog.TxHash.Hex(), "block number", vLog.BlockNumber, "block hash", vLog.BlockHash.Hex(), "err", err)
		block, err := r.client.BlockByHash(r.ctx, vLog.BlockHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get block by hash, block number: %v, block hash: %v, err: %w", vLog.BlockNumber, vLog.BlockHash.Hex(), err)
		}

		found := false
		for _, txInBlock := range block.Transactions() {
			if txInBlock.Hash() == vLog.TxHash {
				tx = txInBlock
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("transaction not found in the block, tx hash: %v, block number: %v, block hash: %v", vLog.TxHash.Hex(), vLog.BlockNumber, vLog.BlockHash.Hex())
		}
	}

	return tx, nil
}

// FetchTxBlobHash fetches tx blob hash corresponding to given event log
func (r *Reader) FetchTxBlobHash(vLog *types.Log) (common.Hash, error) {
	tx, err := r.fetchTx(vLog)
	if err != nil {
		return common.Hash{}, err
	}
	blobHashes := tx.BlobHashes()
	if len(blobHashes) == 0 {
		return common.Hash{}, fmt.Errorf("transaction does not contain any blobs, tx hash: %v", vLog.TxHash.Hex())
	}
	return blobHashes[0], nil
}
