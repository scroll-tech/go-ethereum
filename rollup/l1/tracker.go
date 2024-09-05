package l1

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

type Tracker struct {
	ctx    context.Context
	cancel context.CancelFunc

	client Client

	headers             *common.ShrinkingMap[uint64, *types.Header]
	lastSafeHeader      *types.Header
	lastFinalizedHeader *types.Header

	subscriptionCounter int                                  // used to assign unique IDs to subscriptions
	subscriptions       map[ConfirmationRule][]*subscription // sorted by confirmationRule ascending
	mu                  sync.RWMutex
}

const (
	// defaultSyncInterval is the frequency at which we query for new chain head
	defaultSyncInterval = 12 * time.Second
)

func NewTracker(ctx context.Context, l1Client Client) *Tracker {
	ctx, cancel := context.WithCancel(ctx)
	l1Tracker := &Tracker{
		ctx:           ctx,
		cancel:        cancel,
		client:        l1Client,
		headers:       common.NewShrinkingMap[uint64, *types.Header](1000),
		subscriptions: make(map[ConfirmationRule][]*subscription),
	}

	return l1Tracker
}

func (t *Tracker) Start() {
	log.Info("starting Tracker")
	go func() {
		syncTicker := time.NewTicker(defaultSyncInterval)
		defer syncTicker.Stop()

		for {
			select {
			case <-t.ctx.Done():
				return
			default:
			}
			select {
			case <-t.ctx.Done():
				return
			case <-syncTicker.C:
				// TODO: also sync SafeChainHead and FinalizedChainHead
				err := t.syncLatestHead()
				if err != nil {
					log.Warn("Tracker: failed to sync latest head", "err", err)
				}
			}
		}
	}()
}

func (t *Tracker) headerByNumber(number rpc.BlockNumber) (*types.Header, error) {
	newHeader, err := t.client.HeaderByNumber(t.ctx, big.NewInt(int64(number)))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s header", number)
	}
	if !newHeader.Number.IsUint64() {
		return nil, fmt.Errorf("received unexpected block number in Tracker: %v", newHeader.Number)
	}

	return newHeader, nil
}

func (t *Tracker) syncLatestHead() error {
	newHeader, err := t.headerByNumber(rpc.LatestBlockNumber)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve latest header")
	}

	storedHeader, exists := t.headers.Get(newHeader.Number.Uint64())
	if exists {
		// We already processed the header, nothing to do.
		if storedHeader.Hash() == newHeader.Hash() {
			return nil
		}

		// Since we already processed a header at this height with different hash this means a L1 reorg happened.
		// TODO: reset cache.

		// Notify all subscribers to new LatestChainHead at their respective confirmation depth.
		err = t.notifyLatest(newHeader, true)
		if err != nil {
			return errors.Wrapf(err, "failed to notify subscribers of new latest header")
		}

		return nil
	}

	// Notify all subscribers to new LatestChainHead at their respective confirmation depth.
	err = t.notifyLatest(newHeader, false)
	if err != nil {
		return errors.Wrapf(err, "failed to notify subscribers of new latest header")
	}

	return nil
}

func (t *Tracker) notifyLatest(newHeader *types.Header, reorg bool) error {
	// TODO: add mutex for headers, lastSafeHeader, lastFinalizedHeader
	t.headers.Set(newHeader.Number.Uint64(), newHeader)

	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, sub := range t.subscriptions[LatestChainHead] {
		// Ignore subscriptions with deeper ConfirmationRule than the new block.
		if newHeader.Number.Uint64() < uint64(sub.confirmationRule) {
			continue
		}

		// 1 confirmation == latest block
		// 2 confirmations == latest block - 1
		// ...
		// n confirmations == latest block - (n-1)
		depth := uint64(sub.confirmationRule - 1)
		headerToNotifyNumber := newHeader.Number.Uint64() - depth
		headerToNotify, exists := t.headers.Get(headerToNotifyNumber)
		if !exists {
			// This might happen if there's a gap in the headers cache. We need to fetch the header from the RPC node.
			h, err := t.headerByNumber(rpc.BlockNumber(headerToNotifyNumber))
			if err != nil {
				return errors.Wrapf(err, "failed to retrieve latest header")
			}
			headerToNotify = h
			t.headers.Set(h.Number.Uint64(), h)
		}

		if reorg && sub.lastSentHeader != nil {
			// The subscriber is subscribed to a deeper ConfirmationRule than the reorg depth -> this reorg doesn't affect the subscriber.
			// Since the subscribers are sorted by ConfirmationRule, we can return here.
			if sub.lastSentHeader.Number.Uint64() < headerToNotify.Number.Uint64() {
				return nil
			}

			// We already sent this header to the subscriber. This shouldn't happen here since we're handling a reorg and
			// by definition the last sent header should be different from the header we're notifying about if the header number is the same.
			if sub.lastSentHeader.Hash() == headerToNotify.Hash() {
				continue
			}
		}

		sub.callback(sub.lastSentHeader, headerToNotify, reorg)
		sub.lastSentHeader = headerToNotify
	}

	return nil
}

func (t *Tracker) syncSafeHead() error {
	newHeader, err := t.headerByNumber(rpc.SafeBlockNumber)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve safe header")
	}

	if t.lastSafeHeader != nil {
		// We already saw this header, nothing to do.
		if t.lastSafeHeader.Hash() == newHeader.Hash() {
			return nil
		}

		// This means there was a L1 reorg and the safe block changed. While this is possible, it should be very rare.
		if t.lastSafeHeader.Number.Uint64() >= newHeader.Number.Uint64() {
			t.notifySafeHead(newHeader, true)
			return nil
		}
	}

	// Notify all subscribers to new SafeChainHead.
	t.notifySafeHead(newHeader, false)

	return nil
}

func (t *Tracker) notifySafeHead(newHeader *types.Header, reorg bool) {
	t.lastSafeHeader = newHeader

	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, sub := range t.subscriptions[SafeChainHead] {
		sub.callback(sub.lastSentHeader, newHeader, reorg)
		sub.lastSentHeader = newHeader
	}
}

func (t *Tracker) syncFinalizedHead() error {
	newHeader, err := t.headerByNumber(rpc.FinalizedBlockNumber)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve safe header")
	}

	if t.lastFinalizedHeader != nil {
		// We already saw this header, nothing to do.
		if t.lastFinalizedHeader.Hash() == newHeader.Hash() {
			return nil
		}

		// This means the finalized block changed as read from L1. The Ethereum protocol guarantees that this can never
		// happen. Must be some issue with the RPC node.
		if t.lastFinalizedHeader.Number.Uint64() >= newHeader.Number.Uint64() {
			log.Crit("RPC node faulty: finalized block number decreased", "old", t.lastFinalizedHeader.Number, "new", newHeader.Number, "old hash", t.lastFinalizedHeader.Hash(), "new hash", newHeader.Hash())
		}
	}

	t.notifyFinalizedHead(newHeader)

	// TODO: prune old headers

	return nil
}

func (t *Tracker) notifyFinalizedHead(newHeader *types.Header) {
	t.lastFinalizedHeader = newHeader

	t.mu.RLock()
	defer t.mu.RUnlock()

	// Notify all subscribers to new FinalizedChainHead.
	for _, sub := range t.subscriptions[FinalizedChainHead] {
		sub.callback(sub.lastSentHeader, newHeader, false)
		sub.lastSentHeader = newHeader
	}
}

func (t *Tracker) Subscribe(confirmationRule ConfirmationRule, callback SubscriptionCallback) (unsubscribe func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Validate ConfirmationRule configuration. Invalid rules will cause a panic as it is a programming error.
	var confirmationType ConfirmationRule
	switch {
	case confirmationRule == FinalizedChainHead:
		confirmationType = FinalizedChainHead
	case confirmationRule == SafeChainHead:
		confirmationType = SafeChainHead
	case confirmationRule >= LatestChainHead && confirmationRule <= maxConfirmationRule:
		confirmationType = LatestChainHead
	default:
		log.Crit("invalid confirmation rule", "confirmationRule", confirmationRule)
	}

	sub := newSubscription(t.subscriptionCounter, confirmationRule, callback)

	subscriptionsByType := t.subscriptions[confirmationType]
	subscriptionsByType = append(subscriptionsByType, sub)

	slices.SortFunc(subscriptionsByType, func(a *subscription, b *subscription) int {
		if a.confirmationRule > b.confirmationRule {
			return 1
		} else if a.confirmationRule < b.confirmationRule {
			return -1
		} else {
			// IDs are unique and monotonically increasing, therefore there is always a clear order.
			if a.id > b.id {
				return 1
			} else {
				return -1
			}
		}
	})

	t.subscriptions[confirmationType] = subscriptionsByType
	t.subscriptionCounter++

	return func() {
		t.mu.Lock()
		defer t.mu.Unlock()

		for i, s := range t.subscriptions[sub.confirmationRule] {
			if s.id == sub.id {
				subscriptionsByType = append(subscriptionsByType[:i], subscriptionsByType[i+1:]...)
				break
			}
		}
		t.subscriptions[confirmationRule] = subscriptionsByType
	}
}

func (t *Tracker) Stop() {
	log.Info("stopping Tracker")
	t.cancel()
	log.Info("Tracker stopped")
}

type subscription struct {
	id               int
	confirmationRule ConfirmationRule
	callback         SubscriptionCallback
	lastSentHeader   *types.Header
}

func newSubscription(id int, confirmationRule ConfirmationRule, callback SubscriptionCallback) *subscription {
	return &subscription{
		id:               id,
		confirmationRule: confirmationRule,
		callback:         callback,
	}
}

type ConfirmationRule int8

// maxConfirmationRule is the maximum number of confirmations we can subscribe to.
// This is equal to the best case scenario where Ethereum L1 is finalizing 2 epochs in the past (64 blocks).
const maxConfirmationRule = ConfirmationRule(64)

const (
	FinalizedChainHead = ConfirmationRule(-2)
	SafeChainHead      = ConfirmationRule(-1)
	LatestChainHead    = ConfirmationRule(1)
)

type SubscriptionCallback func(last, new *types.Header, reorg bool)
