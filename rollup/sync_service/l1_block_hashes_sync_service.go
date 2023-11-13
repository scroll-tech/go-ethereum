package sync_service

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
)

// L1BlockHashesSyncService collects all L1 block hashes and stores them in a local database.
// L1BlockHashesTx
// TODO(l1blockhashes): Merge this service's block hashes logic into SyncService as it must use only 1 latestBlockNumber.
// SyncService also represents an encapsulation of synchronising data from L1 to L2.
// Currently, this is separate for the PoC and easier debugging.
type L1BlockHashesSyncService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	client               *BridgeClient
	db                   ethdb.Database
	blockHashesFeed      event.Feed
	pollInterval         time.Duration
	latestProcessedBlock uint64
	scope                event.SubscriptionScope
}

func NewL1BlockHashesSyncService(ctx context.Context, genesisConfig *params.ChainConfig, nodeConfig *node.Config, db ethdb.Database, l1Client EthClient) (*L1BlockHashesSyncService, error) {
	// terminate if the caller does not provide an L1 client (e.g. in tests)
	if l1Client == nil || (reflect.ValueOf(l1Client).Kind() == reflect.Ptr && reflect.ValueOf(l1Client).IsNil()) {
		log.Warn("No L1 client provided, L1 sync service will not run")
		return nil, nil
	}

	if genesisConfig.Scroll.L1Config == nil {
		return nil, fmt.Errorf("missing L1 config in genesis")
	}

	client, err := newBridgeClient(ctx, l1Client, genesisConfig.Scroll.L1Config.L1ChainId, nodeConfig.L1Confirmations, genesisConfig.Scroll.L1Config.L1MessageQueueAddress, genesisConfig.Scroll.L1Config.L1BlockHashesAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bridge client: %w", err)
	}

	// assume deployment block has 0 messages
	latestProcessedBlock := nodeConfig.L1BlockHashesBlock
	blockNumber := rawdb.ReadL1BlockHashesSyncedL1BlockNumber(db)
	if blockNumber != nil {
		// restart from latest synced block number
		latestProcessedBlock = *blockNumber
	}

	ctx, cancel := context.WithCancel(ctx)

	service := L1BlockHashesSyncService{
		ctx:                  ctx,
		cancel:               cancel,
		client:               client,
		db:                   db,
		pollInterval:         DefaultPollInterval,
		latestProcessedBlock: latestProcessedBlock,
	}

	return &service, nil
}

func (s *L1BlockHashesSyncService) Start() {
	if s == nil {
		return
	}

	// wait for initial sync before starting node
	log.Info("Starting L1BlockHashes sync service", "latestProcessedBlock", s.latestProcessedBlock)

	// block node startup during initial sync and print some helpful logs
	latestConfirmed, err := s.client.getLatestConfirmedBlockNumber(s.ctx)
	if err == nil && latestConfirmed > s.latestProcessedBlock+1000 {
		log.Warn("Running initial sync of L1BlockHashes before starting l2geth, this might take a while...")
		s.fetchBlockHashesTx()
		log.Info("L1BlockHashes initial sync completed", "latestProcessedBlock", s.latestProcessedBlock)
	}

	go func() {
		t := time.NewTicker(s.pollInterval)
		defer t.Stop()

		for {
			// don't wait for ticker during startup
			s.fetchBlockHashesTx()

			select {
			case <-s.ctx.Done():
				return
			case <-t.C:
				continue
			}
		}
	}()
}

func (s *L1BlockHashesSyncService) Stop() {
	if s == nil {
		return
	}

	log.Info("Stopping sync service")

	// Unsubscribe all subscriptions registered
	s.scope.Close()

	if s.cancel != nil {
		s.cancel()
	}
}

// SubscribeNewL1BlockHashesTxEvent registers a subscription of NewL1BlockHashesTxEvent and
// starts sending event to the given channel.
func (s *L1BlockHashesSyncService) SubscribeNewL1BlockHashesTxEvent(ch chan<- core.NewL1BlockHashesTxEvent) event.Subscription {
	return s.scope.Track(s.blockHashesFeed.Subscribe(ch))
}

func (s *L1BlockHashesSyncService) fetchBlockHashesTx() {
	latestConfirmed, err := s.client.getLatestConfirmedBlockNumber(s.ctx)
	if err != nil {
		log.Warn("Failed to get latest confirmed block number", "err", err)
		return
	}

	log.Trace("Sync service fetchBlockHashesTx", "latestProcessedBlock", s.latestProcessedBlock, "latestConfirmed", latestConfirmed)

	batchWriter := s.db.NewBatch()
	numBlocksPendingDbWrite := uint64(0)
	numBlockHashesTxPendingDbWrite := 0

	// helper function to flush database writes cached in memory
	flush := func(lastBlock uint64) {
		// update sync progress
		rawdb.WriteL1BlockHashesSyncedBlockNumber(batchWriter, lastBlock)

		// write batch in a single transaction
		err := batchWriter.Write()
		if err != nil {
			// crash on database error, no risk of inconsistency here
			log.Crit("Failed to write L1BlockHashesTx to database", "err", err)
		}

		batchWriter.Reset()
		numBlocksPendingDbWrite = 0
		if numBlockHashesTxPendingDbWrite > 0 {
			s.blockHashesFeed.Send(core.NewL1BlockHashesTxEvent{HasNewBlockHashesTx: true})
			numBlockHashesTxPendingDbWrite = 0
		}
		s.latestProcessedBlock = lastBlock
	}
	from := s.latestProcessedBlock + 1
	to := latestConfirmed

	tx, err := s.client.fetchBlockHashesInRange(s.ctx, from, to)
	if err != nil {
		log.Warn("Failed to fetch L1BlockHashes in range", "fromBlock", from, "toBlock", to)
		return
	}

	if !reflect.DeepEqual(tx, types.L1BlockHashesTx{}) {
		log.Debug("Received new L1BlockHashesTx", "from", from, "toBlock", to)
		rawdb.WriteL1BlockHashesTx(batchWriter, tx, from)
	}

	// TODO(l1blockhashes): if it fetches a lot of block hashes, this might overflow and should be done in chunks.
	// flush new messages to database periodically
	if to == latestConfirmed || batchWriter.ValueSize() >= DbWriteThresholdBytes || numBlocksPendingDbWrite >= DbWriteThresholdBlocks {
		flush(to)
	}
}
