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
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
)

const (
	// DefaultFetchBlockRange is the number of blocks that we collect in a single eth_getLogs query.
	DefaultFetchBlockRange = uint64(100)

	// DefaultPollInterval is the frequency at which we query for new L1 messages.
	DefaultPollInterval = time.Second * 10

	// LogProgressInterval is the frequency at which we log progress.
	LogProgressInterval = time.Second * 10

	// DbWriteThresholdBytes is the size of batched database writes in bytes.
	DbWriteThresholdBytes = 10 * 1024

	// DbWriteThresholdBlocks is the number of blocks scanned after which we write to the database
	// even if we have not collected DbWriteThresholdBytes bytes of data yet. This way, if there is
	// a long section of L1 blocks with no messages and we stop or crash, we will not need to re-scan
	// this secion.
	DbWriteThresholdBlocks = 1000

	// MaxNumCachedL1BlocksTx is the capacity of the L1BlocksTx pool
	MaxNumCachedL1BlocksTx = 100
)

var (
	l1MessageTotalCounter = metrics.NewRegisteredCounter("rollup/l1/message", nil)
)

type L1BlocksTx struct {
	tx          *types.SystemTx
	blockNumber uint64
}

type L1BlocksPool struct {
	enabled              bool
	latestL1BlockNumOnL2 uint64
	l1BlocksTxs          []L1BlocksTx // The L1Blocks txs to be included in the blocks
	pendingL1BlocksTxs   []L1BlocksTx // The L1Blocks txs that are pending confirmation in the blocks
	l1BlocksFeed         event.Feed
}

// SyncService collects all L1 messages and stores them in a local database.
type SyncService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	client               *BridgeClient
	db                   ethdb.Database
	msgCountFeed         event.Feed
	pollInterval         time.Duration
	latestProcessedBlock uint64
	l1BlocksPool         L1BlocksPool
	scope                event.SubscriptionScope
}

func NewSyncService(ctx context.Context, genesisConfig *params.ChainConfig, nodeConfig *node.Config, db ethdb.Database, bc *core.BlockChain, l1Client EthClient) (*SyncService, error) {
	// terminate if the caller does not provide an L1 client (e.g. in tests)
	if l1Client == nil || (reflect.ValueOf(l1Client).Kind() == reflect.Ptr && reflect.ValueOf(l1Client).IsNil()) {
		log.Warn("No L1 client provided, L1 sync service will not run")
		return nil, nil
	}

	if genesisConfig.Scroll.L1Config == nil {
		return nil, fmt.Errorf("missing L1 config in genesis")
	}

	client, err := newBridgeClient(ctx, l1Client, genesisConfig.Scroll.L1Config.L1ChainId, nodeConfig.L1Confirmations, genesisConfig.Scroll.L1Config.L1MessageQueueAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bridge client: %w", err)
	}

	// assume deployment block has 0 messages
	latestProcessedBlock := nodeConfig.L1DeploymentBlock
	block := rawdb.ReadSyncedL1BlockNumber(db)
	if block != nil {
		// restart from latest synced block number
		latestProcessedBlock = *block
	}

	// read the latestL1BlockNumOnL2 from the L1Blocks contract
	state, err := bc.StateAt(bc.CurrentBlock().Root())
	if err != nil {
		return nil, fmt.Errorf("cannot get the state db: %w", err)
	}
	latestL1BlockNumOnL2 := state.GetState(rcfg.L1BlocksAddress, rcfg.LatestBlockNumberSlot).Big().Uint64()

	ctx, cancel := context.WithCancel(ctx)

	service := SyncService{
		ctx:                  ctx,
		cancel:               cancel,
		client:               client,
		db:                   db,
		pollInterval:         DefaultPollInterval,
		latestProcessedBlock: latestProcessedBlock,
		l1BlocksPool: L1BlocksPool{
			enabled:              genesisConfig.Scroll.SystemTx.Enabled,
			latestL1BlockNumOnL2: latestL1BlockNumOnL2,
		},
	}

	return &service, nil
}

func (s *SyncService) Start() {
	if s == nil {
		return
	}

	// wait for initial sync before starting node
	log.Info("Starting L1 message sync service", "latestProcessedBlock", s.latestProcessedBlock)

	// block node startup during initial sync and print some helpful logs
	latestConfirmed, err := s.client.getLatestConfirmedBlockNumber(s.ctx)
	if err == nil && latestConfirmed > s.latestProcessedBlock+1000 {
		log.Warn("Running initial sync of L1 messages before starting l2geth, this might take a while...")
		s.fetchMessages()
		log.Info("L1 message initial sync completed", "latestProcessedBlock", s.latestProcessedBlock)
	}

	go func() {
		t := time.NewTicker(s.pollInterval)
		defer t.Stop()

		for {
			// don't wait for ticker during startup
			s.fetchMessages()

			select {
			case <-s.ctx.Done():
				return
			case <-t.C:
				continue
			}
		}
	}()
}

func (s *SyncService) Stop() {
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

// SubscribeNewL1MsgsEvent registers a subscription of NewL1MsgsEvent and
// starts sending event to the given channel.
func (s *SyncService) SubscribeNewL1MsgsEvent(ch chan<- core.NewL1MsgsEvent) event.Subscription {
	return s.scope.Track(s.msgCountFeed.Subscribe(ch))
}

func (s *SyncService) SubscribeNewL1BlocksTx(ch chan<- core.NewL1MsgsEvent) event.Subscription {
	return s.scope.Track(s.l1BlocksPool.l1BlocksFeed.Subscribe(ch))
}

func (s *SyncService) CollectL1BlocksTxs(latestL1BlockNumberOnL2, maxNumTxs uint64) []*types.SystemTx {
	if len(s.l1BlocksPool.pendingL1BlocksTxs) > 0 {
		// pop the txs from the pending txs in the pool
		i := 0
		for ; i < len(s.l1BlocksPool.pendingL1BlocksTxs); i++ {
			if s.l1BlocksPool.pendingL1BlocksTxs[i].blockNumber > latestL1BlockNumberOnL2 {
				break
			}
		}
		failedTxs := s.l1BlocksPool.pendingL1BlocksTxs[i:]
		if len(failedTxs) > 0 {
			log.Warn("Failed to process L1Blocks txs", "cnt", len(failedTxs))
			s.l1BlocksPool.l1BlocksTxs = append(failedTxs, s.l1BlocksPool.l1BlocksTxs...)
		}
		s.l1BlocksPool.pendingL1BlocksTxs = nil
	}

	cnt := int(maxNumTxs)
	if cnt > len(s.l1BlocksPool.l1BlocksTxs) {
		cnt = len(s.l1BlocksPool.l1BlocksTxs)
	}
	ret := make([]*types.SystemTx, cnt)
	for i := 0; i < cnt; i++ {
		ret[i] = s.l1BlocksPool.l1BlocksTxs[i].tx
	}
	s.l1BlocksPool.pendingL1BlocksTxs = s.l1BlocksPool.l1BlocksTxs[:cnt]
	s.l1BlocksPool.l1BlocksTxs = s.l1BlocksPool.l1BlocksTxs[cnt:]

	return ret
}

func (s *SyncService) fetchMessages() {
	latestConfirmed, err := s.client.getLatestConfirmedBlockNumber(s.ctx)
	if err != nil {
		log.Warn("Failed to get latest confirmed block number", "err", err)
		return
	}

	if s.l1BlocksPool.enabled {
		s.fetchL1Blocks(latestConfirmed)
	}

	s.fetchL1Messages(latestConfirmed)
}

func (s *SyncService) fetchL1Blocks(latestConfirmed uint64) {
	var latestProcessedBlock uint64
	if len(s.l1BlocksPool.l1BlocksTxs) != 0 {
		latestProcessedBlock = s.l1BlocksPool.l1BlocksTxs[len(s.l1BlocksPool.l1BlocksTxs)-1].blockNumber
	} else {
		latestProcessedBlock = s.l1BlocksPool.latestL1BlockNumOnL2
	}
	if latestProcessedBlock == 0 {
		// This is the first L1 blocks system tx on L2, only fetch the latest confirmed L1 block
		latestProcessedBlock = latestConfirmed - 1
	}
	// query in batches
	num := 0
	for from := latestProcessedBlock + 1; from <= latestConfirmed; from += DefaultFetchBlockRange {
		cnt := DefaultFetchBlockRange
		if cnt+uint64(len(s.l1BlocksPool.l1BlocksTxs)) > MaxNumCachedL1BlocksTx {
			cnt = MaxNumCachedL1BlocksTx - uint64(len(s.l1BlocksPool.l1BlocksTxs))
		}
		to := from + cnt - 1
		if to > latestConfirmed {
			to = latestConfirmed
		}
		l1BlocksTxs, err := s.client.fetchL1Blocks(s.ctx, from, to)
		if err != nil {
			log.Warn("Failed to fetch L1Blocks in range", "fromBlock", from, "toBlock", to, "err", err)
			return
		}
		log.Debug("Received new L1 blocks", "fromBlock", from, "toBlock", to, "count", len(l1BlocksTxs))
		for i, tx := range l1BlocksTxs {
			s.l1BlocksPool.l1BlocksTxs = append(s.l1BlocksPool.l1BlocksTxs, L1BlocksTx{tx, from + uint64(i)})
		}
		num += len(l1BlocksTxs)
	}

	if num > 0 {
		s.l1BlocksPool.l1BlocksFeed.Send(core.NewL1MsgsEvent{Count: num})
	}
}

func (s *SyncService) fetchL1Messages(latestConfirmed uint64) {
	log.Trace("Sync service fetchMessages", "latestProcessedBlock", s.latestProcessedBlock, "latestConfirmed", latestConfirmed)

	// keep track of next queue index we're expecting to see
	queueIndex := rawdb.ReadHighestSyncedQueueIndex(s.db)

	batchWriter := s.db.NewBatch()
	numBlocksPendingDbWrite := uint64(0)
	numMessagesPendingDbWrite := 0

	// helper function to flush database writes cached in memory
	flush := func(lastBlock uint64) {
		// update sync progress
		rawdb.WriteSyncedL1BlockNumber(batchWriter, lastBlock)

		// write batch in a single transaction
		err := batchWriter.Write()
		if err != nil {
			// crash on database error, no risk of inconsistency here
			log.Crit("Failed to write L1 messages to database", "err", err)
		}

		batchWriter.Reset()
		numBlocksPendingDbWrite = 0

		if numMessagesPendingDbWrite > 0 {
			l1MessageTotalCounter.Inc(int64(numMessagesPendingDbWrite))
			s.msgCountFeed.Send(core.NewL1MsgsEvent{Count: numMessagesPendingDbWrite})
			numMessagesPendingDbWrite = 0
		}

		s.latestProcessedBlock = lastBlock
	}

	// ticker for logging progress
	t := time.NewTicker(LogProgressInterval)
	numMsgsCollected := 0

	// query in batches
	for from := s.latestProcessedBlock + 1; from <= latestConfirmed; from += DefaultFetchBlockRange {
		select {
		case <-s.ctx.Done():
			// flush pending writes to database
			if from > 0 {
				flush(from - 1)
			}
			return
		case <-t.C:
			progress := 100 * float64(s.latestProcessedBlock) / float64(latestConfirmed)
			log.Info("Syncing L1 messages", "processed", s.latestProcessedBlock, "confirmed", latestConfirmed, "collected", numMsgsCollected, "progress(%)", progress)
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
			log.Warn("Failed to fetch L1 messages in range", "fromBlock", from, "toBlock", to, "err", err)
			return
		}

		if len(msgs) > 0 {
			log.Debug("Received new L1 events", "fromBlock", from, "toBlock", to, "count", len(msgs))
			rawdb.WriteL1Messages(batchWriter, msgs) // collect messages in memory
			numMsgsCollected += len(msgs)
		}

		for _, msg := range msgs {
			if msg.QueueIndex > 0 {
				queueIndex++
			}
			// check if received queue index matches expected queue index
			if msg.QueueIndex != queueIndex {
				log.Error("Unexpected queue index in SyncService", "expected", queueIndex, "got", msg.QueueIndex, "msg", msg)
				return // do not flush inconsistent data to disk
			}
		}

		numBlocksPendingDbWrite += to - from + 1
		numMessagesPendingDbWrite += len(msgs)

		// flush new messages to database periodically
		if to == latestConfirmed || batchWriter.ValueSize() >= DbWriteThresholdBytes || numBlocksPendingDbWrite >= DbWriteThresholdBlocks {
			flush(to)
		}
	}
}
