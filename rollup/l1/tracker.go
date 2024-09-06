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
		return nil, errors.Wrapf(err, "failed to get %s header by number", number)
	}
	if !newHeader.Number.IsUint64() {
		return nil, fmt.Errorf("received unexpected block number in Tracker: %v", newHeader.Number)
	}

	return newHeader, nil
}

func (t *Tracker) headerByHash(hash common.Hash) (*types.Header, error) {
	newHeader, err := t.client.HeaderByHash(t.ctx, hash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s header by hash", hash)
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

	reorged := newReorgedHeaders()

	seenHeader, exists := t.headers.Get(newHeader.Number.Uint64())
	if exists {
		if seenHeader.Hash() == newHeader.Hash() {
			// We already saw this header, nothing to do.
			return nil
		}

		// L1 reorg of (at least) depth 1 as the seenHeader.Hash() != newHeader.Hash().
		reorged.add(seenHeader)
	}

	// Make sure that we have a continuous sequence of headers (chain) in the cache.
	// If there's a gap, we need to fetch the missing headers.
	// A gap can happen due to a reorg or because the node was offline for a while/the RPC didn't return the headers.
	// TODO: what about genesis block?
	current := newHeader
	for newHeader.Number.Uint64() > 1 {
		prevNumber := current.Number.Uint64() - 1
		prevHeader, exists := t.headers.Get(prevNumber)

		if !exists {
			// There's a gap. We need to fetch the previous header.
			prev, err := t.headerByNumber(rpc.BlockNumber(prevNumber))
			if err != nil {
				return errors.Wrapf(err, "failed to retrieve previous header %d", current.Number.Uint64()-1)
			}

			t.headers.Set(prev.Number.Uint64(), prev)
			prevHeader = prev
		}

		// Make sure that the headers are connected in a chain.
		if current.ParentHash == prevHeader.Hash() {
			// We already had this header in cache, this means the chain is complete.
			if exists {
				break
			}

			// We had a gap in the chain. Continue fetching the previous headers.
			current = prevHeader
			continue
		}

		// L1 reorg as the current.ParentHash != prevHeader.Hash(). We need to fetch the new chain.
		newChainPrev, err := t.headerByHash(current.ParentHash)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve new chain previous header")
		}

		// sanity check - should never happen
		if newChainPrev.Number.Uint64() != prevNumber {
			return errors.Errorf("new chain previous header number %d does not match expected number %d", newChainPrev.Number.Uint64(), prevNumber)
		}

		// we need to store the reorged headers to notify the subscribers about the reorg.
		reorged.add(prevHeader)

		// update the headers cache with the new chain
		t.headers.Set(newChainPrev.Number.Uint64(), newChainPrev)

		// continue reconciling the new chain - stops when we reach the forking point
		current = newChainPrev
	}

	// TODO: reset cache beyond newHeader.Number if there was a reorg
	//  might not be necessary if we always wait for longest-chain to take over.
	if !reorged.isEmpty() {

	}

	// Notify all subscribers to new LatestChainHead at their respective confirmation depth.
	err = t.notifyLatest(newHeader, reorged)
	if err != nil {
		return errors.Wrapf(err, "failed to notify subscribers of new latest header")
	}

	return nil
}

func (t *Tracker) notifyLatest(newHeader *types.Header, reorged *reorgedHeaders) error {
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
			// This should never happen since we're making sure that the headers are continuous.
			return errors.Errorf("failed to find header %d in cache", headerToNotifyNumber)
		}

		var reorg bool
		if sub.lastSentHeader != nil {
			// We already sent this header to the subscriber. Nothing to do.
			if sub.lastSentHeader.Hash() == headerToNotify.Hash() {
				continue
			}

			// We are in a reorg. Check if the subscriber is affected by the reorg.
			if !reorged.isEmpty() {
				for _, reorgedHeader := range reorged.headers {
					fmt.Println("reorged headers", reorgedHeader.Number.Uint64())
				}
				fmt.Println("reorged min", reorged.minNumber)
				fmt.Println("reorged max", reorged.maxNumber)
				fmt.Println("sub last sent header", sub.lastSentHeader.Number.Uint64())

				if sub.lastSentHeader.Number.Uint64() < reorged.minNumber {
					// The subscriber is subscribed to a deeper ConfirmationRule than the reorg depth -> this reorg doesn't affect the subscriber.
					reorg = false
				} else {
					// The subscriber is affected by the reorg.
					reorg = true
				}
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
			return errors.Errorf("RPC node faulty: finalized block number decreased from %d to %d", t.lastFinalizedHeader.Number.Uint64(), newHeader.Number.Uint64())
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
		panic(fmt.Sprintf("invalid confirmation rule %d", confirmationRule))
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
