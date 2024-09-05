package l1

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/consensus/ethash"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rpc"
)

type mockETHClient struct {
	chain      []*types.Block
	chainHeads map[rpc.BlockNumber]*types.Block
}

func newMockETHClient() *mockETHClient {
	genesis := &core.Genesis{
		Config: params.TestChainConfig,
	}
	_, chain, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), 100, func(i int, gen *core.BlockGen) {})

	return &mockETHClient{
		chain: chain,
		chainHeads: map[rpc.BlockNumber]*types.Block{
			rpc.LatestBlockNumber:    chain[0],
			rpc.FinalizedBlockNumber: chain[0],
			rpc.SafeBlockNumber:      chain[0],
		},
	}
}

func (m mockETHClient) Header(blockNum int) *types.Header {
	return m.chain[blockNum-1].Header()
}

func (m mockETHClient) setLatestBlock(blockNum int) {
	m.chainHeads[rpc.LatestBlockNumber] = m.chain[blockNum-1]
}
func (m mockETHClient) setFinalizedBlock(blockNum int) {
	m.chainHeads[rpc.FinalizedBlockNumber] = m.chain[blockNum-1]
}
func (m mockETHClient) setSafeBlock(blockNum int) {
	m.chainHeads[rpc.SafeBlockNumber] = m.chain[blockNum-1]
}

func (m *mockETHClient) createFork() {

}

func (m mockETHClient) BlockNumber(ctx context.Context) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockETHClient) ChainID(ctx context.Context) (*big.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockETHClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockETHClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if block, ok := m.chainHeads[rpc.BlockNumber(number.Int64())]; ok {
		return block.Header(), nil
	}

	if number.Uint64() >= uint64(len(m.chain)) {
		return nil, fmt.Errorf("block %d not found", number)
	}

	return m.chain[number.Uint64()-1].Header(), nil
}

func (m mockETHClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockETHClient) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	//TODO implement me
	panic("implement me")
}

func (m mockETHClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	//TODO implement me
	panic("implement me")
}

type subscriptionCallTrace struct {
	last  *types.Header
	new   *types.Header
	reorg bool
}

type subscriptionCalls struct {
	alias    string
	actual   []subscriptionCallTrace
	expected []subscriptionCallTrace
}

func newSubscriptionCalls(tracker *Tracker, alias string, rule ConfirmationRule) (*subscriptionCalls, func()) {
	s := &subscriptionCalls{
		alias:    alias,
		actual:   []subscriptionCallTrace{},
		expected: []subscriptionCallTrace{},
	}

	unsubscribe := tracker.Subscribe(rule, func(last, new *types.Header, reorg bool) {
		s.addActual(last, new, reorg)
	})

	return s, unsubscribe
}

func (s *subscriptionCalls) addActual(last, new *types.Header, reorg bool) {
	s.actual = append(s.actual, subscriptionCallTrace{last, new, reorg})
}

func (s *subscriptionCalls) addExpected(last, new *types.Header, reorg bool) {
	s.expected = append(s.expected, subscriptionCallTrace{last, new, reorg})
}

func (s *subscriptionCalls) requireExpectedCalls(t *testing.T) {
	require.Equalf(t, len(s.expected), len(s.actual), "subscription %s has different number of calls", s.alias)
	require.Equalf(t, s.expected, s.actual, "subscription %s does not match", s.alias)
}

type subscriptionCallsList []*subscriptionCalls

func (s *subscriptionCallsList) requireAll(t *testing.T) {
	for _, sub := range *s {
		sub.requireExpectedCalls(t)
	}
}

func TestTracker_HappyCases(t *testing.T) {
	client := newMockETHClient()
	tracker := NewTracker(context.Background(), client)

	// Prepare subscriptions
	var subs subscriptionCallsList
	sub1, _ := newSubscriptionCalls(tracker, "sub1", LatestChainHead)
	sub2, _ := newSubscriptionCalls(tracker, "sub2", 3)
	sub3, _ := newSubscriptionCalls(tracker, "sub3", FinalizedChainHead)
	sub4, _ := newSubscriptionCalls(tracker, "sub4", SafeChainHead)
	sub5, sub5Unsubscribe := newSubscriptionCalls(tracker, "sub5", LatestChainHead)
	subs = append(subs, sub1, sub2, sub3, sub4, sub5)

	// Block 1
	{
		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(nil, client.Header(1), false)
		sub5.addExpected(nil, client.Header(1), false)

		subs.requireAll(t)
	}

	// Block 2
	{
		client.setLatestBlock(2)
		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(client.Header(1), client.Header(2), false)
		sub5.addExpected(client.Header(1), client.Header(2), false)

		subs.requireAll(t)
	}

	// unsubscribe sub5 -> shouldn't get any notifications anymore
	sub5Unsubscribe()

	// Block 3
	{
		client.setLatestBlock(3)
		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(client.Header(2), client.Header(3), false)
		sub2.addExpected(nil, client.Header(1), false)

		subs.requireAll(t)
	}

	// Block 3 again (there's no new chain head) - nothing should happen
	{
		require.NoError(t, tracker.syncLatestHead())
		subs.requireAll(t)
	}

	// Block 70 - we skip a bunch of blocks
	{
		client.setLatestBlock(70)
		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(client.Header(3), client.Header(70), false)
		sub2.addExpected(client.Header(1), client.Header(68), false)

		subs.requireAll(t)
	}

	// Safe block 5
	{
		client.setSafeBlock(5)
		require.NoError(t, tracker.syncSafeHead())

		sub4.addExpected(nil, client.Header(5), false)

		subs.requireAll(t)
	}

	// Finalize block 5
	{
		client.setFinalizedBlock(5)
		require.NoError(t, tracker.syncFinalizedHead())

		sub3.addExpected(nil, client.Header(5), false)

		subs.requireAll(t)
	}

	// Block 72 - we skip again 1 block
	{
		client.setLatestBlock(72)
		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(client.Header(70), client.Header(72), false)
		sub2.addExpected(client.Header(68), client.Header(70), false)

		subs.requireAll(t)
	}

	// Safe block 6
	{
		client.setSafeBlock(6)
		require.NoError(t, tracker.syncSafeHead())

		sub4.addExpected(client.Header(5), client.Header(6), false)

		subs.requireAll(t)
	}

	// Safe block 6 again (there's no new chain head) - nothing should happen
	{
		require.NoError(t, tracker.syncSafeHead())
		subs.requireAll(t)
	}

	// Finalize block 10
	{
		client.setFinalizedBlock(10)
		require.NoError(t, tracker.syncFinalizedHead())

		sub3.addExpected(client.Header(5), client.Header(10), false)

		subs.requireAll(t)
	}

	// Finalize block 10 again (there's no new chain head) - nothing should happen
	{
		require.NoError(t, tracker.syncFinalizedHead())
		subs.requireAll(t)
	}

	// TODO:
	//  - test invalid confirmation rules
	//  - test reorg
	//  - test multiple subscribers with same confirmation rules and reorg
	//  - test multiple subscribers with different confirmation rules and reorg
	//  .- test finalized, safe
	//  - test finalized panic if reorg, safe reorg
	//  - test pruning of headers when finalized header arrives
	//  .- test unsubscribe
	//  - test running with Start and RPC errors -> recovering automatically
}

func TestTracker_Subscribe_ConfirmationRules(t *testing.T) {
	client := newMockETHClient()
	tracker := NewTracker(context.Background(), client)

	// valid rules
	tracker.Subscribe(FinalizedChainHead, func(last, new *types.Header, reorg bool) {})
	tracker.Subscribe(SafeChainHead, func(last, new *types.Header, reorg bool) {})
	tracker.Subscribe(LatestChainHead, func(last, new *types.Header, reorg bool) {})
	tracker.Subscribe(5, func(last, new *types.Header, reorg bool) {})
	tracker.Subscribe(maxConfirmationRule, func(last, new *types.Header, reorg bool) {})

	require.Panics(t, func() {
		tracker.Subscribe(maxConfirmationRule+1, func(last, new *types.Header, reorg bool) {})
	})
	require.Panics(t, func() {
		tracker.Subscribe(0, func(last, new *types.Header, reorg bool) {})
	})
	require.Panics(t, func() {
		tracker.Subscribe(FinalizedChainHead-1, func(last, new *types.Header, reorg bool) {})
	})
}
