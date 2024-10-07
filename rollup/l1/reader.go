package l1

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

const (
	commitBatchEventName   = "CommitBatch"
	revertBatchEventName   = "RevertBatch"
	finalizeBatchEventName = "FinalizeBatch"

	defaultL1MsgFetchBlockRange        = 500
	defaultRollupEventsFetchBlockRange = 100
)

type Reader struct {
	ctx      context.Context
	config   Config
	client   Client
	filterer *L1MessageQueueFilterer

	scrollChainABI                *abi.ABI
	l1CommitBatchEventSignature   common.Hash
	l1RevertBatchEventSignature   common.Hash
	l1FinalizeBatchEventSignature common.Hash
}

// Config is the configuration parameters of data availability syncing.
type Config struct {
	ScrollChainAddress    common.Address // address of ScrollChain contract
	L1MessageQueueAddress common.Address // address of L1MessageQueue contract
}

// NewReader initializes a new Reader instance
func NewReader(ctx context.Context, config Config, l1Client Client) (*Reader, error) {
	if config.ScrollChainAddress == (common.Address{}) {
		return nil, errors.New("must pass non-zero scrollChainAddress to L1Client")
	}

	scrollChainABI, err := ScrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}

	filterer, err := NewL1MessageQueueFilterer(config.L1MessageQueueAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize L1MessageQueueFilterer, err = %w", err)
	}

	reader := Reader{
		ctx:      ctx,
		config:   config,
		client:   l1Client,
		filterer: filterer,

		scrollChainABI:                scrollChainABI,
		l1CommitBatchEventSignature:   scrollChainABI.Events[commitBatchEventName].ID,
		l1RevertBatchEventSignature:   scrollChainABI.Events[revertBatchEventName].ID,
		l1FinalizeBatchEventSignature: scrollChainABI.Events[finalizeBatchEventName].ID,
	}

	return &reader, nil
}

// GetLatestFinalizedBlockNumber fetches the block number of the latest finalized block from the L1 chain.
func (r *Reader) GetLatestFinalizedBlockNumber() (uint64, error) {
	header, err := r.client.HeaderByNumber(r.ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return 0, err
	}
	if !header.Number.IsInt64() {
		return 0, fmt.Errorf("received unexpected block number in L1Client: %v", header.Number)
	}
	return header.Number.Uint64(), nil
}

// FetchBlockHeaderByNumber fetches the block header by number
func (r *Reader) FetchBlockHeaderByNumber(blockNumber uint64) (*types.Header, error) {
	return r.client.HeaderByNumber(r.ctx, big.NewInt(int64(blockNumber)))
}

// FetchTxData fetches tx data corresponding to given event log
func (r *Reader) FetchTxData(txHash, blockHash common.Hash) ([]byte, error) {
	tx, err := r.fetchTx(txHash, blockHash)
	if err != nil {
		return nil, err
	}
	return tx.Data(), nil
}

// FetchTxBlobHash fetches tx blob hash corresponding to given event log
func (r *Reader) FetchTxBlobHash(txHash, blockHash common.Hash) (common.Hash, error) {
	tx, err := r.fetchTx(txHash, blockHash)
	if err != nil {
		return common.Hash{}, err
	}
	blobHashes := tx.BlobHashes()
	if len(blobHashes) == 0 {
		return common.Hash{}, fmt.Errorf("transaction does not contain any blobs, tx hash: %v", txHash.Hex())
	}
	return blobHashes[0], nil
}

// FetchRollupEventsInRange retrieves and parses commit/revert/finalize rollup events between block numbers: [from, to].
func (r *Reader) FetchRollupEventsInRange(from, to uint64) (RollupEvents, error) {
	log.Trace("L1Client fetchRollupEventsInRange", "fromBlock", from, "toBlock", to)
	var logs []types.Log

	err := r.queryInBatches(from, to, defaultRollupEventsFetchBlockRange, func(from, to uint64) error {
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(from)), // inclusive
			ToBlock:   big.NewInt(int64(to)),   // inclusive
			Addresses: []common.Address{
				r.config.ScrollChainAddress,
			},
			Topics: make([][]common.Hash, 1),
		}
		query.Topics[0] = make([]common.Hash, 3)
		query.Topics[0][0] = r.l1CommitBatchEventSignature
		query.Topics[0][1] = r.l1RevertBatchEventSignature
		query.Topics[0][2] = r.l1FinalizeBatchEventSignature

		logsBatch, err := r.client.FilterLogs(r.ctx, query)
		if err != nil {
			return fmt.Errorf("failed to filter logs, err: %w", err)
		}
		logs = append(logs, logsBatch...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.processLogsToRollupEvents(logs)
}

func (r *Reader) FetchL1MessagesInRange(fromBlock, toBlock uint64) ([]types.L1MessageTx, error) {
	var msgs []types.L1MessageTx

	err := r.queryInBatches(fromBlock, toBlock, defaultL1MsgFetchBlockRange, func(from, to uint64) error {
		it, err := r.filterer.FilterQueueTransaction(&bind.FilterOpts{
			Start:   from,
			End:     &to,
			Context: r.ctx,
		}, nil, nil)
		if err != nil {
			return err
		}
		for it.Next() {
			event := it.Event
			log.Trace("Received new L1 QueueTransaction event", "event", event)

			if !event.GasLimit.IsUint64() {
				return fmt.Errorf("invalid QueueTransaction event: QueueIndex = %v, GasLimit = %v", event.QueueIndex, event.GasLimit)
			}

			msgs = append(msgs, types.L1MessageTx{
				QueueIndex: event.QueueIndex,
				Gas:        event.GasLimit.Uint64(),
				To:         &event.Target,
				Value:      event.Value,
				Data:       event.Data,
				Sender:     event.Sender,
			})
		}
		return it.Error()
	})
	if err != nil {
		return nil, err
	}
	return msgs, nil
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

func (r *Reader) queryInBatches(fromBlock, toBlock uint64, batchSize int, queryFunc func(from, to uint64) error) error {
	for from := fromBlock; from <= toBlock; from += uint64(batchSize) {
		to := from + defaultL1MsgFetchBlockRange - 1
		if to > toBlock {
			to = toBlock
		}
		err := queryFunc(from, to)
		if err != nil {
			return err
		}
	}
	return nil
}

// fetchTx fetches tx corresponding to given event log
func (r *Reader) fetchTx(txHash, blockHash common.Hash) (*types.Transaction, error) {
	tx, _, err := r.client.TransactionByHash(r.ctx, txHash)
	if err != nil {
		log.Debug("failed to get transaction by hash, probably an unindexed transaction, fetching the whole block to get the transaction",
			"tx hash", txHash.Hex(), "block hash", blockHash.Hex(), "err", err)
		block, err := r.client.BlockByHash(r.ctx, blockHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get block by hash, block hash: %v, err: %w", blockHash.Hex(), err)
		}

		found := false
		for _, txInBlock := range block.Transactions() {
			if txInBlock.Hash() == txHash {
				tx = txInBlock
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("transaction not found in the block, tx hash: %v, block hash: %v", txHash.Hex(), blockHash.Hex())
		}
	}

	return tx, nil
}
