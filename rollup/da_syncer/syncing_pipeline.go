package da_syncer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/rollup_sync_service"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
)

// Config is the configuration parameters of data availability syncing.
type Config struct {
	FetcherMode      FetcherMode            // mode of fetcher
	SnapshotFilePath string                 // path to snapshot file
	BlobSource       blob_client.BlobSource // blob source
}

// defaultSyncInterval is the frequency at which we query for new rollup event.
const defaultSyncInterval = 1 * time.Millisecond

type SyncingPipeline struct {
	ctx        context.Context
	cancel     context.CancelFunc
	db         ethdb.Database
	blockchain *core.BlockChain
	blockQueue *BlockQueue
	daSyncer   *DASyncer
}

func NewSyncingPipeline(ctx context.Context, blockchain *core.BlockChain, genesisConfig *params.ChainConfig, db ethdb.Database, ethClient sync_service.EthClient, l1DeploymentBlock uint64, config Config) (*SyncingPipeline, error) {
	ctx, cancel := context.WithCancel(ctx)

	scrollChainABI, err := rollup_sync_service.ScrollChainMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}

	l1Client, err := rollup_sync_service.NewL1Client(ctx, ethClient, genesisConfig.Scroll.L1Config.L1ChainId, genesisConfig.Scroll.L1Config.ScrollChainAddress, scrollChainABI)
	if err != nil {
		cancel()
		return nil, err
	}
	var blobClient blob_client.BlobClient
	switch config.BlobSource {
	case blob_client.BlobScan:
		blobClient = blob_client.NewBlobScanClient(genesisConfig.Scroll.DAConfig.BlobScanAPIEndpoint)
	case blob_client.BlockNative:
		blobClient = blob_client.NewBlockNativeClient(genesisConfig.Scroll.DAConfig.BlockNativeAPIEndpoint)
	default:
		cancel()
		return nil, fmt.Errorf("unknown blob scan client: %d", config.BlobSource)
	}

	dataSourceFactory := NewDataSourceFactory(blockchain, genesisConfig, config, l1Client, blobClient, db)
	syncedL1Height := l1DeploymentBlock - 1
	from := rawdb.ReadDASyncedL1BlockNumber(db)
	if from != nil {
		syncedL1Height = *from
	}
	DAQueue := NewDAQueue(syncedL1Height, dataSourceFactory)
	batchQueue := NewBatchQueue(DAQueue, db)
	blockQueue := NewBlockQueue(batchQueue)
	daSyncer := NewDASyncer(blockchain)

	return &SyncingPipeline{
		ctx:        ctx,
		cancel:     cancel,
		db:         db,
		blockchain: blockchain,
		blockQueue: blockQueue,
		daSyncer:   daSyncer,
	}, nil
}

func (sp *SyncingPipeline) Step() error {
	block, err := sp.blockQueue.NextBlock(sp.ctx)
	if err != nil {
		return err
	}
	err = sp.daSyncer.SyncOneBlock(block)
	return err
}

func (sp *SyncingPipeline) Start() {
	log.Info("Starting SyncingPipeline")

	go func() {
		syncTicker := time.NewTicker(defaultSyncInterval)
		defer syncTicker.Stop()

		for {
			err := sp.Step()
			if err != nil {
				if strings.HasPrefix(err.Error(), "not consecutive block") {
					log.Warn("syncing pipeline step failed, probably because of restart", "err", err)
				} else {
					log.Crit("syncing pipeline step failed", "err", err)
				}
			}
			select {
			case <-sp.ctx.Done():
				return
			case <-syncTicker.C:
				select {
				case <-sp.ctx.Done():
					return
				default:
				}
				continue
			}
		}
	}()
}

func (sp *SyncingPipeline) Stop() {
	log.Info("Stopping DaSyncer")
	sp.cancel()
}
