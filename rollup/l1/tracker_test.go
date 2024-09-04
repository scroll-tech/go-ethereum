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

	return m.chain[number.Uint64()].Header(), nil
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

func TestTracker(t *testing.T) {
	client := newMockETHClient()
	tracker := NewTracker(context.Background(), client)

	tracker.Subscribe(LatestChainHead, func(last, new *types.Header, reorg bool) {
		fmt.Println("sub 1: new block", new.Number, reorg)
	})

	tracker.Subscribe(3, func(last, new *types.Header, reorg bool) {
		fmt.Println("sub 2: new block", new.Number, reorg)
	})

	tracker.Subscribe(FinalizedChainHead, func(last, new *types.Header, reorg bool) {
		fmt.Println("sub 3 (finalized): new block", new.Number, reorg)
	})

	err := tracker.syncLatestHead()
	require.NoError(t, err)

	fmt.Println("----------------------------------")
	client.setLatestBlock(2)

	err = tracker.syncLatestHead()
	require.NoError(t, err)

	fmt.Println("----------------------------------")
	client.setLatestBlock(3)

	err = tracker.syncLatestHead()
	require.NoError(t, err)

	fmt.Println("----------------------------------")
	client.setFinalizedBlock(1)

	err = tracker.syncFinalizedHead()
	require.NoError(t, err)

	// TODO:
	//  - test invalid confirmation rules
	//  - test reorg
	//  - test multiple subscribers with same confirmation rules and reorg
	//  - test multiple subscribers with different confirmation rules and reorg
	//  - test finalized, safe
	//  - test finalized panic if reorg, safe reorg
	//  - test pruning of headers when finalized header arrives
	//  - test unsubscribe
	//  - test running with Start and RPC errors -> recovering automatically
}
