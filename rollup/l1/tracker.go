package l1

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
	"github.com/scroll-tech/go-ethereum/rpc"
)

type Tracker struct {
	ctx    context.Context
	cancel context.CancelFunc

	client           sync_service.EthClient
	lastSyncedHeader *types.Header
	subList          headerSubList
	scope            event.SubscriptionScope
}

const (
	// defaultSyncInterval is the frequency at which we query for new chain head
	defaultSyncInterval = 12 * time.Second
)

func NewTracker(ctx context.Context, l1Client sync_service.EthClient) (*Tracker, error) {
	ctx, cancel := context.WithCancel(ctx)
	l1Tracker := &Tracker{
		ctx:    ctx,
		cancel: cancel,

		client: l1Client,
	}
	l1Tracker.Start()
	return l1Tracker, nil
}

func (t *Tracker) headerByDepth(depth ChainDepth, latestNumber *big.Int) (*types.Header, error) {
	var blockNumber *big.Int
	switch depth {
	case LatestBlock:
		blockNumber = big.NewInt(int64(rpc.LatestBlockNumber))
	case SafeBlock:
		blockNumber = big.NewInt(int64(rpc.SafeBlockNumber))
	case FinalizedBlock:
		blockNumber = big.NewInt(int64(rpc.FinalizedBlockNumber))
	default:
		blockNumber = big.NewInt(0).Sub(latestNumber, big.NewInt(int64(depth)))
	}
	header, err := t.client.HeaderByNumber(t.ctx, blockNumber)
	if err != nil {
		return nil, err
	}
	return header, nil
}

func (t *Tracker) newHead(header *types.Header) {
	t.lastSyncedHeader = header
	t.subList.stopSending()
	t.subList.sendNewHeads(t.headerByDepth, header)
}

func (t *Tracker) syncLatestHead() error {
	header, err := t.client.HeaderByNumber(t.ctx, big.NewInt(int64(rpc.LatestBlockNumber)))
	if err != nil {
		return err
	}
	if !header.Number.IsInt64() {
		return fmt.Errorf("received unexpected block number in L1Client: %v", header.Number)
	}
	// sync is continuous
	if t.lastSyncedHeader != nil || header.ParentHash != t.lastSyncedHeader.Hash() {
		t.newHead(header)
	} else { // reorg happened or some blocks were not synced
		// todo: clear cache
		t.newHead(header)
	}
	return nil
}

func (t *Tracker) Subscribe(channel HeaderChan, depth ChainDepth) event.Subscription {
	sub := &headerSub{
		list:    &t.subList,
		depth:   depth,
		channel: channel,
		err:     make(chan error, 1),
	}
	t.subList.add(sub)
	return t.scope.Track(sub)
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
				err := t.syncLatestHead()
				if err != nil {
					log.Warn("Tracker: failed to sync latest head", "err", err)
				}
			}
		}
	}()
}

func (t *Tracker) Stop() {
	log.Info("stopping Tracker")
	t.cancel()
	t.scope.Close()
	log.Info("Tracker stopped")
}

type headerSubList struct {
	mu   sync.Mutex
	list []*headerSub
}

func (l *headerSubList) add(sub *headerSub) {
	l.mu.Lock()
	l.list = append(l.list, sub)
	sort.Slice(l.list, func(i, j int) bool {
		return l.list[i].depth < l.list[j].depth
	})
	l.mu.Unlock()
}

func (l *headerSubList) remove(sub *headerSub) {
	l.mu.Lock()
	index := -1
	for i, subl := range l.list {
		if subl == sub {
			index = i
			break
		}
	}
	if index != -1 {
		l.list = append(l.list[:index], l.list[index+1:]...)
	}
	l.mu.Unlock()
}

func (l *headerSubList) sendNewHeads(fetchHeaderFunc func(ChainDepth, *big.Int) (*types.Header, error), header *types.Header) {
	l.mu.Lock()
	for _, sub := range l.list {
		sub.value, _ = fetchHeaderFunc(sub.depth, header.Number)
	}
	for _, sub := range l.list {
		if sub.value != nil {
			stopChan := make(chan bool)
			sub.stopChan = stopChan
			// start new goroutine to send new head
			// if subscriber will not read from channel until next update, this goroutine will be stopped and new started
			go func(channel HeaderChan, value *types.Header) {
				select {
				case channel <- value:
				case <-stopChan:
				}
			}(sub.channel, sub.value)
		}
	}
	l.mu.Unlock()
}

func (l *headerSubList) stopSending() {
	l.mu.Lock()
	for _, sub := range l.list {
		if sub.stopChan != nil {
			close(sub.stopChan)
			sub.stopChan = nil
		}
	}
	l.mu.Unlock()
}

type headerSub struct {
	list     *headerSubList
	depth    ChainDepth
	channel  HeaderChan
	errOnce  sync.Once
	err      chan error
	value    *types.Header // value to send next time
	stopChan chan bool     // channel used to stop existing goroutine sending new header
}

func (sub *headerSub) Unsubscribe() {
	sub.errOnce.Do(func() {
		sub.list.remove(sub)
		close(sub.err)
	})
}

func (sub *headerSub) Err() <-chan error {
	return sub.err
}

type ChainDepth int64

const (
	SafeBlock      = ChainDepth(-3)
	FinalizedBlock = ChainDepth(-2)
	LatestBlock    = ChainDepth(-1)
)

type HeaderChan chan *types.Header
