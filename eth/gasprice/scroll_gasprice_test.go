// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package gasprice

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/scroll-tech/go-ethereum/trie"
)

const (
	blockGasLimit = params.TxGas * 3
)

type testTxData struct {
	priorityFee int64
	gasLimit    uint64
}

type opTestBackend struct {
	block    *types.Block
	receipts []*types.Receipt
}

func (b *opTestBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	panic("not implemented")
}

func (b *opTestBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	return b.block, nil
}

func (b *opTestBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.receipts, nil
}

func (b *opTestBackend) Pending() (*types.Block, types.Receipts, *state.StateDB) {
	panic("not implemented")
}

func (b *opTestBackend) ChainConfig() *params.ChainConfig {
	return b.chain.Config()
}

func (b *opTestBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return nil
}

var _ OracleBackend = (*opTestBackend)(nil)

func newOpTestBackend(t *testing.T, txs []testTxData) *opTestBackend {
	var (
		key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		signer = types.LatestSigner(params.TestChainConfig)
	)
	// only the most recent block is considered for optimism priority fee suggestions, so this is
	// where we add the test transactions
	ts := []*types.Transaction{}
	rs := []*types.Receipt{}
	header := types.Header{}
	header.GasLimit = blockGasLimit
	var nonce uint64
	for _, tx := range txs {
		txdata := &types.DynamicFeeTx{
			ChainID:   params.TestChainConfig.ChainID,
			Nonce:     nonce,
			To:        &common.Address{},
			Gas:       params.TxGas,
			GasFeeCap: big.NewInt(100 * params.GWei),
			GasTipCap: big.NewInt(tx.priorityFee),
			Data:      []byte{},
		}
		t := types.MustSignNewTx(key, signer, txdata)
		ts = append(ts, t)
		r := types.Receipt{}
		r.GasUsed = tx.gasLimit
		header.GasUsed += r.GasUsed
		rs = append(rs, &r)
		nonce++
	}
	hasher := trie.NewStackTrie(nil)
	b := types.NewBlock(&header, &types.Body{Transactions: ts}, nil, hasher, types.DefaultBlockConfig)
	return &opTestBackend{block: b, receipts: rs}
}

func TestSuggestOptimismPriorityFee(t *testing.T) {
	minSuggestion := new(big.Int).SetUint64(1e8 * params.Wei)
	cases := []struct {
		txdata []testTxData
		want   *big.Int
	}{
		{
			// block well under capacity, expect min priority fee suggestion
			txdata: []testTxData{{params.GWei, 21000}},
			want:   minSuggestion,
		},
		{
			// 2 txs, still under capacity, expect min priority fee suggestion
			txdata: []testTxData{{params.GWei, 21000}, {params.GWei, 21000}},
			want:   minSuggestion,
		},
		{
			// 2 txs w same priority fee (1 gwei), but second tx puts it right over capacity
			txdata: []testTxData{{params.GWei, 21000}, {params.GWei, 21001}},
			want:   big.NewInt(1100000000), // 10 percent over 1 gwei, the median
		},
		{
			// 3 txs, full block. return 10% over the median tx (10 gwei * 10% == 11 gwei)
			txdata: []testTxData{{10 * params.GWei, 21000}, {1 * params.GWei, 21000}, {100 * params.GWei, 21000}},
			want:   big.NewInt(11 * params.GWei),
		},
	}
	for i, c := range cases {
		backend := newOpTestBackend(t, c.txdata)
		oracle := NewOracle(backend, Config{MinSuggestedPriorityFee: minSuggestion}, big.NewInt(params.GWei))
		got := oracle.SuggestOptimismPriorityFee(context.Background(), backend.block.Header(), backend.block.Hash())
		if got.Cmp(c.want) != 0 {
			t.Errorf("Gas price mismatch for test case %d: want %d, got %d", i, c.want, got)
		}
	}
}
