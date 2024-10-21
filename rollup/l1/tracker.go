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

	genesis        common.Hash
	canonicalChain *common.ShrinkingMap[uint64, *types.Header]
	headers        *common.ShrinkingMap[common.Hash, *types.Header]

	lastSafeHeader      *types.Header
	lastFinalizedHeader *types.Header

	subscriptionCounter int                                  // used to assign unique IDs to subscriptions
	subscriptions       map[ConfirmationRule][]*subscription // sorted by confirmationRule ascending
	mu                  sync.RWMutex
}

const (
	// defaultSyncInterval is the frequency at which we query for new chain head
	defaultSyncInterval  = 12 * time.Second
	defaultPruneInterval = 60 * time.Second
)

func NewTracker(ctx context.Context, l1Client Client, genesis common.Hash) *Tracker {
	ctx, cancel := context.WithCancel(ctx)
	l1Tracker := &Tracker{
		ctx:            ctx,
		cancel:         cancel,
		client:         l1Client,
		genesis:        genesis,
		canonicalChain: common.NewShrinkingMap[uint64, *types.Header](1000),
		headers:        common.NewShrinkingMap[common.Hash, *types.Header](1000),
		subscriptions:  make(map[ConfirmationRule][]*subscription),
	}

	return l1Tracker
}

func (t *Tracker) Start() {
	log.Info("starting Tracker")
	go func() {
		syncTicker := time.NewTicker(defaultSyncInterval)
		defer syncTicker.Stop()
		pruneTicker := time.NewTicker(defaultPruneInterval)
		defer pruneTicker.Stop()
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
			case <-pruneTicker.C:
				t.pruneOldHeaders()
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

	// Store the header in cache.
	t.headers.Set(newHeader.Hash(), newHeader)

	return newHeader, nil
}

func (t *Tracker) headerByHash(hash common.Hash) (*types.Header, error) {
	// Check if we already have the header in cache.
	if header, exists := t.headers.Get(hash); exists {
		return header, nil
	}

	newHeader, err := t.client.HeaderByHash(t.ctx, hash)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s header by hash", hash)
	}
	if !newHeader.Number.IsUint64() {
		return nil, fmt.Errorf("received unexpected block number in Tracker: %v", newHeader.Number)
	}

	// Store the header in cache.
	t.headers.Set(newHeader.Hash(), newHeader)

	return newHeader, nil
}

func (t *Tracker) syncLatestHead() error {
	newHeader, err := t.headerByNumber(rpc.LatestBlockNumber)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve latest header")
	}

	reorged := newReorgedHeaders()

	seenHeader, exists := t.canonicalChain.Get(newHeader.Number.Uint64())
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
	current := newHeader
	for {
		prevNumber := current.Number.Uint64() - 1
		prevHeader, exists := t.canonicalChain.Get(prevNumber)

		if prevNumber == 0 {
			if current.ParentHash != t.genesis {
				return errors.Errorf("failed to find genesis block in canonical chain")
			}

			// We reached the genesis block. The chain is complete.
			break
		}

		if !exists {
			// There's a gap. We need to fetch the previous header.
			prev, err := t.headerByNumber(rpc.BlockNumber(prevNumber))
			if err != nil {
				return errors.Wrapf(err, "failed to retrieve previous header %d", current.Number.Uint64()-1)
			}

			t.canonicalChain.Set(prev.Number.Uint64(), prev)
			prevHeader = prev
		}

		// Make sure that the headers are connected in a chain.
		if current.ParentHash == prevHeader.Hash() {
			// We already had this header in the canonical chain, this means the chain is complete.
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

		// update the canonical chain with the new chain
		t.canonicalChain.Set(newChainPrev.Number.Uint64(), newChainPrev)

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
	t.canonicalChain.Set(newHeader.Number.Uint64(), newHeader)

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
		headerToNotify, exists := t.canonicalChain.Get(headerToNotifyNumber)
		if !exists {
			// This should never happen since we're making sure that the headers are continuous.
			return errors.Errorf("failed to find header %d in canonical chain", headerToNotifyNumber)
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
				//fmt.Println("reorged min", reorged.minNumber)
				//fmt.Println("reorged max", reorged.maxNumber)
				fmt.Println("sub last sent header", sub.lastSentHeader.Number.Uint64())

				if sub.lastSentHeader.Number.Uint64() < reorged.min().Number.Uint64() {
					// The subscriber is subscribed to a deeper ConfirmationRule than the reorg depth -> this reorg doesn't affect the subscriber.
					reorg = false
				} else {
					// The subscriber is affected by the reorg.
					reorg = true
				}
			}
		}

		if reorg {
			fmt.Println("reorged min", reorged.min().Number.Uint64())
			minReorgedHeader := reorged.min()
			oldChain := t.chain(minReorgedHeader, sub.lastSentHeader, true)

			// TODO: we should store both the reorged and new chain in a structure such as reorgedHeaders
			//   maybe repurpose to headerChain or something similar -> have this for reorged and new chain
			newChainMin, exists := t.canonicalChain.Get(minReorgedHeader.Number.Uint64() - 1) // new chain min -1 because t.chain() excludes start header
			if !exists {
				return errors.Errorf("failed to find header %d in canonical chain", minReorgedHeader.Number.Uint64())
			}
			newChain := t.chain(newChainMin, headerToNotify, false)
			sub.callback(oldChain, newChain)
		} else {
			sub.callback(nil, t.chain(sub.lastSentHeader, headerToNotify, false))
		}
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
		// TODO: implement handling of old chain -> this is concurrent to the canonical chain, so we might need to handle this differently
		//  but: think about the use cases of the safe block: usually it's just about marking the safe head, so there's no need for the old chain.
		//  this could mean that we should have a different type of callback for safe and finalized head.
		sub.callback(nil, t.chain(sub.lastSentHeader, newHeader, false))
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

	// TODO: prune old headers from headers cache and canonical chain

	return nil
}

func (t *Tracker) notifyFinalizedHead(newHeader *types.Header) {
	t.lastFinalizedHeader = newHeader

	t.mu.RLock()
	defer t.mu.RUnlock()

	// Notify all subscribers to new FinalizedChainHead.
	for _, sub := range t.subscriptions[FinalizedChainHead] {
		newChain := t.chain(sub.lastSentHeader, newHeader, false)

		sub.callback(nil, newChain)
		sub.lastSentHeader = newHeader
	}
}

// generates the chain limited by start and end headers. Star may be included or not depending on includeStart
func (t *Tracker) chain(start, end *types.Header, includeStart bool) []*types.Header {
	var chain []*types.Header
	var exists, genesisChain bool
	current := end

	var startHash common.Hash
	if start == nil {
		startHash = t.genesis
		genesisChain = true
	} else {
		startHash = start.Hash()
	}

	if current.Hash() == startHash {
		chain = append(chain, current)
		return chain
	}

	for current.Hash() != startHash {
		chain = append(chain, current)
		parentHash := current.ParentHash
		if genesisChain && parentHash == t.genesis {
			break
		}

		current, exists = t.headers.Get(parentHash)
		if !exists {
			// This should never happen since we're making sure that the headers are continuous.
			panic(fmt.Sprintf("failed to find header %s in cache", parentHash))
		}
	}
	if includeStart && start != nil {
		chain = append(chain, start)
	}

	return chain
}

func (t *Tracker) Subscribe(confirmationRule ConfirmationRule, callback SubscriptionCallback, maxHeadersSent int) (unsubscribe func()) {
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

	sub := newSubscription(t.subscriptionCounter, confirmationRule, callback, maxHeadersSent)

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

func (t *Tracker) pruneOldHeaders() {
	// can prune all headers that are older than last sent header - 2 epochs (reorg deeper than that can't happen)
	t.mu.Lock()
	defer t.mu.Unlock()
	var minNumber *big.Int
	for _, confRule := range []ConfirmationRule{LatestChainHead, SafeChainHead, FinalizedChainHead} {
		for _, sub := range t.subscriptions[confRule] {
			if sub.lastSentHeader == nil { // did not sent anything to this subscriber, so it's impossible to determine no, which headers could be pruned
				return
			}
			if minNumber == nil {
				minNumber = big.NewInt(0).Set(sub.lastSentHeader.Number)
				continue
			}
			if sub.lastSentHeader.Number.Cmp(minNumber) < 0 {
				minNumber.Set(sub.lastSentHeader.Number)
			}
		}
	}
	if minNumber == nil {
		return
	}
	minNumber.Sub(minNumber, big.NewInt(int64(maxConfirmationRule)))

	// prune from canonical chain
	keys := t.canonicalChain.Keys()
	for _, key := range keys {
		if key <= minNumber.Uint64() {
			t.canonicalChain.Delete(key)
		}
	}

	// prune from all headers
	headers := t.headers.Values()
	for _, header := range headers {
		if header.Number.Cmp(minNumber) <= 0 {
			t.headers.Delete(header.Hash())
		}
	}
}

func (t *Tracker) Stop() {
	log.Info("stopping Tracker")
	t.cancel()
	log.Info("Tracker stopped")
}
