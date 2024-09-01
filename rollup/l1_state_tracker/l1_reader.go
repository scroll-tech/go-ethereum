package l1_state_tracker

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
)

type L1Reader struct {
	ctx        context.Context
	config     Config
	client     sync_service.EthClient
	blobClient blob_client.BlobClient

	l1CommitBatchEventSignature   common.Hash
	l1RevertBatchEventSignature   common.Hash
	l1FinalizeBatchEventSignature common.Hash
}

// Config is the configuration parameters of data availability syncing.
type Config struct {
	BlobScanAPIEndpoint    string         // BlobScan blob api endpoint
	BlockNativeAPIEndpoint string         // BlockNative blob api endpoint
	BeaconNodeAPIEndpoint  string         // Beacon node api endpoint
	scrollChainAddress     common.Address // address of ScrollChain contract
}

// NewL1Reader initializes a new L1Reader instance
func NewL1Reader(ctx context.Context, config Config, l1Client sync_service.EthClient, scrollChainABI *abi.ABI) (*L1Reader, error) {
	if config.scrollChainAddress == (common.Address{}) {
		return nil, errors.New("must pass non-zero scrollChainAddress to L1Client")
	}

	blobClientList := blob_client.NewBlobClientList()
	if config.BeaconNodeAPIEndpoint != "" {
		beaconNodeClient, err := blob_client.NewBeaconNodeClient(config.BeaconNodeAPIEndpoint)
		if err != nil {
			log.Warn("failed to create BeaconNodeClient", "err", err)
		} else {
			blobClientList.AddBlobClient(beaconNodeClient)
		}
	}
	if config.BlobScanAPIEndpoint != "" {
		blobClientList.AddBlobClient(blob_client.NewBlobScanClient(config.BlobScanAPIEndpoint))
	}
	if config.BlockNativeAPIEndpoint != "" {
		blobClientList.AddBlobClient(blob_client.NewBlockNativeClient(config.BlockNativeAPIEndpoint))
	}
	if blobClientList.Size() == 0 {
		log.Crit("DA syncing is enabled but no blob client is configured. Please provide at least one blob client via command line flag.")
	}

	client := L1Reader{
		ctx:        ctx,
		config:     config,
		client:     l1Client,
		blobClient: blobClientList,

		l1CommitBatchEventSignature:   scrollChainABI.Events["CommitBatch"].ID,
		l1RevertBatchEventSignature:   scrollChainABI.Events["RevertBatch"].ID,
		l1FinalizeBatchEventSignature: scrollChainABI.Events["FinalizeBatch"].ID,
	}

	return &client, nil
}

// FetchTxData fetches tx data corresponding to given event log
func (r *L1Reader) FetchTxData(vLog *types.Log) ([]byte, error) {
	tx, err := r.fetchTx(vLog)
	if err != nil {
		return nil, err
	}
	return tx.Data(), nil
}

// FetchBlobByEventLog returns blob corresponding for the given event log
func (r *L1Reader) FetchBlobByEventLog(vLog *types.Log) (*kzg4844.Blob, error) {
	versionedHash, err := r.fetchTxBlobHash(vLog)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob hash, err: %w", err)
	}
	header, err := r.FetchBlockHeaderByNumber(big.NewInt(0).SetUint64(vLog.BlockNumber))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch header by number, err: %w", err)
	}
	return r.blobClient.GetBlobByVersionedHashAndBlockTime(r.ctx, versionedHash, header.Time)
}

// FetchRollupEventsInRange retrieves and parses commit/revert/finalize rollup events between block numbers: [from, to].
func (r *L1Reader) FetchRollupEventsInRange(from, to uint64) ([]types.Log, error) {
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
	return logs, nil
}

// FetchBlockHeaderByNumber fetches the block header by number
func (r *L1Reader) FetchBlockHeaderByNumber(blockNumber *big.Int) (*types.Header, error) {
	return r.client.HeaderByNumber(r.ctx, blockNumber)
}

// fetchTx fetches tx corresponding to given event log
func (r *L1Reader) fetchTx(vLog *types.Log) (*types.Transaction, error) {
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

// fetchTxBlobHash fetches tx blob hash corresponding to given event log
func (r *L1Reader) fetchTxBlobHash(vLog *types.Log) (common.Hash, error) {
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
