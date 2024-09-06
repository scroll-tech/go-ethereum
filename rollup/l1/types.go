package l1

import (
	"context"
	"math"
	"math/big"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type Client interface {
	BlockNumber(ctx context.Context) (uint64, error)
	ChainID(ctx context.Context) (*big.Int, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error)
	BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
}

type reorgedHeaders struct {
	headers   map[common.Hash]*types.Header
	maxNumber uint64
	minNumber uint64
}

func newReorgedHeaders() *reorgedHeaders {
	return &reorgedHeaders{
		maxNumber: 0,
		minNumber: math.MaxUint64,
		headers:   make(map[common.Hash]*types.Header),
	}
}

func (r *reorgedHeaders) add(header *types.Header) {
	r.headers[header.Hash()] = header

	if header.Number.Uint64() > r.maxNumber {
		r.maxNumber = header.Number.Uint64()
	}
	if header.Number.Uint64() < r.minNumber {
		r.minNumber = header.Number.Uint64()
	}
}

func (r *reorgedHeaders) contains(hash common.Hash) bool {
	_, exists := r.headers[hash]
	return exists
}

func (r *reorgedHeaders) isEmpty() bool {
	return len(r.headers) == 0
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
