package sync_service

import (
	"context"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
)

// DefaultFetchBlockRange is the number of blocks that we collect in a single eth_getLogs query.
const DefaultFetchBlockRange = uint64(20)

// DefaultPollInterval is the frequency at which we query for new L1 messages.
const DefaultPollInterval = time.Second * 15

// DbWriteThresholdBytes is the size of batched database writes in bytes.
const DbWriteThresholdBytes = 10 * 1024

// DbWriteThresholdBlocks is the number of blocks scanned after which we write to the database
// even if we have not collected DbWriteThresholdBytes bytes of data yet. This way, if there is
// a long section of L1 blocks with no messages and we stop or crash, we will not need to re-scan
// this secion.
const DbWriteThresholdBlocks = 100

// SyncService collects all L1 messages and stores them in a local database.
type SyncService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	client               *BridgeClient
	db                   ethdb.Database
	pollInterval         time.Duration
	latestProcessedBlock uint64
}

func NewSyncService(ctx context.Context, genesisConfig *params.ChainConfig, nodeConfig *node.Config, db ethdb.Database, l1Client EthClient) (*SyncService, error) {
	// terminate if the caller does not provide an L1 client (e.g. in tests)
	if l1Client == nil {
		log.Warn("No L1 client provided, L1 sync service will not run")
		return nil, nil
	}

	if genesisConfig.L1Config == nil {
		return nil, fmt.Errorf("missing L1 config in genesis")
	}

	client, err := newBridgeClient(ctx, l1Client, genesisConfig.L1Config.L1ChainId, nodeConfig.L1Confirmations, genesisConfig.L1Config.L1MessageQueueAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bridge client: %w", err)
	}

	// restart from latest synced block number
	latestProcessedBlock := uint64(0)
	block := rawdb.ReadSyncedL1BlockNumber(db)
	if block != nil {
		latestProcessedBlock = *block
	} else {
		// assume deployment block has 0 messages
		latestProcessedBlock = nodeConfig.L1DeploymentBlock
	}

	ctx, cancel := context.WithCancel(ctx)

	service := SyncService{
		ctx:                  ctx,
		cancel:               cancel,
		client:               client,
		db:                   db,
		pollInterval:         DefaultPollInterval,
		latestProcessedBlock: latestProcessedBlock,
	}

	return &service, nil
}

func (s *SyncService) Start() {
	if s == nil {
		return
	}

	log.Info("Starting sync service", "latestProcessedBlock", s.latestProcessedBlock)

	t := time.NewTicker(s.pollInterval)
	defer t.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-t.C:
			s.fetchMessages()
		}
	}
}

func (s *SyncService) Stop() {
	if s == nil {
		return
	}

	log.Info("Stopping sync service")

	if s.cancel != nil {
		defer s.cancel()
	}
}

func (s *SyncService) fetchMessages() {
	latestConfirmed, err := s.client.getLatestConfirmedBlockNumber(s.ctx)
	if err != nil {
		log.Warn("failed to get latest confirmed block number", "err", err)
		return
	}

	log.Trace("Sync service fetchMessages", "latestProcessedBlock", s.latestProcessedBlock, "latestConfirmed", latestConfirmed)

	batchWriter := s.db.NewBatch()
	numBlocksPendingDbWrite := uint64(0)

	// helper function to flush database writes cached in memory
	flush := func(lastBlock uint64) {
		// update sync progress
		rawdb.WriteSyncedL1BlockNumber(batchWriter, lastBlock)

		// write batch in a single transaction
		err := batchWriter.Write()
		if err != nil {
			// crash on database error, no risk of inconsistency here
			log.Crit("failed to write L1 messages to database", "err", err)
		}

		s.latestProcessedBlock = lastBlock
		batchWriter.Reset()
		numBlocksPendingDbWrite = 0
	}

	// query in batches
	for from := s.latestProcessedBlock + 1; from <= latestConfirmed; from += DefaultFetchBlockRange {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		to := from + DefaultFetchBlockRange - 1

		if to > latestConfirmed {
			to = latestConfirmed
		}

		msgs, err := s.client.fetchMessagesInRange(s.ctx, from, to)
		if err != nil {
			// flush pending writes to database
			if from > 0 {
				flush(from - 1)
			}

			log.Warn("failed to fetch L1 messages in range", "fromBlock", from, "toBlock", to, "err", err)
			return
		}

		if len(msgs) > 0 {
			log.Info("Received new L1 events", "fromBlock", from, "toBlock", to, "count", len(msgs))
			rawdb.WriteL1Messages(batchWriter, msgs) // collect messages in memory
		}

		numBlocksPendingDbWrite += to - from

		// flush new messages to database periodically
		if to == latestConfirmed || batchWriter.ValueSize() >= DbWriteThresholdBytes || numBlocksPendingDbWrite >= DbWriteThresholdBlocks {
			flush(to)
		}
	}
}
