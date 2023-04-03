package sync_service

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
)

const FetchLimit = uint64(20)
const PollInterval = time.Second * 15

type SyncService struct {
	bc                    *core.BlockChain
	cancel                context.CancelFunc
	client                *ethclient.Client
	ctx                   context.Context
	db                    ethdb.Database
	latestProcessedBlock  uint64
	pollInterval          time.Duration
	L1MessageQueueAddress *common.Address
}

func NewSyncService(ctx context.Context, config *params.ChainConfig, nodeConfig *node.Config, bc *core.BlockChain, db ethdb.Database) (*SyncService, error) {
	if bc == nil {
		return nil, errors.New("must pass BlockChain to SyncService")
	}

	client, err := ethclient.Dial(nodeConfig.L1Endpoint)
	if err != nil {
		return nil, err
	}

	// sanity check: compare chain IDs
	chainId, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}
	if chainId.Uint64() != config.L1Config.L1ChainId {
		return nil, fmt.Errorf("unexpected chain ID, expected = %v, got = %v", config.L1Config.L1ChainId, chainId)
	}

	// restart from latest synced block number
	latestProcessedBlock := rawdb.ReadSyncedL1BlockNumber(db)
	if latestProcessedBlock == nil {
		latestProcessedBlock = big.NewInt(0).Sub(nodeConfig.L1DeploymentBlock, big.NewInt(1))
	}

	ctx, cancel := context.WithCancel(ctx)

	service := SyncService{
		bc:                    bc,
		cancel:                cancel,
		client:                client,
		ctx:                   ctx,
		db:                    db,
		latestProcessedBlock:  latestProcessedBlock.Uint64(), // TODO
		pollInterval:          PollInterval,
		L1MessageQueueAddress: config.L1Config.L1MessageQueueAddress,
	}

	return &service, nil
}

func (s *SyncService) Start() {
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

func (s *SyncService) Stop() error {
	log.Info("Stopping sync service")

	if s.cancel != nil {
		defer s.cancel()
	}
	return nil
}

func (s *SyncService) fetchMessages() error {
	// TODO
	latestConfirmed, err := s.client.BlockNumber(s.ctx)
	if err != nil {
		log.Warn("eth_blockNumber failed", "err", err)
		return nil
	}

	// query in batches
	for from := s.latestProcessedBlock + 1; from <= latestConfirmed; from += FetchLimit {
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}

		to := from + FetchLimit - 1

		if to > latestConfirmed {
			to = latestConfirmed
		}

		msgs, err := s.fetchMessagesInRange(from, to)
		if err != nil {
			return err
		}

		s.StoreMessages(msgs)

		s.latestProcessedBlock = to
		s.SetLatestSyncedL1BlockNumber(to)
	}

	return nil
}

func (s *SyncService) fetchMessagesInRange(from, to uint64) ([]types.L1MessageTx, error) {
	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(0).SetUint64(from),
		ToBlock:   big.NewInt(0).SetUint64(to),
		Addresses: []common.Address{
			*s.L1MessageQueueAddress,
		},
		Topics: [][]common.Hash{
			{L1QueueTransactionEventSignature},
		},
	}

	logs, err := s.client.FilterLogs(s.ctx, query)
	if err != nil {
		log.Warn("eth_getLogs failed", "err", err)
		return nil, err
	}

	if len(logs) == 0 {
		return nil, nil
	}

	log.Info("Received new L1 events", "fromBlock", from, "toBlock", to, "count", len(logs))

	msgs, err := s.parseLogs(logs)
	if err != nil {
		log.Error("Failed to parse emitted events logs", "err", err)
		return nil, err
	}

	return msgs, nil
}

func (s *SyncService) parseLogs(logs []types.Log) ([]types.L1MessageTx, error) {
	var msgs []types.L1MessageTx

	for _, vLog := range logs {
		event := L1QueueTransactionEvent{}
		err := UnpackLog(L1MessageQueueABI, &event, "QueueTransaction", vLog)
		if err != nil {
			log.Warn("Failed to unpack L1 QueueTransaction event", "err", err)
			return msgs, err
		}

		msgs = append(msgs, types.L1MessageTx{
			Nonce:  event.QueueIndex.Uint64(),
			Gas:    event.GasLimit.Uint64(),
			To:     &event.Target,
			Value:  event.Value,
			Data:   event.Data,
			Sender: &event.Sender,
		})
	}

	return msgs, nil
}

func (s *SyncService) SetLatestSyncedL1BlockNumber(number uint64) {
	rawdb.WriteSyncedL1BlockNumber(s.db, big.NewInt(0).SetUint64(number))
}

func (s *SyncService) StoreMessages(msgs []types.L1MessageTx) {
	if len(msgs) > 0 {
		rawdb.WriteL1Messages(s.db, msgs)
	}
}
