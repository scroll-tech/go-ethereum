package l1

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

const (
	defaultFetchInterval = 5 * time.Second
)

type MsgStorageState struct {
	StartBlockHeader *types.Header
	EndBlockHeader   *types.Header
}

type MsgStorage struct {
	state MsgStorageState

	ctx    context.Context
	cancel context.CancelFunc

	msgs                  *common.ShrinkingMap[uint64, storedL1Message]
	reader                *Reader
	tracker               *Tracker
	confirmationRule      ConfirmationRule
	unsubscribeTracker    func()
	newChainNotifications chan newChainNotification

	msgsMu sync.RWMutex
}

func NewMsgStorage(ctx context.Context, tracker *Tracker, reader *Reader, confirmationRule ConfirmationRule) (*MsgStorage, error) {
	if tracker == nil || reader == nil {
		return nil, fmt.Errorf("failed to create MsgStorage, reader or tracker is nil")
	}
	ctx, cancel := context.WithCancel(ctx)
	msgStorage := &MsgStorage{
		ctx:                   ctx,
		cancel:                cancel,
		msgs:                  common.NewShrinkingMap[uint64, storedL1Message](1000),
		reader:                reader,
		tracker:               tracker,
		confirmationRule:      confirmationRule,
		newChainNotifications: make(chan newChainNotification, 10),
	}
	return msgStorage, nil
}

func (ms *MsgStorage) Start() {
	log.Info("starting MsgStorage")
	ms.unsubscribeTracker = ms.tracker.Subscribe(ms.confirmationRule, func(old, new []*types.Header) {
		ms.newChainNotifications <- newChainNotification{old, new}
	}, 64)
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
				if len(ms.newChainNotifications) != 0 {
					err := ms.fetchMessages()
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

func (ms *MsgStorage) fetchMessages() error {
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
		events, err := ms.reader.FetchL1MessageEventsInRange(start, end)
		if err != nil {
			return fmt.Errorf("failed to fetch l1 messages in range, start: %d, end: %d, err: %w", start, end, err)
		}
		msgsToStore := make([]storedL1Message, len(events))
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
		// update storage state
		ms.state.EndBlockHeader = new[0]
		if ms.state.StartBlockHeader == nil {
			ms.state.StartBlockHeader = new[len(new)-1]
		}
	}
	return nil
}

// PruneMessages deletes all messages that are older or equal to provided index
func (ms *MsgStorage) PruneMessages(lastIndex uint64) {
	ms.msgsMu.Lock()
	defer ms.msgsMu.Unlock()

	// todo: update state for graceful restart
	deleted := ms.msgs.Delete(lastIndex)
	for deleted {
		lastIndex--
		deleted = ms.msgs.Delete(lastIndex)
	}
}

func (ms *MsgStorage) Stop() {
	log.Info("stopping MsgStorage")
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
