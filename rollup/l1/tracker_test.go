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
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rpc"
)

const mockChainLength = 200

type mockETHClient struct {
	chain      []*types.Block
	chainHeads map[rpc.BlockNumber]*types.Block

	forkCount int64

	genesis *core.Genesis
	db      ethdb.Database
}

func newMockETHClient() *mockETHClient {
	genesis := &core.Genesis{
		Config: params.TestChainConfig,
	}
	db, chain, _ := core.GenerateChainWithGenesis(genesis, ethash.NewFaker(), mockChainLength, func(i int, gen *core.BlockGen) {})

	return &mockETHClient{
		chain: chain,
		chainHeads: map[rpc.BlockNumber]*types.Block{
			rpc.LatestBlockNumber:    chain[0],
			rpc.FinalizedBlockNumber: chain[0],
			rpc.SafeBlockNumber:      chain[0],
		},
		genesis:   genesis,
		db:        db,
		forkCount: 1324,
	}
}

func (m *mockETHClient) Header(blockNum int) *types.Header {
	return m.chain[blockNum-1].Header()
}

func (m *mockETHClient) setLatestBlock(blockNum int) {
	m.chainHeads[rpc.LatestBlockNumber] = m.chain[blockNum-1]
}
func (m *mockETHClient) setFinalizedBlock(blockNum int) {
	m.chainHeads[rpc.FinalizedBlockNumber] = m.chain[blockNum-1]
}
func (m *mockETHClient) setSafeBlock(blockNum int) {
	m.chainHeads[rpc.SafeBlockNumber] = m.chain[blockNum-1]
}

func (m *mockETHClient) createFork(blockNum int) {
	forkingPointNumber := blockNum - 1
	forkingPoint := m.chain[forkingPointNumber]

	newChain, _ := core.GenerateChain(m.genesis.Config, forkingPoint, ethash.NewFaker(), m.db, mockChainLength-blockNum, func(i int, gen *core.BlockGen) {
		m.forkCount++
		gen.SetDifficulty(big.NewInt(m.forkCount))
	})

	//for i, block := range newChain {
	//	fmt.Println(i, block.Number(), block.Hash())
	//}
	//fmt.Println("---------------")
	//for i, block := range m.chain[:blockNum] {
	//	fmt.Println(i, block.Number(), block.Hash())
	//}

	m.chain = append(m.chain[:blockNum], newChain...)
	m.chainHeads[rpc.LatestBlockNumber] = forkingPoint
}

func (m *mockETHClient) BlockNumber(ctx context.Context) (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockETHClient) ChainID(ctx context.Context) (*big.Int, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockETHClient) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockETHClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if block, ok := m.chainHeads[rpc.BlockNumber(number.Int64())]; ok {
		return block.Header(), nil
	}

	if number.Uint64() >= uint64(len(m.chain)) {
		return nil, fmt.Errorf("block %d not found", number)
	}

	return m.chain[number.Uint64()-1].Header(), nil
}

func (m *mockETHClient) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	for _, block := range m.chain {
		if block.Hash() == hash {
			return block.Header(), nil
		}
	}

	return nil, fmt.Errorf("block %s not found", hash.String())
}

func (m *mockETHClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockETHClient) TransactionByHash(ctx context.Context, txHash common.Hash) (tx *types.Transaction, isPending bool, err error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockETHClient) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
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
		//if last == nil {
		//	fmt.Println("subscription", alias, "new", new.Number, "reorg", reorg)
		//	return
		//}
		//fmt.Println("subscription", alias, "last", last.Number, "new", new.Number, "reorg", reorg)
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

	for i, expected := range s.expected {
		actual := s.actual[i]
		if expected.last != nil && actual.last != nil {
			require.Equalf(t, expected.last, actual.last, "subscription %s call %d has different last header - expected %d (%s), got %d (%s)", s.alias, i, expected.last.Number, expected.last.Hash().String(), actual.last.Number, actual.last.Hash().String())
		} else {
			require.Equalf(t, expected.last, actual.last, "subscription %s call %d has different last header", s.alias, i)
		}
		require.Equalf(t, expected.new, actual.new, "subscription %s call %d has different new header - expected %d (%s), got %d (%s)", s.alias, i, expected.new.Number, expected.new.Hash().String(), actual.new.Number, actual.new.Hash().String())
		require.Equalf(t, expected.reorg, actual.reorg, "subscription %s call %d has different reorg flag", s.alias, i)
	}
}

type subscriptionCallsList []*subscriptionCalls

func (s *subscriptionCallsList) requireAll(t *testing.T) {
	for _, sub := range *s {
		sub.requireExpectedCalls(t)
	}
}

// TestTracker_HappyCases tests the tracker with various happy scenarios:
//   - subscribing to different confirmation rules (latest, finalized, safe, N blocks)
//   - multiple subscribers for the same chain heads
//   - unsubscribe
//   - RPC delivered an old (or same as previous) block we've already seen -> no notifications
//   - RPC delivered a new block -> notify subscribers accordingly
//   - skipping blocks (RPC delivers a block that is not the next one) -> notify subscribers accordingly
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
	sub6, _ := newSubscriptionCalls(tracker, "sub6", FinalizedChainHead)
	sub7, _ := newSubscriptionCalls(tracker, "sub7", SafeChainHead)
	subs = append(subs, sub1, sub2, sub3, sub4, sub5, sub6, sub7)

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

	// Block 2 (RPC delivered an old block that we've already seen) - nothing should happen
	{
		client.setLatestBlock(2)
		require.NoError(t, tracker.syncLatestHead())
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
		sub7.addExpected(nil, client.Header(5), false)

		subs.requireAll(t)
	}

	// Finalize block 5
	{
		client.setFinalizedBlock(5)
		require.NoError(t, tracker.syncFinalizedHead())

		sub3.addExpected(nil, client.Header(5), false)
		sub6.addExpected(nil, client.Header(5), false)

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
		sub7.addExpected(client.Header(5), client.Header(6), false)

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
		sub6.addExpected(client.Header(5), client.Header(10), false)

		subs.requireAll(t)
	}

	// Finalize block 10 again (there's no new chain head) - nothing should happen
	{
		require.NoError(t, tracker.syncFinalizedHead())
		subs.requireAll(t)
	}
}

// TODO:
//  - test pruning of headers when finalized header arrives
//  - test running with Start and RPC errors -> recovering automatically

// TestTracker_Subscribe_Unsubscribe tests valid and invalid ConfirmationRule.
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

func TestTracker_Safe_Finalized_Reorg(t *testing.T) {
	client := newMockETHClient()
	tracker := NewTracker(context.Background(), client)

	// Prepare subscriptions
	var subs subscriptionCallsList
	sub1, _ := newSubscriptionCalls(tracker, "sub1", FinalizedChainHead)
	sub2, _ := newSubscriptionCalls(tracker, "sub2", FinalizedChainHead)
	sub3, _ := newSubscriptionCalls(tracker, "sub3", SafeChainHead)
	sub4, _ := newSubscriptionCalls(tracker, "sub4", SafeChainHead)
	subs = append(subs, sub1, sub2, sub3, sub4)

	// Block 32 Safe
	{
		client.setSafeBlock(32)

		require.NoError(t, tracker.syncSafeHead())

		sub3.addExpected(nil, client.Header(32), false)
		sub4.addExpected(nil, client.Header(32), false)

		subs.requireAll(t)
	}
	// Block 32 Safe again (no new block)
	{
		require.NoError(t, tracker.syncSafeHead())
		subs.requireAll(t)
	}

	// Block 32 Finalized
	{
		client.setFinalizedBlock(32)

		require.NoError(t, tracker.syncFinalizedHead())

		sub1.addExpected(nil, client.Header(32), false)
		sub2.addExpected(nil, client.Header(32), false)

		subs.requireAll(t)
	}
	// Block 32 Finalized again (no new block)
	{
		require.NoError(t, tracker.syncFinalizedHead())
		subs.requireAll(t)
	}

	// Block 24 Safe (reorg)
	{
		client.setSafeBlock(24)

		require.NoError(t, tracker.syncSafeHead())

		sub3.addExpected(client.Header(32), client.Header(24), true)
		sub4.addExpected(client.Header(32), client.Header(24), true)

		subs.requireAll(t)
	}

	// Block 24 Finalized - faulty RPC node
	{
		client.setFinalizedBlock(24)

		require.Error(t, tracker.syncFinalizedHead())

		subs.requireAll(t)
	}
}

func TestTracker_LatestChainHead_Reorg(t *testing.T) {
	client := newMockETHClient()
	tracker := NewTracker(context.Background(), client)

	// Prepare subscriptions
	var subs subscriptionCallsList
	sub1, _ := newSubscriptionCalls(tracker, "sub1", LatestChainHead)
	sub2, _ := newSubscriptionCalls(tracker, "sub2", 3)
	sub3, _ := newSubscriptionCalls(tracker, "sub3", 3)
	sub4, _ := newSubscriptionCalls(tracker, "sub4", 5)
	subs = append(subs, sub1, sub2, sub3, sub4)

	// Block 1
	{
		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(nil, client.Header(1), false)

		subs.requireAll(t)
	}

	// Block 90 - we skip a bunch of blocks
	{
		client.setLatestBlock(90)
		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(client.Header(1), client.Header(90), false)
		sub2.addExpected(nil, client.Header(88), false)
		sub3.addExpected(nil, client.Header(88), false)
		sub4.addExpected(nil, client.Header(86), false)

		subs.requireAll(t)
	}

	// Block 90 - reorg of depth 1 - only sub1 affected
	beforeReorg90 := client.Header(90)
	beforeReorg88 := client.Header(88)
	beforeReorg86 := client.Header(86)
	{
		client.createFork(89)
		client.setLatestBlock(90)

		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(beforeReorg90, client.Header(90), true)

		subs.requireAll(t)
	}

	// Block 98 - gap - since subs 2-4 were not affected by the reorg they should not be notified about the reorg (form their PoV it's just a gap)
	{
		client.setLatestBlock(98)

		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(client.Header(90), client.Header(98), false)
		sub2.addExpected(beforeReorg88, client.Header(96), false)
		sub3.addExpected(beforeReorg88, client.Header(96), false)
		sub4.addExpected(beforeReorg86, client.Header(94), false)

		subs.requireAll(t)
	}

	// reorg of depth 1 + new block
	beforeReorg98 := client.Header(98)
	{
		client.createFork(97)
		client.setLatestBlock(99)

		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(beforeReorg98, client.Header(99), true)
		sub2.addExpected(client.Header(96), client.Header(97), false)
		sub3.addExpected(client.Header(96), client.Header(97), false)
		sub4.addExpected(client.Header(94), client.Header(95), false)

		subs.requireAll(t)
	}

	// Block 99 - reorg of depth 4, subs 1-3 affected
	beforeReorg99 := client.Header(99)
	beforeReorg97 := client.Header(97)
	{
		client.createFork(95)
		client.setLatestBlock(99)

		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(beforeReorg99, client.Header(99), true)
		sub2.addExpected(beforeReorg97, client.Header(97), true)
		sub3.addExpected(beforeReorg97, client.Header(97), true)

		subs.requireAll(t)
	}

	fmt.Println("==================================")
	return

	// Block 80 - reorg and go back to block 80
	beforeReorg99 = client.Header(99)
	beforeReorg97 = client.Header(97)
	beforeReorg95 := client.Header(95)
	{
		client.createFork(79)
		client.setLatestBlock(80)

		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(beforeReorg99, client.Header(80), true)
		sub2.addExpected(beforeReorg97, client.Header(80), true)
		sub3.addExpected(beforeReorg97, client.Header(80), true)
		sub4.addExpected(beforeReorg95, client.Header(80), true)

		subs.requireAll(t)
	}
	return

	// Deep reorg - chain goes back to genesis
	beforeReorg80 := client.Header(80)
	{
		client.createFork(1)
		client.setLatestBlock(80)

		require.NoError(t, tracker.syncLatestHead())

		sub1.addExpected(beforeReorg80, client.Header(80), true)
		sub2.addExpected(beforeReorg80, client.Header(78), true)
		sub3.addExpected(beforeReorg80, client.Header(78), true)
		sub4.addExpected(beforeReorg80, client.Header(76), true)

		subs.requireAll(t)
	}
}
