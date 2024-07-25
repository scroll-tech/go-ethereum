package da_syncer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common/backoff"
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

type SyncingPipeline struct {
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	expBackoff *backoff.Exponential

	db         ethdb.Database
	blockchain *core.BlockChain
	blockQueue *BlockQueue
	daSyncer   *DASyncer
}

func NewSyncingPipeline(ctx context.Context, blockchain *core.BlockChain, genesisConfig *params.ChainConfig, db ethdb.Database, ethClient sync_service.EthClient, l1DeploymentBlock uint64, config Config) (*SyncingPipeline, error) {
	scrollChainABI, err := rollup_sync_service.ScrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}

	l1Client, err := rollup_sync_service.NewL1Client(ctx, ethClient, genesisConfig.Scroll.L1Config.L1ChainId, genesisConfig.Scroll.L1Config.ScrollChainAddress, scrollChainABI)
	if err != nil {
		return nil, err
	}

	var blobClient blob_client.BlobClient
	switch config.BlobSource {
	case blob_client.BlobScan:
		blobClient = blob_client.NewBlobScanClient(genesisConfig.Scroll.DAConfig.BlobScanAPIEndpoint)
	case blob_client.BlockNative:
		blobClient = blob_client.NewBlockNativeClient(genesisConfig.Scroll.DAConfig.BlockNativeAPIEndpoint)
	default:
		return nil, fmt.Errorf("unknown blob scan client: %d", config.BlobSource)
	}

	dataSourceFactory := NewDataSourceFactory(blockchain, genesisConfig, config, l1Client, blobClient, db)
	syncedL1Height := l1DeploymentBlock - 1
	from := rawdb.ReadDASyncedL1BlockNumber(db)
	if from != nil {
		syncedL1Height = *from
	}

	daQueue := NewDAQueue(syncedL1Height, dataSourceFactory)
	batchQueue := NewBatchQueue(daQueue, db)
	blockQueue := NewBlockQueue(batchQueue)
	daSyncer := NewDASyncer(blockchain)

	ctx, cancel := context.WithCancel(ctx)
	return &SyncingPipeline{
		ctx:        ctx,
		cancel:     cancel,
		expBackoff: backoff.NewExponential(100*time.Millisecond, 10*time.Second, 100*time.Millisecond),
		db:         db,
		blockchain: blockchain,
		blockQueue: blockQueue,
		daSyncer:   daSyncer,
	}, nil
}

func (s *SyncingPipeline) Step() error {
	block, err := s.blockQueue.NextBlock(s.ctx)
	if err != nil {
		return err
	}
	err = s.daSyncer.SyncOneBlock(block)
	return err
}

func (s *SyncingPipeline) Start() {
	log.Info("Starting SyncingPipeline")

	s.wg.Add(1)
	go func() {
		s.mainLoop()
		s.wg.Done()
	}()
}

func (s *SyncingPipeline) mainLoop() {
	stepCh := make(chan struct{}, 1)
	var delayedStepCh <-chan time.Time

	// reqStep is a helper function to request a step to be executed.
	// If delay is true, it will request a delayed step with exponential backoff, otherwise it will request an immediate step.
	reqStep := func(delay bool) {
		if delay {
			if delayedStepCh == nil {
				delayDur := s.expBackoff.NextDuration()
				delayedStepCh = time.After(delayDur)
				log.Debug("requesting delayed step", "delay", delayDur, "attempt", s.expBackoff.Attempt())
			} else {
				log.Debug("ignoring step request because of ongoing delayed step", "attempt", s.expBackoff.Attempt())
			}
		} else {
			select {
			case stepCh <- struct{}{}:
			default:
			}
		}
	}

	// start pipeline
	reqStep(false)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		select {
		case <-s.ctx.Done():
			return
		case <-delayedStepCh:
			delayedStepCh = nil
			reqStep(false)
		case <-stepCh:
			err := s.Step()
			if err == nil {
				reqStep(false)
				s.expBackoff.Reset()
				continue
			}

			if errors.Is(err, io.EOF) {
				reqStep(true)
				continue
			}
			if errors.Is(err, context.Canceled) {
				return
			}

			if strings.HasPrefix(err.Error(), "not consecutive block") {
				log.Warn("syncing pipeline step failed, probably because of restart", "err", err)
			} else {
				log.Crit("syncing pipeline step failed", "err", err)
			}
		}
	}
}

func (s *SyncingPipeline) Stop() {
	log.Info("Stopping DaSyncer...")
	s.cancel()
	s.wg.Wait()
	log.Info("Stopped DaSyncer... Done")
}
