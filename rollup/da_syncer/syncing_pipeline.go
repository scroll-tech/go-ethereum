package da_syncer

import (
	"context"
	"errors"
	"time"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
)

// Config is the configuration parameters of data availability syncing.
type Config struct {
	FetcherMode      FetcherMode // mode of fetcher
	SnapshotFilePath string      // path to snapshot file
}

var (
	errInvalidChain = errors.New("retrieved hash chain is invalid")
)

// defaultSyncInterval is the frequency at which we query for new rollup event.
const defaultSyncInterval = 1 * time.Second

type SyncingPipeline struct {
	ctx        context.Context
	cancel     context.CancelFunc
	db         ethdb.Database
	blockchain *core.BlockChain
	blockQueue *BlockQueue
	daSyncer   *DaSyncer
}

func NewSyncingPipeline(ctx context.Context, blockchain *core.BlockChain, genesisConfig *params.ChainConfig, db ethdb.Database, ethClient sync_service.EthClient, l1DeploymentBlock uint64, config Config) (*SyncingPipeline, error) {
	ctx, cancel := context.WithCancel(ctx)
	var err error

	l1Client, err := newL1Client(ctx, genesisConfig, ethClient)
	if err != nil {
		cancel()
		return nil, err
	}
	blobClient, err := newBlobScanClient()
	if err != nil {
		cancel()
		return nil, err
	}
	dataSourceFactory := NewDataSourceFactory(blockchain, genesisConfig, config, l1Client, blobClient, db)
	// todo: keep synced l1 height somewhere
	var syncedL1Height uint64 = l1DeploymentBlock - 1
	from := rawdb.ReadDASyncedL1BlockNumber(db)
	if from != nil {
		syncedL1Height = *from
	}
	daQueue := NewDaQueue(syncedL1Height, dataSourceFactory)
	batchQueue := NewBatchQueue(daQueue, db)
	blockQueue := NewBlockQueue(batchQueue)
	daSyncer := NewDaSyncer(blockchain)

	return &SyncingPipeline{
		ctx:        ctx,
		cancel:     cancel,
		db:         db,
		blockchain: blockchain,
		blockQueue: blockQueue,
		daSyncer:   daSyncer,
	}, nil
}

func (sp *SyncingPipeline) Step(ctx context.Context) error {
	block, err := sp.blockQueue.NextBlock(ctx)
	if err != nil {
		return err
	}
	err = sp.daSyncer.SyncOneBlock(block)
	return err
}

func (sp *SyncingPipeline) Start() {
	if sp == nil {
		return
	}

	log.Info("Starting SyncingPipeline")

	go func() {
		syncTicker := time.NewTicker(defaultSyncInterval)
		defer syncTicker.Stop()

		for {
			err := sp.Step(sp.ctx)
			if err != nil {
				log.Warn("syncing pipeline step failed", "err", err)
			}
			select {
			case <-sp.ctx.Done():
				return
			case <-syncTicker.C:
				continue
			}
		}
	}()
}

func (sp *SyncingPipeline) Stop() {
	if sp == nil {
		return
	}

	log.Info("Stopping DaSyncer")

	if sp.cancel != nil {
		sp.cancel()
	}
}
