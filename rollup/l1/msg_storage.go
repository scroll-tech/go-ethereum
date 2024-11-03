package l1

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

const (
	defaultFetchInterval   = 5 * time.Second
	maxMsgsStored          = 2000
	defaultFetchBlockRange = uint64(500)
)

type MsgStorageState struct {
	StartBlockHeader *types.Header
	EndBlockHeader   *types.Header
}

type MsgStorage struct {
	state MsgStorageState

	ctx    context.Context
	cancel context.CancelFunc

	db                    ethdb.Database
	msgs                  *common.ShrinkingMap[uint64, storedL1Message]
	reader                *Reader
	tracker               *Tracker
	confirmationRule      ConfirmationRule
	unsubscribeTracker    func()
	newChainNotifications chan newChainNotification
	latestFinalized       *types.Header

	msgsMu sync.RWMutex
}

func NewMsgStorage(ctx context.Context, tracker *Tracker, reader *Reader, db ethdb.Database, confirmationRule ConfirmationRule, l1DeploymentBlockHeader *types.Header) (*MsgStorage, error) {
	if tracker == nil || reader == nil {
		return nil, fmt.Errorf("failed to create MsgStorage, reader or tracker is nil")
	}
	ctx, cancel := context.WithCancel(ctx)
	msgStorage := &MsgStorage{
		ctx:                   ctx,
		cancel:                cancel,
		db:                    db,
		msgs:                  common.NewShrinkingMap[uint64, storedL1Message](maxMsgsStored),
		reader:                reader,
		tracker:               tracker,
		confirmationRule:      confirmationRule,
		newChainNotifications: make(chan newChainNotification, 10),
		state: MsgStorageState{
			StartBlockHeader: l1DeploymentBlockHeader,
			EndBlockHeader:   l1DeploymentBlockHeader,
		},
		latestFinalized: l1DeploymentBlockHeader,
	}
	return msgStorage, nil
}

func (ms *MsgStorage) Start() {
	log.Info("starting MsgStorage")

	data := rawdb.ReadL1MsgStorageState(ms.db)
	state := &MsgStorageState{}
	if err := rlp.Decode(bytes.NewReader(data), state); err == nil {
		ms.state = *state
		ms.resyncToState()
	}
	ms.unsubscribeTracker = ms.tracker.Subscribe(ms.confirmationRule, func(isOnline bool, old, new []*types.Header) bool {
		if isOnline {
			ms.newChainNotifications <- newChainNotification{old, new}
			return true
		} else {
			ms.latestFinalized = new[0]
			// check if latest synced block is already finalized or later -> can become an online subscriber
			if ms.state.EndBlockHeader.Number.Uint64() >= new[0].Number.Uint64() {
				return true
			}
		}
		return false
	})
	go func() {
		fetchTicker := time.NewTicker(defaultFetchInterval)
		defer fetchTicker.Stop()

		for {
			select {
			case <-ms.ctx.Done():
				return
			default:
			}
			select {
			case <-ms.ctx.Done():
				return
			case <-fetchTicker.C:
				if ms.state.EndBlockHeader.Number.Uint64() < ms.latestFinalized.Number.Uint64() {
					err := ms.syncToTheFinalizedOrLimit()
					if err != nil {
						log.Warn("MsgStorage: failed to fetch messages", "err", err)
					}
				} else if len(ms.newChainNotifications) != 0 {
					err := ms.syncRecentNotifs()
					if err != nil {
						log.Warn("MsgStorage: failed to fetch messages", "err", err)
					}
				}
			}

		}
	}()
}

// ReadL1Message retrieves the L1 message corresponding to the enqueue index.
func (ms *MsgStorage) ReadL1Message(queueIndex uint64) *types.L1MessageTx {
	ms.msgsMu.RLock()
	defer ms.msgsMu.RUnlock()
	msg, exists := ms.msgs.Get(queueIndex)
	if !exists {
		return nil
	}
	return msg.l1msg
}

// IterateL1MessagesFrom creates an L1MessageIterator that iterates over
// all L1 message in the MsgStorage starting at the provided enqueue index.
func (ms *MsgStorage) IterateL1MessagesFrom(fromQueueIndex uint64) L1MessageIterator {
	return L1MessageIterator{
		curIndex:   fromQueueIndex,
		msgStorage: ms,
	}
}

// ReadL1MessagesFrom retrieves up to `maxCount` L1 messages starting at `startIndex`.
func (ms *MsgStorage) ReadL1MessagesFrom(startIndex, maxCount uint64) []types.L1MessageTx {
	if maxCount == 0 {
		return []types.L1MessageTx{}
	}
	msgs := make([]types.L1MessageTx, 0, maxCount)

	for index := startIndex; len(msgs) < int(maxCount); index++ {
		storedL1Msg, exists := ms.msgs.Get(index)
		if !exists {
			break // No more messages to read
		}
		msg := storedL1Msg.l1msg
		// Sanity check for QueueIndex
		if msg.QueueIndex != index {
			log.Crit(
				"Unexpected QueueIndex in ReadL1MessagesFrom",
				"expected", index,
				"got", msg.QueueIndex,
				"startIndex", startIndex,
				"maxCount", maxCount,
			)
		}
		msgs = append(msgs, *msg)
	}
	return msgs
}

func (ms *MsgStorage) syncRange(from, to uint64) error {
	events, err := ms.reader.FetchL1MessageEventsInRange(from, to)
	if err != nil {
		return fmt.Errorf("failed to fetch l1 messages in range, start: %d, end: %d, err: %w", from, to, err)
	}
	msgsToStore := make([]storedL1Message, 0, len(events))
	for _, event := range events {
		msg := &types.L1MessageTx{
			QueueIndex: event.QueueIndex,
			Gas:        event.GasLimit.Uint64(),
			To:         &event.Target,
			Value:      event.Value,
			Data:       event.Data,
			Sender:     event.Sender,
		}
		msgsToStore = append(msgsToStore, storedL1Message{
			l1msg:      msg,
			headerHash: event.Raw.BlockHash,
		})
	}
	ms.msgsMu.Lock()
	for _, msg := range msgsToStore {
		ms.msgs.Set(msg.l1msg.QueueIndex, msg)
	}
	ms.msgsMu.Unlock()
	return nil
}

func (ms *MsgStorage) resyncToState() error {
	log.Info("MsgStorage: resync to state", "start block number", ms.state.StartBlockHeader.Number.Uint64(), "endBlockNumber", ms.state.EndBlockHeader.Number.Uint64())
	endBlockNumber := ms.state.EndBlockHeader.Number.Uint64()
	for from := ms.state.StartBlockHeader.Number.Uint64(); from <= endBlockNumber; from += defaultFetchBlockRange {
		to := from + defaultFetchBlockRange - 1
		if to > endBlockNumber {
			to = endBlockNumber
		}
		if err := ms.syncRange(from, to); err != nil {
			return err
		}
	}
	log.Info("MsgStorage: resync to state finished", "number of messages", ms.msgs.Size())

	// don't need to update state here, because we are syncing to match defined state
	return nil
}

func (ms *MsgStorage) syncToTheFinalizedOrLimit() error {
	if ms.msgs.Size() >= maxMsgsStored {
		return nil
	}
	for {
		from := ms.state.EndBlockHeader.Number.Uint64() + 1
		if from > ms.latestFinalized.Number.Uint64() {
			break
		}
		to := from + defaultFetchBlockRange
		if err := ms.syncRange(from, to); err != nil {
			return err
		}
		// update storage state
		toHeader, err := ms.reader.FetchBlockHeaderByNumber(to)
		if err != nil {
			return fmt.Errorf("failed to fetch block header by number %d, err: %w", to, err)
		}
		ms.state.EndBlockHeader = toHeader
		ms.updateStorageState()
		// if the number of stored l1 msgs exceeded the limit stop syncing. It will remain when some of the old messages pruned
		if ms.msgs.Size() >= maxMsgsStored {
			break
		}
	}
	return nil
}

func (ms *MsgStorage) syncRecentNotifs() error {
	var notifs []newChainNotification
out:
	for {
		select {
		case msg := <-ms.newChainNotifications:
			notifs = append(notifs, msg)
		default:
			break out
		}
	}

	// go through all chain notifications and process
	for _, newChainNotification := range notifs {
		old, new := newChainNotification.old, newChainNotification.new

		// check if there is old chain to delete l1msgs from
		if old != nil {
			// find msgs that come for reorged chain
			ms.msgsMu.RLock()
			msgs := ms.msgs.Values()
			ms.msgsMu.RUnlock()
			var indexesToDelete []uint64
			for _, msg := range msgs {
				contains := false
				for _, header := range old {
					if header.Hash() == msg.headerHash {
						contains = true
						break
					}
				}
				if contains {
					indexesToDelete = append(indexesToDelete, msg.l1msg.QueueIndex)
				}
			}
			if len(indexesToDelete) > 0 {
				ms.msgsMu.Lock()
				for _, index := range indexesToDelete {
					ms.msgs.Delete(index)
				}
				ms.msgsMu.Unlock()
			}
		}

		// load messages from new chain
		start := new[len(new)-1].Number.Uint64()
		end := new[0].Number.Uint64()
		if err := ms.syncRange(start, end); err != nil {
			return err
		}
		// update storage state
		ms.state.EndBlockHeader = new[0]
		ms.updateStorageState()
	}
	return nil
}

func (ms *MsgStorage) updateStorageState() {
	bytes, err := rlp.EncodeToBytes(ms.state)
	if err != nil {
		log.Crit("Failed to RLP encode message storage state", "err", err)
	}
	rawdb.WriteL1MsgStorageState(ms.db, bytes)
}

// PruneMessages deletes all messages that are older or equal to provided index
func (ms *MsgStorage) PruneMessages(lastIndex uint64) {
	log.Info("PruneMessages", "size", ms.msgs.Size(), "lastIndex", lastIndex)
	ms.msgsMu.Lock()
	defer ms.msgsMu.Unlock()

	// update storage state
	lastBlock, exists := ms.msgs.Get(lastIndex)
	fmt.Println(lastBlock, exists)
	if exists {
		header, err := ms.reader.FetchBlockHeaderByHash(lastBlock.headerHash)
		if err == nil {
			ms.state.StartBlockHeader = header
			ms.updateStorageState()
		}
	}

	deleted := ms.msgs.Delete(lastIndex)
	for deleted {
		lastIndex--
		deleted = ms.msgs.Delete(lastIndex)
	}
}

func (ms *MsgStorage) Stop() {
	log.Info("stopping MsgStorage")
	ms.unsubscribeTracker()
	ms.cancel()
	log.Info("MsgStorage stopped")
}

type storedL1Message struct {
	l1msg      *types.L1MessageTx
	headerHash common.Hash
}

type newChainNotification struct {
	old []*types.Header
	new []*types.Header
}

type L1MessageIterator struct {
	curIndex   uint64
	curMsg     *types.L1MessageTx
	msgStorage *MsgStorage
}

// Next moves the iterator to the next key/value pair.
// It returns false when there is no next L1Msg
func (it *L1MessageIterator) Next() bool {
	it.curMsg = it.msgStorage.ReadL1Message(it.curIndex)
	it.curIndex++
	if it.curMsg == nil {
		return false
	} else {
		return true
	}
}

// L1Message returns the current L1 message.
func (it *L1MessageIterator) L1Message() types.L1MessageTx {
	return *it.curMsg
}
