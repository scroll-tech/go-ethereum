package sync_service

import (
	"context"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
)

const DefaultFetchBlockRange = uint64(20)
const DefaultPollInterval = time.Second * 15

type SyncService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	client               *BridgeClient
	db                   ethdb.Database
	pollInterval         time.Duration
	latestProcessedBlock uint64
}

func NewSyncService(ctx context.Context, genesisConfig *params.ChainConfig, nodeConfig *node.Config, db ethdb.Database) (*SyncService, error) {
	client, err := newBridgeClient(ctx, nodeConfig.L1Endpoint, genesisConfig.L1Config.L1ChainId, nodeConfig.L1Confirmations, genesisConfig.L1Config.L1MessageQueueAddress)
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
			log.Warn("failed to fetch messages in range", "err", err)
			return
		}

		if len(msgs) > 0 {
			log.Info("Received new L1 events", "fromBlock", from, "toBlock", to, "count", len(msgs))
		}

		s.StoreMessages(msgs)

		s.latestProcessedBlock = to
		s.SetLatestSyncedL1BlockNumber(to)
	}
}

func (s *SyncService) SetLatestSyncedL1BlockNumber(number uint64) {
	rawdb.WriteSyncedL1BlockNumber(s.db, number)
}

func (s *SyncService) StoreMessages(msgs []types.L1MessageTx) {
	if len(msgs) > 0 {
		rawdb.WriteL1Messages(s.db, msgs)
	}
}
